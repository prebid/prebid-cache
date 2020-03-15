package endpoints

import (
	"context"
	"encoding/json"
	//"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/sirupsen/logrus"
)

// PutHandler serves "POST /cache" requests.
func NewPutHandler(backend backends.Backend, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	putAnyRequestPool := sync.Pool{New: func() interface{} { return &PutRequest{} }}
	putResponsePool := sync.Pool{New: func() interface{} { return &PutResponse{} }}
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Unmarshall *http.Request into a putResponsePool object known as `put`
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		put := putAnyRequestPool.Get().(*PutRequest)
		defer putAnyRequestPool.Put(put)

		err = json.Unmarshal(body, put)
		if err != nil {
			http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
			return
		}

		if len(put.Puts) > maxNumValues {
			http.Error(w, fmt.Sprintf("More keys than allowed: %d", maxNumValues), http.StatusBadRequest)
			return
		}

		// Get a response object from the resource pool that we'll fill with processed info
		resps := putResponsePool.Get().(*PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		resps.toCacheStrings = make([]string, len(put.Puts))
		defer putResponsePool.Put(resps)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		validateIndexChannel := make(chan int)
		backGetIndexChannel := make(chan int)
		backPutIndexChannel := make(chan int)
		exitSignalChannel := make(chan *exitInfo)
		done := make(chan bool)

		go returnIfError(w, exitSignalChannel, done)
		go validateAndEncode(put, resps, validateIndexChannel, exitSignalChannel, backGetIndexChannel)
		go callBackendGet(put, resps, backend, ctx, allowKeys, backGetIndexChannel, backPutIndexChannel)
		go callBackendPut(backend, put, resps, ctx, backPutIndexChannel, exitSignalChannel)

		// Start adding orders to be processed
		for i, _ := range put.Puts {
			validateIndexChannel <- i
		}
		close(validateIndexChannel)

		for i := 0; i < len(put.Puts); i++ {
			<-done
		}

		bytes, err := json.Marshal(resps)
		if err != nil {
			http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

//REFACTOR NewPutHandler(...)
//make this and the next function into one, called `validateAndEncode`, or something. If we go for channels,
//make this channel communicate with the call channel or the put channel or a `terminate` or `done` channel
//in case of an error
func validateAndEncode(p *PutRequest, resp *PutResponse, indexChan <-chan int, exitChan chan<- *exitInfo, backGetIndex chan<- int) {
	index := <-indexChan
	var toCache string

	if len(p.Puts[index].Value) == 0 {
		exit := &exitInfo{
			errMsg: "Missing value.",
			status: http.StatusBadRequest,
		}
		exitChan <- exit
		return
	}
	if p.Puts[index].TTLSeconds < 0 {
		exit := &exitInfo{
			errMsg: fmt.Sprintf("request.puts[%d].ttlseconds must not be negative.", p.Puts[index].TTLSeconds),
			status: http.StatusBadRequest,
		}
		exitChan <- exit
		return
	}

	if p.Puts[index].Type == backends.XML_PREFIX {
		if p.Puts[index].Value[0] != byte('"') || p.Puts[index].Value[len(p.Puts[index].Value)-1] != byte('"') {
			exit := &exitInfo{
				errMsg: fmt.Sprintf("XML messages must have a String value. Found %v", p.Puts[index].Value),
				status: http.StatusBadRequest,
			}
			exitChan <- exit
			return
		}

		// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
		// for example... so we'll need to un-escape it before we consider it to be XML content.
		var interpreted string
		json.Unmarshal(p.Puts[index].Value, &interpreted)
		toCache = p.Puts[index].Type + interpreted
	} else if p.Puts[index].Type == backends.JSON_PREFIX {
		toCache = p.Puts[index].Type + string(p.Puts[index].Value)
	} else {
		exit := &exitInfo{
			errMsg: fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Puts[index].Type),
			status: http.StatusBadRequest,
		}
		exitChan <- exit
		return
	}
	logrus.Debugf("Storing value: %s", toCache)
	u2, err := uuid.NewV4()
	if err != nil {
		exit := &exitInfo{
			errMsg: "Error generating version 4 UUID",
			status: http.StatusInternalServerError,
		}
		exitChan <- exit
		return
	}
	resp.Responses[index].UUID = u2.String()
	resp.toCacheStrings[index] = toCache
	backGetIndex <- index
	return
}

// Definitely this should be its own function, pull the valid `p` object from the pool, since `resps` is
// also a pool object, store its resps.Responses[i].UUID value and put them both back into the pool
// or channel, whichever approach we seem to be the best
func callBackendGet(p *PutRequest, resp *PutResponse, backend backends.Backend, ctx context.Context, allowKeys bool, backGetIndex <-chan int, backPutIndex chan<- int) {
	index := <-backGetIndex
	if allowKeys && len(p.Puts[index].Key) > 0 {
		s, err := backend.Get(ctx, p.Puts[index].Key)
		if err != nil || len(s) == 0 {
			resp.Responses[index].UUID = p.Puts[index].Key
		} else {
			resp.Responses[index].UUID = ""
		}
	}
	backPutIndex <- index
}

func callBackendPut(backend backends.Backend, p *PutRequest, resp *PutResponse, ctx context.Context, backPutIndex <-chan int, exitChan chan<- *exitInfo) {
	index := <-backPutIndex
	if len(resp.Responses[index].UUID) > 0 {
		err := backend.Put(ctx, resp.Responses[index].UUID, resp.toCacheStrings[index], p.Puts[index].TTLSeconds)
		if err != nil {
			if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
				exit := &exitInfo{
					errMsg: fmt.Sprintf("POST /cache element exceeded max size: %v", err),
					status: http.StatusBadRequest,
				}
				exitChan <- exit
				return
			}

			logrus.Error("POST /cache Error while writing to the backend: ", err)
			switch err {
			case context.DeadlineExceeded:
				logrus.Error("POST /cache timed out:", err)
				exit := &exitInfo{
					errMsg: "Timeout writing value to the backend",
					status: HttpDependencyTimeout,
				}
				exitChan <- exit
				return
			default:
				logrus.Error("POST /cache had an unexpected error:", err)
				exit := &exitInfo{
					errMsg: err.Error(),
					status: http.StatusInternalServerError,
				}
				exitChan <- exit
				return
			}
		}
	}
	exit := &exitInfo{
		errMsg: "",
		status: http.StatusOK,
	}
	exitChan <- exit

	return
}

//PARALLELISM
func returnIfError(w http.ResponseWriter, exitChan <-chan *exitInfo, done chan<- bool) {
	exit := <-exitChan
	//fmt.Sprintf(":: returnIfError, exitChan came with: %v \n", exit)
	if exit.status != http.StatusOK {
		http.Error(w, exit.errMsg, exit.status)
	}
	done <- true
}

//Other structs
type PutRequest struct {
	Puts []PutObject `json:"puts"`
}

type PutObject struct {
	Type       string          `json:"type"`
	TTLSeconds int             `json:"ttlseconds"`
	Value      json.RawMessage `json:"value"`
	Key        string          `json:"key"`
}

type PutResponseObject struct {
	UUID string `json:"uuid"`
}

type PutResponse struct {
	Responses      []PutResponseObject `json:"responses"`
	toCacheStrings []string
}

type exitInfo struct {
	errMsg string
	status int
}

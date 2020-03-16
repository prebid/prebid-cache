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

		getIndexChannel := make(chan int)
		putIndexChannel := make(chan int)
		done := make(chan bool)
		var exitData *exitInfo = &exitInfo{errMsg: "", status: http.StatusOK}

		// Make sure all of our requests come error free, return error if not
		validateAndEncode(put, resps, exitData)
		if exitData.status != http.StatusOK {
			http.Error(w, exitData.errMsg, exitData.status)
			return
		}

		// For every validated entry in the request, run `backend.Get()` in parallel, if needed
		if allowKeys {
			go func() {
				for i, _ := range put.Puts {
					getIndexChannel <- i
				}
				close(getIndexChannel)
			}()

			// Take out the index numbers found in `getIndexChannel` to call the `backend.Get()`
			go callBackendGet(backend, put, resps, ctx, getIndexChannel, done)
			<-done
		}

		// For every validated entry in the request, run `backend.Put()` in parallel, if needed
		go func() {
			for i, _ := range put.Puts {
				putIndexChannel <- i
			}
			close(putIndexChannel)
		}()

		go callBackendPut(backend, put, resps, ctx, exitData, putIndexChannel, done)
		<-done

		// If any errors were found in the `backend.Put()` calls, exit and throw the error
		if exitData.status != http.StatusOK {
			http.Error(w, exitData.errMsg, exitData.status)
			return
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

func validateAndEncode(puts *PutRequest, resp *PutResponse, exit *exitInfo) {
	for index, p := range puts.Puts {
		var toCache string

		if len(p.Value) == 0 {
			exit.errMsg = "Missing value."
			exit.status = http.StatusBadRequest
			return
		}
		if p.TTLSeconds < 0 {
			exit.errMsg = fmt.Sprintf("request.puts[%d].ttlseconds must not be negative.", p.TTLSeconds)
			exit.status = http.StatusBadRequest
			return
		}

		if p.Type == backends.XML_PREFIX {
			if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
				exit.errMsg = fmt.Sprintf("XML messages must have a String value. Found %v", p.Value)
				exit.status = http.StatusBadRequest
				return
			}

			// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
			// for example... so we'll need to un-escape it before we consider it to be XML content.
			var interpreted string
			json.Unmarshal(p.Value, &interpreted)
			toCache = p.Type + interpreted
		} else if p.Type == backends.JSON_PREFIX {
			toCache = p.Type + string(p.Value)
		} else {
			exit.errMsg = fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type)
			exit.status = http.StatusBadRequest
			return
		}
		logrus.Debugf("Storing value: %s", toCache)
		u2, err := uuid.NewV4()
		if err != nil {
			exit.errMsg = "Error generating version 4 UUID"
			exit.status = http.StatusInternalServerError
			return
		}
		resp.Responses[index].UUID = u2.String()
		resp.toCacheStrings[index] = toCache
	}
	return
}

func callBackendGet(backend backends.Backend, p *PutRequest, resp *PutResponse, ctx context.Context, getIndexChannel <-chan int, done chan<- bool) {
	for index := range getIndexChannel {
		if len(p.Puts[index].Key) > 0 {
			s, err := backend.Get(ctx, p.Puts[index].Key)
			if err != nil || len(s) == 0 {
				resp.Responses[index].UUID = p.Puts[index].Key
			} else {
				resp.Responses[index].UUID = ""
			}
		}
	}
	done <- true
}

func callBackendPut(backend backends.Backend, p *PutRequest, resp *PutResponse, ctx context.Context, exit *exitInfo, putIndexChannel <-chan int, done chan<- bool) {
	var skip bool = false
	for index := range putIndexChannel {
		if len(resp.Responses[index].UUID) > 0 && !skip {
			err := backend.Put(ctx, resp.Responses[index].UUID, resp.toCacheStrings[index], p.Puts[index].TTLSeconds)
			if err != nil {
				if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
					exit = &exitInfo{
						errMsg: fmt.Sprintf("POST /cache element exceeded max size: %v", err),
						status: http.StatusBadRequest,
					}
					skip = true
				}

				logrus.Error("POST /cache Error while writing to the backend: ", err)
				switch err {
				case context.DeadlineExceeded:
					logrus.Error("POST /cache timed out:", err)
					exit = &exitInfo{
						errMsg: "Timeout writing value to the backend",
						status: HttpDependencyTimeout,
					}
					skip = true
				default:
					logrus.Error("POST /cache had an unexpected error:", err)
					exit = &exitInfo{
						errMsg: err.Error(),
						status: http.StatusInternalServerError,
					}
					skip = true
				}
			}
		}
	}
	done <- true
}

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

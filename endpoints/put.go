package endpoints

import (
	"context"
	"encoding/json"
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
func NewPutHandler(backendClient backends.Backend, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	putAnyRequestPool := sync.Pool{
		New: func() interface{} {
			return &PutRequest{}
		},
	}

	putResponsePool := sync.Pool{
		New: func() interface{} {
			return &PutResponse{}
		},
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Unmarshall *http.Request into a putResponsePool object
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		backend := &backendCallObject{client: backendClient, allowKeys: allowKeys}

		backend.put = putAnyRequestPool.Get().(*PutRequest)
    backend.put.Puts = make([]PutObject, 0)
		defer putAnyRequestPool.Put(backend.put)

		err = json.Unmarshal(body, backend.put)
		if err != nil {
			http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
			return
		}

		// Get a response object from the resource pool that we'll fill with processed info
		backend.resp = putResponsePool.Get().(*PutResponse)
		backend.resp.Responses = make([]PutResponseObject, len(backend.put.Puts))
		backend.resp.toCacheStrings = make([]string, len(backend.put.Puts))
		defer putResponsePool.Put(backend.resp)

		var cancel func()
		backend.ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		exitData := &exitInfo{errMsg: "", status: http.StatusOK}

		if len(backend.put.Puts) == 0 {
			exitData.errMsg = "No keys were sent in backend request"
			exitData.status = http.StatusBadRequest
		} else if len(backend.put.Puts) > maxNumValues {
			exitData.errMsg = fmt.Sprintf("More keys than allowed: %d", maxNumValues)
			exitData.status = http.StatusBadRequest
		} else { // 1 <= len(backend.put.Puts) < maxNumValues
			// Processonly if all requests come error free
			validateAndEncode(backend, exitData)
			if exitData.status == http.StatusOK {
				if len(backend.put.Puts) == 1 {
					callBackendGet(backend, 0)
					callBackendPut(backend, exitData, 0)
				} else {
					// Run process in parallel
					var waitGroup sync.WaitGroup
					waitGroup.Add(len(backend.put.Puts))

					for i, _ := range backend.put.Puts {
						go callBackendInParallel(backend, exitData, i, &waitGroup)
					}

					waitGroup.Wait()
				}
			}
		}

		if exitData.status != http.StatusOK {
			http.Error(w, exitData.errMsg, exitData.status)
			return
		}

		bytes, err := json.Marshal(backend.resp)
		if err != nil {
			http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}

func validateAndEncode(backend *backendCallObject, exit *exitInfo) {
	for index, p := range backend.put.Puts {
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
		backend.resp.Responses[index].UUID = u2.String()
		backend.resp.toCacheStrings[index] = toCache
	}
	return
}

type backendCallObject struct {
	client    backends.Backend
	put       *PutRequest
	resp      *PutResponse
	allowKeys bool
	ctx       context.Context
}

func callBackendInParallel(backend *backendCallObject, exit *exitInfo, index int, wg *sync.WaitGroup) {
	defer wg.Done()
	callBackendGet(backend, index)
	callBackendPut(backend, exit, index)
}

func callBackendGet(backend *backendCallObject, index int) {
	if backend.allowKeys && len(backend.put.Puts[index].Key) > 0 {
		s, err := backend.client.Get(backend.ctx, backend.put.Puts[index].Key)
		if err != nil || len(s) == 0 {
			backend.resp.Responses[index].UUID = backend.put.Puts[index].Key
		} else {
			backend.resp.Responses[index].UUID = ""
		}
	}
}

func callBackendPut(backend *backendCallObject, exit *exitInfo, index int) {
	if len(backend.resp.Responses[index].UUID) > 0 {
		err := backend.client.Put(backend.ctx, backend.resp.Responses[index].UUID, backend.resp.toCacheStrings[index], backend.put.Puts[index].TTLSeconds)
		if err != nil {
			if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
				exit.errMsg = fmt.Sprintf("POST /cache element exceeded max size: %v", err)
				exit.status = http.StatusBadRequest
				return
			}

			logrus.Error("POST /cache Error while writing to the backend: ", err)
			switch err {
			case context.DeadlineExceeded:
				logrus.Error("POST /cache timed out:", err)
				exit.errMsg = "Timeout writing value to the backend"
				exit.status = HttpDependencyTimeout
				return
			default:
				logrus.Error("POST /cache had an unexpected error:", err)
				exit.errMsg = err.Error()
				exit.status = http.StatusInternalServerError
				return
			}
		}
	}
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

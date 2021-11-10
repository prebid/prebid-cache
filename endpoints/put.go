package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/utils"
	"github.com/sirupsen/logrus"
)

// PutHandler serves "POST /cache" requests.
type PutHandler struct {
	backend backends.Backend
	cfg     putHandlerConfig
	memory  syncPools
	metrics *metrics.Metrics
}

type putHandlerConfig struct {
	maxNumValues int
	allowKeys    bool
}

type syncPools struct {
	requestPool     sync.Pool
	putResponsePool sync.Pool
}

func NewPutHandler(storage backends.Backend, metrics *metrics.Metrics, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	putHandler := &PutHandler{}

	// Assign storage client to put endpoint
	putHandler.backend = storage

	// pass metrics engine
	putHandler.metrics = metrics

	// Pass configuration values
	putHandler.cfg = putHandlerConfig{
		maxNumValues: maxNumValues,
		allowKeys:    allowKeys,
	}

	// Instantiate thread-safe memory pools
	putHandler.memory = syncPools{
		requestPool: sync.Pool{
			New: func() interface{} {
				return &PutRequest{}
			},
		},
		putResponsePool: sync.Pool{
			New: func() interface{} {
				return &PutResponse{}
			},
		},
	}

	return putHandler.handle
}

// parseRequest unmarshals the incoming put request into a thread-safe memory pool
// If the incoming request could not be unmarshalled or if the request comes with more
// elements to put than the maximum allowed in Prebid Cache's configuration, the
// corresponding error is returned
func (e *PutHandler) parseRequest(r *http.Request) (*PutRequest, error) {
	if r == nil {
		return nil, utils.PutBadRequestError{}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, utils.PutBadRequestError{}
	}
	defer r.Body.Close()

	// Allocate a PutRequest object in thread-safe memory
	put := e.memory.requestPool.Get().(*PutRequest)
	put.Puts = make([]PutObject, 0)

	if err := json.Unmarshal(body, put); err != nil {
		// place memory back in sync pool
		e.memory.requestPool.Put(put)
		return nil, utils.PutBadRequestError{body}
	}

	if len(put.Puts) > e.cfg.maxNumValues {
		// place memory back in sync pool
		e.memory.requestPool.Put(put)
		return nil, utils.PutMaxNumValuesError{len(put.Puts), e.cfg.maxNumValues}
	}

	return put, nil
}

// parsePutObject returns an error if the PutObject comes with an invalid field
// and formats the string according to its type:
//   - XML content gets unmarshaled in order to un-escape it and then gets
//     prepended by its type
//   - JSON content gets prepended by its type
// No other formats are supported.
func parsePutObject(p PutObject) (string, error) {
	var toCache string

	// Make sure there's data to store
	if len(p.Value) == 0 {
		return "", errors.New("Missing required field value.")
	}

	// Make sure a non-negative time-to-live quantity was provided
	if p.TTLSeconds < 0 {
		return "", fmt.Errorf("ttlseconds must not be negative %d.", p.TTLSeconds)
	}

	// Limit the type of data to XML or JSON
	if p.Type == backends.XML_PREFIX {
		if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
			return "", fmt.Errorf("XML messages must have a String value. Found %v", p.Value)
		}

		// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
		// for example... so we'll need to un-escape it before we consider it to be XML content.
		var interpreted string
		if err := json.Unmarshal(p.Value, &interpreted); err != nil {
			return "", fmt.Errorf("Error unmarshalling XML value: %v", p.Value)
		}

		toCache = p.Type + interpreted
	} else if p.Type == backends.JSON_PREFIX {
		toCache = p.Type + string(p.Value)
	} else {
		return "", fmt.Errorf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type)
	}

	return toCache, nil
}

func logBackendError(err error, index int) (error, int) {
	if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
		return utils.PutBadPayloadSizeError{err.Error(), index}, http.StatusBadRequest
	}

	logrus.Error("POST /cache Error while writing to the backend: ", err)
	switch err {
	case context.DeadlineExceeded:
		logrus.Error("POST /cache timed out:", err)
		return utils.PutDeadlineExceededError{}, utils.HttpDependencyTimeout
	default:
		logrus.Error("POST /cache had an unexpected error:", err)
		return utils.PutInternalServerError{err.Error()}, http.StatusInternalServerError
	}
	return nil, 0
}

// handle is the handler function that gets assigned to the POST method of the `/cache` endpoint
func (e *PutHandler) handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	e.metrics.RecordPutTotal()

	start := time.Now()

	if bytes, err := e.processPutRequest(r); err == nil {
		// successfully stored all elements storage service or database, write http response
		// and record duration metrics
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		e.metrics.RecordPutDuration(time.Since(start))
	} else {
		// At least one of the elements in the incomming request could not be stored
		// write the http error and log corresponding metrics
		http.Error(w, err.Error(), err.StatusCode())

		if err.StatusCode() >= 400 && err.StatusCode() < 500 {
			e.metrics.RecordPutBadRequest()
		} else {
			e.metrics.RecordPutError()
		}
	}
}

// processPutRequest parses, unmarshals, and validates the incomming request; then calls the backend Put()
// implementation on every element of the "puts" array. This function exits after all elements in the
// "puts" array have been stored in the backend, or after the first error is found
func (e *PutHandler) processPutRequest(r *http.Request) ([]byte, *utils.PrebidCacheError) {
	// Parse and validate incomming request
	put, err := e.parseRequest(r)
	if err != nil {
		return nil, utils.NewPrebidCacheError(err, http.StatusBadRequest)
	}
	defer e.memory.requestPool.Put(put)

	// Allocate a PutResponse object in thread-safe memory
	resps := e.memory.putResponsePool.Get().(*PutResponse)
	resps.Responses = make([]PutResponseObject, len(put.Puts))
	defer e.memory.putResponsePool.Put(resps)

	// Send elements to storage service or database
	if pcErr := e.putElements(put, resps); pcErr != nil {
		return nil, pcErr
	}

	// Marshal Prebid Cache's response
	bytes, err := json.Marshal(resps)
	if err != nil {
		return nil, utils.NewPrebidCacheError(errors.New("Failed to serialize UUIDs into JSON."), http.StatusInternalServerError)
	}

	return bytes, nil
}

// putElements calls the backend storage Put() implementation for every element in put.Puts array and stores the
// corresponding UUID's inside the corresponding PutResponse objects. If an error is found, exits even if the
// rest of the put elements have not been stored.
// TODO: Rewrite this function to operate in parallel
// TODO: For those storage clients that support storing multiple elements in a single call, build a batch and send them together
// TODO: store errors in resps and allow Prebid Cache to provide error details in an "errors" field in the response
func (e *PutHandler) putElements(put *PutRequest, resps *PutResponse) *utils.PrebidCacheError {
	for i, p := range put.Puts {
		toCache, err := parsePutObject(p)
		if err != nil {
			return utils.NewPrebidCacheError(err, http.StatusBadRequest)
		}

		// Only allow setting a provided key if configured (and ensure a key is provided).
		if e.cfg.allowKeys && len(p.Key) > 0 {
			resps.Responses[i].UUID = p.Key
			e.metrics.RecordPutKeyProvided()
		} else if resps.Responses[i].UUID, err = utils.GenerateRandomId(); err != nil {
			return utils.NewPrebidCacheError(errors.New("Error generating version 4 UUID"), http.StatusInternalServerError)
		}

		// If we have a blank UUID, don't store anything.
		// Eventually we may want to provide error details, but as of today this is the only non-fatal error
		// Future error details could go into a second property of the Responses object, such as "errors"
		if len(resps.Responses[i].UUID) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			err = e.backend.Put(ctx, resps.Responses[i].UUID, toCache, p.TTLSeconds)
			if err != nil {
				if _, ok := err.(utils.RecordExistsError); ok {
					// Record didn't get overwritten, return a reponse with an empty UUID string
					resps.Responses[i].UUID = ""
				} else {
					err, code := logBackendError(err, i)
					return utils.NewPrebidCacheError(err, code)
				}
			}
			logrus.Tracef("PUT /cache uuid=%s", resps.Responses[i].UUID)
		}
	}
	return nil
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
	//Error string `json:"error,omitempty"`
}

type PutResponse struct {
	Responses []PutResponseObject `json:"responses"`
}

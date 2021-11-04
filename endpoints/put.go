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

func NewPutHandler(backend backends.Backend, metrics *metrics.Metrics, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	putHandler := &PutHandler{
		backend: backend,
		cfg: putHandlerConfig{
			maxNumValues: maxNumValues,
			allowKeys:    allowKeys,
		},
		memory: syncPools{
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
		},
		metrics: metrics,
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

func formatPutError(err error, index int) (error, int) {
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

func (e *PutHandler) handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	e.metrics.RecordPutTotal()

	start := time.Now()

	err := e.doPut(w, r, ps)

	// If the calling function never calls WriterHeader explicitly, Go auto-fills it with a 200
	if err.Code() == 0 || err.Code() >= 200 && err.Code() < 300 {
		e.metrics.RecordPutDuration(time.Since(start))
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	} else {
		if err.Code() >= 400 && err.Code() < 500 {
			e.metrics.RecordPutBadRequest()
		} else {
			e.metrics.RecordPutError()
		}
		http.Error(w, err.Error(), err.Code())
	}
}

func (e *PutHandler) doPut(w http.ResponseWriter, r *http.Request, ps httprouter.Params) []error {
	put, err := e.parseRequest(r)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusBadRequest)
		return []error(utils.NewPrebidCacheError(err, http.StatusBadRequest))
	}
	defer e.memory.requestPool.Put(put)

	// Allocate a PutResponse object in thread-safe memory
	resps := e.memory.putResponsePool.Get().(*PutResponse)
	resps.Responses = make([]PutResponseObject, len(put.Puts))
	defer e.memory.putResponsePool.Put(resps)

	var errs []error
	for i, p := range put.Puts {
		toCache, err := parsePutObject(p)
		if err != nil {
			//http.Error(w, err.Error(), http.StatusBadRequest)
			errs := append(errs, utils.NewPrebidCacheError(err, http.StatusBadRequest))
			continue
		}

		// Only allow setting a provided key if configured (and ensure a key is provided).
		if e.cfg.allowKeys && len(p.Key) > 0 {
			resps.Responses[i].UUID = p.Key
			// Record put that comes with a Key
		} else if resps.Responses[i].UUID, err = utils.GenerateRandomId(); err != nil {
			errs := append(errs, utils.NewPrebidCacheError(errors.New("Error generating version 4 UUID"), http.StatusInternalServerError))
			//http.Error(w, fmt.Sprintf("Error generating version 4 UUID"), http.StatusInternalServerError)
			// If we have a blank UUID, don't store anything.
			// Eventually we may want to provide error details, but as of today this is the only non-fatal error
			// Future error details could go into a second property of the Responses object, such as "errors"
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		err = e.backend.Put(ctx, resps.Responses[i].UUID, toCache, p.TTLSeconds)
		if err != nil {
			err, code := formatPutError(err, i)
			return []error(utils.NewPrebidCacheError(err, code))
			//http.Error(w, err.Error(), code)
			//return
		}
		logrus.Tracef("PUT /cache uuid=%s", resps.Responses[i].UUID)
	}

	bytes, err := json.Marshal(resps)
	if err != nil {
		return []error(utils.NewPrebidCacheError(errors.New("Failed to serialize UUIDs into JSON."), http.StatusInternalServerError))
		//http.Error(w, errors.New("Failed to serialize UUIDs into JSON."), http.StatusInternalServerError)
		//return
	}

	/* Handles POST */
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
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

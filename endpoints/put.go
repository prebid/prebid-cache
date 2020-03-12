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

		resps := putResponsePool.Get().(*PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer putResponsePool.Put(resps)

		for i, p := range put.Puts {
			//Actions 1 & 2: Validate `p`, create and assign UUID to `resps`, and return toCache
			toCache, err, status := validateAndEncode(&p, &resps.Responses[i])
			if err != nil {
				http.Error(w, err.Error(), status)
				return
			}

			//Action 3: Call Get only if `allowKeys` is true
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			// Only allow setting a provided key if configured (and ensure a key is provided).
			if allowKeys {
				callBackendGet(&p, resps.Responses[i], backend, ctx)
			}

			//Action 4: Call Put
			// If we have a blank UUID, don't store anything.
			// Eventually we may want to provide error details, but as of today this is the only non-fatal error
			// Future error details could go into a second property of the Responses object, such as "errors"
			if err, status := callBackendPut(backend, &p, resps.Responses[i], ctx, toCache); err != nil {
				http.Error(w, err.Error(), status)
				return
			}
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

//make this and the next function into one, called `validateAndEncode`, or something. If we go for channels,
//make this channel communicate with the call channel or the put channel or a `terminate` or `done` channel
//in case of an error
func validateAndEncode(p *PutObject, resp *PutResponseObject) (string, error, int) { //make this and the next function
	var toCache string

	if len(p.Value) == 0 {
		return toCache, errors.New("Missing value."), http.StatusBadRequest
	}
	if p.TTLSeconds < 0 {
		return toCache, errors.New(fmt.Sprintf("request.puts[%d].ttlseconds must not be negative.", p.TTLSeconds)), http.StatusBadRequest
	}

	if p.Type == backends.XML_PREFIX {
		if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
			return toCache, errors.New(fmt.Sprintf("XML messages must have a String value. Found %v", p.Value)), http.StatusBadRequest
		}

		// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
		// for example... so we'll need to un-escape it before we consider it to be XML content.
		var interpreted string
		json.Unmarshal(p.Value, &interpreted)
		toCache = p.Type + interpreted
	} else if p.Type == backends.JSON_PREFIX {
		toCache = p.Type + string(p.Value)
	} else {
		return toCache, errors.New(fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type)), http.StatusBadRequest
	}
	logrus.Debugf("Storing value: %s", toCache)
	u2, err := uuid.NewV4()
	if err != nil {
		return toCache, errors.New("Error generating version 4 UUID"), http.StatusInternalServerError
	}
	resp.UUID = u2.String()
	return toCache, nil, http.StatusOK
}

// Definitely this should be its own function, pull the valid `p` object from the pool, since `resps` is
// also a pool object, store its resps.Responses[i].UUID value and put them both back into the pool
// or channel, whichever approach we seem to be the best
func callBackendGet(p *PutObject, resp PutResponseObject, backend backends.Backend, ctx context.Context) {
	if len(p.Key) > 0 {
		s, err := backend.Get(ctx, p.Key)
		if err != nil || len(s) == 0 {
			//resps.Responses[i].UUID = p.Key
			resp.UUID = p.Key
		} else {
			resp.UUID = ""
		}
	}
}

func callBackendPut(backend backends.Backend, p *PutObject, resp PutResponseObject, ctx context.Context, toCache string) (error, int) {
	if len(resp.UUID) > 0 {
		err := backend.Put(ctx, resp.UUID, toCache, p.TTLSeconds)
		if err != nil {
			if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
				//http.Error(w, fmt.Sprintf("POST /cache element %d exceeded max size: %v", i, err), http.StatusBadRequest)
				return errors.New(fmt.Sprintf("POST /cache element exceeded max size: %v", err)), http.StatusBadRequest
			}

			logrus.Error("POST /cache Error while writing to the backend: ", err)
			switch err {
			case context.DeadlineExceeded:
				logrus.Error("POST /cache timed out:", err)
				//http.Error(w, "Timeout writing value to the backend", HttpDependencyTimeout)
				return errors.New("Timeout writing value to the backend"), HttpDependencyTimeout
			default:
				logrus.Error("POST /cache had an unexpected error:", err)
				//http.Error(w, err.Error(), http.StatusInternalServerError)
				return err, http.StatusInternalServerError
			}
			//return are those 3 errors mutually excdlusive?
		}
	}
	return nil, http.StatusOK
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
	Responses []PutResponseObject `json:"responses"`
}

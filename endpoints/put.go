package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	uuid "github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
)

// PutHandler serves "POST /cache" requests.
func NewPutHandler(backend backends.Backend, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	// TODO(future PR): Break this giant function apart
	putAnyRequestPool := sync.Pool{
		New: func() interface{} {
			return PutRequest{}
		},
	}

	putResponsePool := sync.Pool{
		New: func() interface{} {
			return PutResponse{}
		},
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		put := putAnyRequestPool.Get().(PutRequest)
		defer putAnyRequestPool.Put(put)

		err = json.Unmarshal(body, &put)
		if err != nil {
			http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
			return
		}

		if len(put.Puts) > maxNumValues {
			http.Error(w, fmt.Sprintf("More keys than allowed: %d", maxNumValues), http.StatusBadRequest)
			return
		}

		resps := putResponsePool.Get().(PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer putResponsePool.Put(resps)

		for i, p := range put.Puts {
			if len(p.Value) == 0 {
				http.Error(w, "Missing value.", http.StatusBadRequest)
				return
			}
			if p.TTLSeconds < 0 {
				http.Error(w, fmt.Sprintf("request.puts[%d].ttlseconds must not be negative.", p.TTLSeconds), http.StatusBadRequest)
				return
			}

			var toCache string
			if p.Type == backends.XML_PREFIX {
				if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
					http.Error(w, fmt.Sprintf("XML messages must have a String value. Found %v", p.Value), http.StatusBadRequest)
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
				http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
				return
			}

			logrus.Debugf("Storing value: %s", toCache)
			u2, err := uuid.NewV4()
			if err != nil {
				http.Error(w, fmt.Sprintf("Error generating version 4 UUID"), http.StatusBadRequest)
			}
			resps.Responses[i].UUID = u2.String()

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			// Only allow setting a provided key if configured (and ensure a key is provided).
			if allowKeys && len(p.Key) > 0 {
				s, err := backend.Get(ctx, p.Key)
				if err != nil || len(s) == 0 {
					resps.Responses[i].UUID = p.Key
				} else {
					resps.Responses[i].UUID = ""
				}
			}
			// If we have a blank UUID, don't store anything.
			// Eventually we may want to provide error details, but as of today this is the only non-fatal error
			// Future error details could go into a second property of the Responses object, such as "errors"
			if len(resps.Responses[i].UUID) > 0 {
				err = backend.Put(ctx, resps.Responses[i].UUID, toCache, p.TTLSeconds)
				if err != nil {
					if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
						http.Error(w, fmt.Sprintf("POST /cache element %d exceeded max size: %v", i, err), http.StatusBadRequest)
						return
					}

					logrus.Error("POST /cache Error while writing to the backend: ", err)
					switch err {
					case context.DeadlineExceeded:
						logrus.Error("POST /cache timed out:", err)
						http.Error(w, "Timeout writing value to the backend", HttpDependencyTimeout)
					default:
						logrus.Error("POST /cache had an unexpected error:", err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
					return
				}
			}

		}

		bytes, err := json.Marshal(&resps)
		if err != nil {
			http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
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
	Responses []PutResponseObject `json:"responses"`
}

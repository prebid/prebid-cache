package endpoints

import (
	"encoding/json"
	"net/http"
	"fmt"
	"time"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"sync"
	"github.com/Prebid-org/prebid-cache/backends"
	"context"
	"github.com/satori/go.uuid"
	"github.com/Sirupsen/logrus"
)

// PutHandler serves "POST /cache" requests.
func NewPutHandler(backend backends.Backend) func(http.ResponseWriter, *http.Request, httprouter.Params) {
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

		if len(put.Puts) > MaxNumValues {
			http.Error(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
			return
		}

		resps := putResponsePool.Get().(PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer putResponsePool.Put(resps)

		for i, p := range put.Puts {
			if len(p.Value) > MaxValueLength {
				http.Error(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
				return
			}

			if len(p.Value) == 0 {
				http.Error(w, "Missing value.", http.StatusBadRequest)
				return
			}

			var toCache string
			if p.Type == XML_PREFIX {
				if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
					http.Error(w, fmt.Sprintf("XML messages must have a String value. Found %v", p.Value), http.StatusBadRequest)
					return
				}

				// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
				// for example... so we'll need to un-escape it before we consider it to be XML content.
				var interpreted string
				json.Unmarshal(p.Value, &interpreted)
				toCache = p.Type + interpreted
			} else if p.Type == JSON_PREFIX {
				toCache = p.Type + string(p.Value)
			} else {
				http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
				return
			}

			logrus.Debugf("Storing value: %s", toCache)
			resps.Responses[i].UUID = uuid.NewV4().String()
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			err = backend.Put(ctx, resps.Responses[i].UUID, toCache)

			if err != nil {
				logrus.Error("POST /cache Error while writing to the backend:", err)
				switch err {
				case context.DeadlineExceeded:
					http.Error(w, "Timeout writing value to the backend", HttpDependencyTimeout)
				default:
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
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
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type PutResponseObject struct {
	UUID string `json:"uuid"`
}

type PutResponse struct {
	Responses []PutResponseObject `json:"responses"`
}
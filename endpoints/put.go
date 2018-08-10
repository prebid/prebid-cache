package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/constant"
	log "github.com/prebid/prebid-cache/logger"
	"github.com/prebid/prebid-cache/stats"
	"github.com/satori/go.uuid"
)

// PutHandler serves "POST /cache" requests.
func NewPutHandler(backend backends.Backend, maxNumValues int) func(http.ResponseWriter, *http.Request, httprouter.Params) {
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
		start := time.Now()
		stats.LogCacheRequestedPutStats()
		logger.Info("POST /cache called")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read the request body.")
			http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		put := putAnyRequestPool.Get().(PutRequest)
		defer putAnyRequestPool.Put(put)

		err = json.Unmarshal(body, &put)
		if err != nil {
			stats.LogCacheFailedPutStats(constant.InvalidJSON)
			http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
			return
		}

		if len(put.Puts) > maxNumValues {
			stats.LogCacheFailedPutStats(constant.KeyCountExceeded)
			http.Error(w, fmt.Sprintf("More keys than allowed: %d", maxNumValues), http.StatusBadRequest)
			return
		}

		resps := putResponsePool.Get().(PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer putResponsePool.Put(resps)

		for i, p := range put.Puts {
			if len(p.Value) == 0 {
				logger.Error("Missing value")
				http.Error(w, "Missing value.", http.StatusBadRequest)
				return
			}

			var toCache string
			if p.Type == backends.XML_PREFIX {
				if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
					logger.Error("XML messages must have a String value. Found %v", p.Value)
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
				logger.Error("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type)
				http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
				return
			}

			logger.Debug("Storing value: %s", toCache)

			resps.Responses[i].UUID = uuid.NewV4().String()
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			backendStartTime := time.Now().Nanosecond()
			err = backend.Put(ctx, resps.Responses[i].UUID, toCache)
			backendEndTime := time.Now().Nanosecond()
			backendDiffTime := (backendEndTime.Sub(backendStartTime)) / 1000000
			logger.Info("Time taken by backend.Put: %v", backendDiffTime)
			if err != nil {

				if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
					stats.LogCacheFailedPutStats(constant.MaxSizeExceeded)
					http.Error(w, fmt.Sprintf("POST /cache element %d exceeded max size: %v", i, err), http.StatusBadRequest)
					return
				}

				logger.Error("POST /cache Error while writing to the backend: ", err)
				switch err {
				case context.DeadlineExceeded:
					stats.LogCacheFailedPutStats(constant.TimedOut)
					logger.Error("POST /cache timed out:", err)
					http.Error(w, "Timeout writing value to the backend", HttpDependencyTimeout)
				default:
					stats.LogCacheFailedPutStats(constant.UnexpErr)
					logger.Error("POST /cache had an unexpected error:", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				end := time.Now().Nanosecond()
				totalTime := (end.Sub(start.Nanosecond())) / 1000000
				logger.Info("Total time for put: %v", totalTime)
				return
			}
			// log info
			bid := make(map[string]interface{})
			var bodyStr string
			json.Unmarshal(p.Value, &bodyStr)
			bodyByte := []byte(bodyStr)
			json.Unmarshal(bodyByte, &bid)
			if bid != nil && bid["ext"] != nil {
				bidExt := bid["ext"].(map[string]interface{})
				pubID := bidExt["pubId"]           // TODO: check key name and type
				platformID := bidExt["platformId"] // TODO: check key name and type
				requestID := bidExt["requestId"]   // TODO: check key name and type
				log.DebugWithRequestID(requestID.(string), "pubId: %s, platformId: %s, UUID: %s, Time: %v, Referer: %s", pubID, platformID, resps.Responses[i].UUID, start.Unix(), r.Referer())
			}

		}

		bytes, err := json.Marshal(&resps)
		if err != nil {
			logger.Error("Failed to serialize UUIDs into JSON.")
			http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		end := time.Now().Nanosecond()
		totalTime := (end.Sub(start.Nanosecond())) / 1000000
		logger.Info("Total time for put: %v", totalTime)
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

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
	"github.com/PubMatic-OpenWrap/prebid-cache/backends"
	backendDecorators "github.com/PubMatic-OpenWrap/prebid-cache/backends/decorators"
	"github.com/PubMatic-OpenWrap/prebid-cache/constant"
	log "github.com/PubMatic-OpenWrap/prebid-cache/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/stats"
	"github.com/PubMatic-OpenWrap/prebid-cache/utils"
	"github.com/julienschmidt/httprouter"
)

func NewPutHandler(backend backends.Backend, maxNumValues int, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	// TODO(future PR): Break this giant function apart
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

		put := putAnyRequestPool.Get().(*PutRequest)
		put.Puts = make([]PutObject, 0)
		defer putAnyRequestPool.Put(put)

		err = json.Unmarshal(body, put)
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

		resps := putResponsePool.Get().(*PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer putResponsePool.Put(resps)

		for i, p := range put.Puts {
			if len(p.Value) == 0 {
				logger.Error("Missing value")
				http.Error(w, "Missing value.", http.StatusBadRequest)
				return
			}

			if p.TTLSeconds < 0 {
				http.Error(w, "Error request ttl seconds value must not be negative.", http.StatusBadRequest)
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

			// Only allow setting a provided key if configured (and ensure a key is provided).
			if allowKeys && len(p.Key) > 0 {
				resps.Responses[i].UUID = p.Key
			} else if resps.Responses[i].UUID, err = utils.GenerateRandomId(); err != nil {
				http.Error(w, fmt.Sprintf("Error generating version 4 UUID"), http.StatusInternalServerError)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			// If we have a blank UUID, don't store anything.
			// Eventually we may want to provide error details, but as of today this is the only non-fatal error
			// Future error details could go into a second property of the Responses object, such as "errors"
			if len(resps.Responses[i].UUID) > 0 {
				backendStartTime := time.Now()
				err = backend.Put(ctx, resps.Responses[i].UUID, toCache, p.TTLSeconds)
				logger.Info("Time taken by backend.Put: %v", time.Now().Sub(backendStartTime))
				if err != nil {
					// If entry already existed for UUID, it shouldn't get overwritten and a RecordExistsError is expected
					if _, ok := err.(utils.RecordExistsError); ok {
						// Record didn't get overwritten, return a reponse with an empty UUID string
						resps.Responses[i].UUID = ""
					} else {
						if _, ok := err.(*backendDecorators.BadPayloadSize); ok {
							stats.LogCacheFailedPutStats(constant.MaxSizeExceeded)
							http.Error(w, fmt.Sprintf("POST /cache element %d exceeded max size: %v", i, err), http.StatusBadRequest)
							return
						}

						logger.Error("POST /cache Error while writing to the backend: %+v", err)
						switch err {
						case context.DeadlineExceeded:
							stats.LogCacheFailedPutStats(constant.TimedOut)
							logger.Error("POST /cache timed out: %+v", err)
							http.Error(w, "Timeout writing value to the backend", HttpDependencyTimeout)
						default:
							stats.LogCacheFailedPutStats(constant.UnexpErr)
							logger.Error("POST /cache had an unexpected error: %+v", err)
							http.Error(w, err.Error(), http.StatusInternalServerError)
						}

						logger.Info("Total time for put: %v", time.Now().Sub(start))
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
				logger.Info("PUT /cache uuid=%s", resps.Responses[i].UUID)
			}
		}

		bytes, err := json.Marshal(resps)
		if err != nil {
			logger.Error("Failed to serialize UUIDs into JSON.")
			http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
		logger.Info("Total time for put: %v", time.Now().Sub(start))
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

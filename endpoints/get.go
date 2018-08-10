package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/constant"
	"github.com/prebid/prebid-cache/stats"
)

func NewGetHandler(backend backends.Backend) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		start := time.Now()
		logger.Info("Get /cache called")
		stats.LogCacheRequestedGetStats()
		id, err := parseUUID(r)
		if err != nil {
			if id == "" {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, err.Error(), http.StatusNotFound)
			}
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		logger.Debug("UUID: %s requested at time: %v, Referer: %s", id, start.Unix(), r.Referer())
		backendStartTime := time.Now()
		value, err := backend.Get(ctx, id)

		backendEndTime := time.Now()
		backendDiffTime := (backendEndTime.Sub(backendStartTime)).Nanoseconds() / 1000000
		logger.Info("Time taken by backend.Get: %v", backendDiffTime)
		if err != nil {
			stats.LogCacheMissStats()
			logger.Info("Cache miss for uuid: %v", id)
			http.Error(w, "No content stored for uuid="+id, http.StatusNotFound)
			end := time.Now()
			totalTime := (end.Sub(start)).Nanoseconds() / 1000000
			logger.Info("Total time for get: %v", totalTime)
			return
		}

		if strings.HasPrefix(value, backends.XML_PREFIX) {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(value)[len(backends.XML_PREFIX):])
		} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(value)[len(backends.JSON_PREFIX):])
		} else {
			http.Error(w, "Cache data was corrupted. Cannot determine type.", http.StatusInternalServerError)
		}
		end := time.Now()
		totalTime := (end.Sub(start)).Nanoseconds() / 1000000
		logger.Info("Total time for get: %v", totalTime)
	}
}

type GetResponse struct {
	Value interface{} `json:"value"`
}

func parseUUID(r *http.Request) (string, error) {
	id := r.URL.Query().Get("uuid")
	var err error = nil
	if id == "" {
		err = errors.New("Missing required parameter uuid")
		stats.LogCacheFailedGetStats(constant.UUIDMissing)
	} else if len(id) != 36 {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		err = fmt.Errorf("No content stored for uuid=%s", id)
		stats.LogCacheFailedGetStats(constant.InvalidUUID)
	}
	return id, err
}

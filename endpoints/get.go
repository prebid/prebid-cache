package endpoints

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

// GetHandler serves "GET /cache" requests.
type GetHandler struct {
	backend         backends.Backend
	metrics         *metrics.Metrics
	allowCustomKeys bool
}

func NewGetHandler(storage backends.Backend, metrics *metrics.Metrics, allowCustomKeys bool) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	getHandler := &GetHandler{
		// Assign storage client to put endpoint
		backend: storage,
		// pass metrics engine
		metrics: metrics,
		// Pass configuration value
		allowCustomKeys: allowCustomKeys,
	}

	// Return handle function
	return getHandler.handle
}

func (e *GetHandler) handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	e.metrics.RecordGetTotal()
	start := time.Now()

	uuid, gerr := parseUUID(r, e.allowCustomKeys)
	if gerr != nil {
		// parseUUID either returns http.StatusBadRequest or http.StatusNotFound. Both should be
		// accounted under the RecordPutBadRequest()
		e.metrics.RecordGetBadRequest()
		outputError(w, gerr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	value, err := e.backend.Get(ctx, uuid)
	if err != nil {
		e.metrics.RecordGetBadRequest()
		outputError(w, utils.NewPrebidCacheGetError(uuid, err, http.StatusNotFound))
		return
	}

	if gerr := writeGetResponse(w, uuid, value); gerr == nil {
		// successfully retrieved value under uuid from the backend storage
		e.metrics.RecordGetDuration(time.Since(start))
	} else {
		e.metrics.RecordGetError()
		outputError(w, gerr)
		return
	}
	return
}

type GetResponse struct {
	Value interface{} `json:"value"`
}

// parseUUID extracts the uuid value from the query and validates its
// lenght in case custom keys are not allowed.
func parseUUID(r *http.Request, allowCustomKeys bool) (string, *utils.PrebidCacheGetError) {
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		return "", utils.NewPrebidCacheGetError("", utils.MissingKeyError{}, http.StatusBadRequest)
	}
	if len(uuid) != 36 && (!allowCustomKeys) {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		return uuid, utils.NewPrebidCacheGetError(uuid, utils.KeyLengthError{}, http.StatusNotFound)
	}
	return uuid, nil
}

func writeGetResponse(w http.ResponseWriter, id string, value string) *utils.PrebidCacheGetError {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(value)[len(backends.XML_PREFIX):])
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value)[len(backends.JSON_PREFIX):])
	} else {
		return utils.NewPrebidCacheGetError(id, errors.New("Cache data was corrupted. Cannot determine type."), http.StatusInternalServerError)
	}
	return nil
}

// outputError will prefix error messages with "GET /cache" and, if uuid string list is passed, will
// follow with the first element of it in the following fashion: "uuid=FIRST_ELEMENT_ON_UUID_PARAM".
// Expects non-nil error
func outputError(w http.ResponseWriter, err *utils.PrebidCacheGetError) {
	logError(err)
	http.Error(w, err.Error(), err.StatusCode())
}

func logError(e *utils.PrebidCacheGetError) {
	if e.IsKeyNotFound() {
		log.Debug(e.Error())
	} else {
		log.Error(e.Error())
	}
}

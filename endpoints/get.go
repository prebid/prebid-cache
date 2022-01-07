package endpoints

import (
	"context"
	"fmt"
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
		// Assign storage client to get endpoint
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

	uuid, parseErr := parseUUID(r, e.allowCustomKeys)
	if parseErr != nil {
		// parseUUID either returns http.StatusBadRequest or http.StatusNotFound. Both should be
		// accounted using RecordGetBadRequest()
		e.metrics.RecordGetBadRequest()
		handleException(w, parseErr, uuid)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	storedData, err := e.backend.Get(ctx, uuid)
	if err != nil {
		e.metrics.RecordGetBadRequest()
		handleException(w, err, uuid)
		return
	}

	if err := writeGetResponse(w, uuid, storedData); err != nil {
		e.metrics.RecordGetError()
		handleException(w, err, uuid)
		return
	}

	// successfully retrieved value under uuid from the backend storage
	e.metrics.RecordGetDuration(time.Since(start))
	return
}

// parseUUID extracts the uuid value from the query and validates its
// lenght in case custom keys are not allowed.
func parseUUID(r *http.Request, allowCustomKeys bool) (string, error) {
	uuid := r.URL.Query().Get("uuid")
	if uuid == "" {
		return "", utils.NewPBCError(utils.MISSING_KEY)
	}
	if len(uuid) != 36 && (!allowCustomKeys) {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		return uuid, utils.NewPBCError(utils.KEY_LENGTH)
	}
	return uuid, nil
}

// writeGetResponse writes the "Content-Type" header and sends back the stored data as a response if
// the sotred data is prefixed by either the "xml" or "json"
func writeGetResponse(w http.ResponseWriter, id string, storedData string) error {
	if strings.HasPrefix(storedData, backends.XML_PREFIX) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(storedData)[len(backends.XML_PREFIX):])
	} else if strings.HasPrefix(storedData, backends.JSON_PREFIX) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(storedData)[len(backends.JSON_PREFIX):])
	} else {
		return utils.NewPBCError(utils.UNKNOWN_STORED_DATA_TYPE)
	}
	return nil
}

// handleException logs and replies to the request with the error message and HTTP code
func handleException(w http.ResponseWriter, err error, uuid string) {
	if err != nil {
		// Build error message
		errMsgBuilder := strings.Builder{}
		errMsgBuilder.WriteString("GET /cache")
		if len(uuid) > 0 {
			errMsgBuilder.WriteString(fmt.Sprintf(" uuid=%s", uuid))
		}
		errMsgBuilder.WriteString(fmt.Sprintf(": %s", err.Error()))
		errMsg := errMsgBuilder.String()

		// Determine the response status code based on error type
		errCode := http.StatusInternalServerError
		isKeyNotFound := false
		if pbcErr, isPBCErr := err.(utils.PBCError); isPBCErr {
			errCode = pbcErr.StatusCode
			isKeyNotFound = pbcErr.Type == utils.KEY_NOT_FOUND
		}

		// Determine lob level
		if isKeyNotFound {
			log.Debug(errMsg)
		} else {
			log.Error(errMsg)
		}

		// Write error response
		http.Error(w, errMsg, errCode)
	}
}

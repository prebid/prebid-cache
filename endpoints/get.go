package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

func NewGetHandler(backend backends.Backend, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err, status := parseUUID(r, allowKeys)
		if err != nil {
			handleException(w, err, status, id)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		value, err := backend.Get(ctx, id)
		if err != nil {
			handleException(w, err, http.StatusNotFound, id)
			return
		}

		if err, status := writeGetResponse(w, id, value); err != nil {
			handleException(w, err, status, id)
			return
		}
		return
	}
}

type GetResponse struct {
	Value interface{} `json:"value"`
}

func parseUUID(r *http.Request, allowKeys bool) (string, error, int) {
	id := r.URL.Query().Get("uuid")
	if id == "" {
		return "", utils.MissingKeyError{}, http.StatusBadRequest
	}
	if len(id) != 36 && (!allowKeys) {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		return id, utils.KeyLengthError{}, http.StatusNotFound
	}
	return id, nil, http.StatusOK
}

func writeGetResponse(w http.ResponseWriter, id string, value string) (error, int) {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(value)[len(backends.XML_PREFIX):])
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value)[len(backends.JSON_PREFIX):])
	} else {
		return errors.New("Cache data was corrupted. Cannot determine type."), http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

// handleException will prefix error messages with "GET /cache" and, if uuid string list is passed, will
// follow with the first element of it in the following fashion: "uuid=FIRST_ELEMENT_ON_UUID_PARAM".
// Expects non-nil error
func handleException(w http.ResponseWriter, err error, status int, uuid string) {

	var msg string
	if len(uuid) > 0 {
		msg = fmt.Sprintf("GET /cache uuid=%s: %s", uuid, err.Error())
	} else {
		msg = fmt.Sprintf("GET /cache: %s", err.Error())
	}

	toLogger(err, msg)

	http.Error(w, msg, status)
}

func toLogger(err error, msg string) {
	if _, isKeyNotFound := err.(utils.KeyNotFoundError); isKeyNotFound {
		log.Debug(msg)
	} else {
		log.Error(msg)
	}
}

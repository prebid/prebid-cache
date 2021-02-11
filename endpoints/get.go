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
	log "github.com/sirupsen/logrus"
)

func NewGetHandler(backend backends.Backend, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err, status := parseUUID(r, allowKeys)
		if err != nil {
			respondAndLogError(w, err, status)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		value, err := backend.Get(ctx, id)
		if err != nil {
			respondAndLogError(w, fmt.Errorf("uuid=%s: %s", id, err.Error()), http.StatusNotFound)
			return
		}

		if err, status := writeGetResponse(w, id, value); err != nil {
			respondAndLogError(w, err, status)
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
		return "", errors.New("Missing required parameter uuid"), http.StatusBadRequest
	}
	if len(id) != 36 && (!allowKeys) {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		return "", fmt.Errorf("uuid=%s: invalid uuid lenght", id), http.StatusNotFound
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
		return fmt.Errorf("uuid=%s: Cache data was corrupted. Cannot determine type.", id), http.StatusInternalServerError
	}
	return nil, http.StatusOK
}

func respondAndLogError(w http.ResponseWriter, err error, status int) {
	log.Errorf(err.Error())
	http.Error(w, err.Error(), status)
}

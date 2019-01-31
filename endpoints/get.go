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
)

func NewGetHandler(backend backends.Backend, allowKeys bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id, err := parseUUID(r, allowKeys)
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
		value, err := backend.Get(ctx, id)

		if err != nil {
			http.Error(w, "No content stored for uuid="+id, http.StatusNotFound)
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
	}
}

type GetResponse struct {
	Value interface{} `json:"value"`
}

func parseUUID(r *http.Request, allowKeys bool) (string, error) {
	id := r.URL.Query().Get("uuid")
	var err error = nil
	if id == "" {
		err = errors.New("Missing required parameter uuid")
	} else if len(id) != 36 && (!allowKeys) {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		err = fmt.Errorf("No content stored for uuid=%s", id)
	}
	return id, err
}

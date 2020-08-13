package endpoints

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

//Default route for the prebid-cache
func NewIndexHandler(emptyIndexResponse bool) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if emptyIndexResponse {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "This application stores short-term data for use in Prebid.")
		}
	}
}

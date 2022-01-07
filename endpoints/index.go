package endpoints

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// NewIndexHandler returns the default '/' route handle function
func NewIndexHandler(message string) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, message)
	}
}

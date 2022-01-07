package endpoints

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// NewIndexHandler is the default '/' route handler
func NewIndexHandler(message string) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, message)
	}
}

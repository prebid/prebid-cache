package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Status is the handler function of the "/status" endpoint
func Status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// We might want more logic here eventually... but for now, we're ok to serve more traffic as
	// long as the server responds.
	w.WriteHeader(http.StatusNoContent)
}

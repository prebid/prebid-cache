package endpoints

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

//Handle Default route for the prebid-cache
func Index(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//Default routing instead of showing 404 not found error
	// Added Default routing handler added message instead of trowing 404 Error
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "This application stores short-term data for use in Prebid.")
}

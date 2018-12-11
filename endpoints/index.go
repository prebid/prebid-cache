package endpoints

import (
	"fmt"
	"net/http"
)

func Index(w http.ResponseWriter, r *http.Request) {
	//Default routing instead of showing 404 not found error
	// Added Default routing handler added message instead of trowing 404 Error
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf("Prebid Cache Index Route.")
}

package utils

import (
	"net/http"
)

// Error types
const (
	MISSING_KEY               = iota // GET http.StatusBadRequest
	RECORD_EXISTS                    // PUT http.StatusBadRequest
	PUT_MAX_NUM_VALUES               // PUT http.StatusBadRequest
	PUT_BAD_REQUEST                  // PUT http.StatusBadRequest
	NEGATIVE_TTL                     // PUT http.StatusBadRequest
	MALFORMED_XML                    // PUT http.StatusBadRequest
	UNSUPPORTED_DATA_TO_STORE        // PUT http.StatusBadRequest
	MISSING_VALUE                    // PUT http.StatusBadRequest
	BAD_PAYLOAD_SIZE                 // PUT http.StatusBadRequest
	UNKNOWN_STORED_DATA_TYPE         // GET http.StatusInternalServerError
	PUT_INTERNAL_SERVER              // PUT http.StatusInternalServerError
	MARSHAL_RESPONSE                 // PUT http.StatusInternalServerError
	KEY_NOT_FOUND                    // GET http.StatusNotFound
	KEY_LENGTH                       // GET http.StatusNotFound
	PUT_DEADLINE_EXCEEDED            // PUT HttpDependencyTimeout
)

// HTTPDependencyTimeout is the status code for errors due to a downstream dependency timeout.
const HTTPDependencyTimeout = 597

// Map Prebid Cache's error codes to their corresponding response status codes
var errToStatusCodes map[int]int = map[int]int{
	MISSING_KEY:               http.StatusBadRequest,
	RECORD_EXISTS:             http.StatusBadRequest,
	PUT_MAX_NUM_VALUES:        http.StatusBadRequest,
	PUT_BAD_REQUEST:           http.StatusBadRequest,
	NEGATIVE_TTL:              http.StatusBadRequest,
	MALFORMED_XML:             http.StatusBadRequest,
	UNSUPPORTED_DATA_TO_STORE: http.StatusBadRequest,
	MISSING_VALUE:             http.StatusBadRequest,
	BAD_PAYLOAD_SIZE:          http.StatusBadRequest,
	UNKNOWN_STORED_DATA_TYPE:  http.StatusInternalServerError,
	PUT_INTERNAL_SERVER:       http.StatusInternalServerError,
	MARSHAL_RESPONSE:          http.StatusInternalServerError,
	KEY_NOT_FOUND:             http.StatusNotFound,
	KEY_LENGTH:                http.StatusNotFound,
	PUT_DEADLINE_EXCEEDED:     HTTPDependencyTimeout,
}

// Map Prebid Cache's error codes to constant error message if they have one. Not all
// error types are here since some of them have non-constant error messages and are assigned upon
// creation
var errToMsgs map[int]string = map[int]string{
	MISSING_KEY:              "missing required parameter uuid",
	RECORD_EXISTS:            "Record exists with provided key.",
	MISSING_VALUE:            "Missing required field value.",
	UNKNOWN_STORED_DATA_TYPE: "Cache data was corrupted. Cannot determine type.",
	MARSHAL_RESPONSE:         "Failed to serialize UUIDs into JSON.",
	KEY_NOT_FOUND:            "Key not found",
	KEY_LENGTH:               "invalid uuid length",
	PUT_DEADLINE_EXCEEDED:    "timeout writing value to the backend.",
}

// PBCError implements the error interface
type PBCError struct {
	Type       int
	StatusCode int
	msg        string
}

func NewPBCError(errType int, msgs ...string) PBCError {
	// Store error's type
	re := PBCError{Type: errType}

	// Assign a return status code
	if statusCode, exists := errToStatusCodes[errType]; exists {
		re.StatusCode = statusCode
	}

	// If custom error message, assign
	for _, msg := range msgs {
		re.msg = re.msg + msg
	}

	return re
}

func (e PBCError) Error() string {
	// If msg field was populated, use it
	if len(e.msg) > 0 {
		return e.msg
	}

	// Find its corresponding error message according to its errType
	if msg, exists := errToMsgs[e.Type]; exists {
		return msg
	}

	// If we couldn't find an error message for this errType and error
	// didn't come with a msg field, return an empty string
	return ""
}

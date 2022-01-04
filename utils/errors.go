package utils

import (
	"fmt"
	"net/http"
	"strings"
)

// Status code for errors due to a downstream dependency timeout.
const HttpDependencyTimeout = 597

/**************************/
/* Get errors			  */
/**************************/
// Key not found
type KeyNotFoundError struct{}

func (e KeyNotFoundError) Error() string {
	return "Key not found"
}

// Missing UUID error
type MissingKeyError struct{}

func (e MissingKeyError) Error() string {
	return "missing required parameter uuid"
}

// Invalid UUID length
type KeyLengthError struct{}

func (e KeyLengthError) Error() string {
	return "invalid uuid length"
}

// UnknownStoredDataType happens when retrieved data is neither XML nof JSON
type UnknownStoredDataType struct{}

func (e UnknownStoredDataType) Error() string {
	return "Cache data was corrupted. Cannot determine type."
}

// Other Prebid Cache error
type UnknownPrebidCacheError struct{}

func (e UnknownPrebidCacheError) Error() string {
	return "Internal Prebid Cache error"
}

/**************************/
/* Put errors			  */
/**************************/
// Put error wrapper
func NewPrebidCacheError(e error, statusCode int) *PrebidCacheError {
	return &PrebidCacheError{e, statusCode}
}

type PrebidCacheError struct {
	err    error
	status int
}

func (ce *PrebidCacheError) Error() string {
	return ce.err.Error()
}

func (ce *PrebidCacheError) StatusCode() int {
	return ce.status
}

// Bad request gets returned when incoming request could not get
// unmarshalled or is invalid JSON
type PutBadRequestError struct {
	Body []byte
}

func (e PutBadRequestError) Error() string {
	return "Request body " + string(e.Body) + " is not valid JSON."
}

//
type MissingValueError struct{}

func (e MissingValueError) Error() string {
	return "Missing required field value."
}

//
type NegativeTTLError struct {
	TTLSeconds int
}

func (e NegativeTTLError) Error() string {
	return fmt.Sprintf("ttlseconds must not be negative %d.", e.TTLSeconds)
}

//
type MalformedXMLError struct {
	Msg string
}

func (e MalformedXMLError) Error() string {
	return e.Msg
}

// Record exists is the error we assign to the different backend errors
// that describe a failure to put a value under a UUID that is already taken
type RecordExistsError struct{}

func (e RecordExistsError) Error() string {
	return "Record exists with provided key."
}

// UnsupportedDataToStoreError happens when Prebid Cache tries to store
// data other than XML of JSON
type UnsupportedDataToStoreError struct {
	UnknownDataType string
}

func (e UnsupportedDataToStoreError) Error() string {
	return fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", e.UnknownDataType)
}

// PutBadPayloadSizeError happens when a payload gets rejected
// for going over the configured max size.
type PutBadPayloadSizeError struct {
	Msg   string
	Index int
}

func (e PutBadPayloadSizeError) Error() string {
	return fmt.Sprintf("POST /cache element %d exceeded max size: %v", e.Index, e.Msg)
}

// PutDeadlineExceededError happens when we get a context.DeadlineExceeded:
type PutDeadlineExceededError struct{}

func (e PutDeadlineExceededError) Error() string {
	return "timeout writing value to the backend."
}

// PutInternalServerError when the backend client returns an error
type PutInternalServerError struct {
	Msg string
}

func (e PutInternalServerError) Error() string {
	return e.Msg
}

// PutMaxNumValuesError happens when an incomming request tries to put more
// values than Prebid Cache has been configured to allow in a single request
type PutMaxNumValuesError struct {
	NumValues    int
	MaxNumValues int
}

func (e PutMaxNumValuesError) Error() string {
	return fmt.Sprintf("trying to put %d keys which is more than the number allowed: %d", e.NumValues, e.MaxNumValues)
}

// MarshalResponseError
type MarshalResponseError struct{}

func (e MarshalResponseError) Error() string {
	return "Failed to serialize UUIDs into JSON."
}

/***************************/
/* Complementary functions */
/***************************/
func GetErrorInfo(err interface{}, method, uuid string) (string, int, bool) {
	errMsgBuilder := strings.Builder{}
	errStatusCode := http.StatusNotFound
	isKeyNotFoundError := false

	// Build error message to log and respond with
	errMsgBuilder.WriteString(method)
	errMsgBuilder.WriteString(" /cache")
	if len(uuid) > 0 {
		errMsgBuilder.WriteString(fmt.Sprintf(" uuid=%s", uuid))
	}
	errMsgBuilder.WriteString(fmt.Sprintf(": %s", err.(error).Error()))

	// Status code to respond with depending of the type of error
	switch err.(type) {
	case KeyNotFoundError:
		// http.StatusNotFound
		isKeyNotFoundError = true
	case KeyLengthError:
		// http.StatusNotFound
	case MissingKeyError:
		errStatusCode = http.StatusBadRequest
	case UnknownStoredDataType:
		errStatusCode = http.StatusInternalServerError
	default:
		errStatusCode = http.StatusInternalServerError
	}

	return errMsgBuilder.String(), errStatusCode, isKeyNotFoundError
}

func PutErrorInfo(err interface{}) int {
	// Status code to respond with depending of the type of error
	switch err.(type) {
	case RecordExistsError:
		return http.StatusBadRequest
	case PutMaxNumValuesError:
		return http.StatusBadRequest
	case PutBadRequestError:
		return http.StatusBadRequest
	case NegativeTTLError:
		return http.StatusBadRequest
	case MalformedXMLError:
		return http.StatusBadRequest
	case UnsupportedDataToStoreError:
		return http.StatusBadRequest
	case MissingValueError:
		return http.StatusBadRequest
	case PutDeadlineExceededError:
		return HttpDependencyTimeout
	case PutInternalServerError:
		return http.StatusInternalServerError
	case MarshalResponseError:
		return http.StatusInternalServerError
	case PutBadPayloadSizeError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

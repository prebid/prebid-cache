package utils

import (
	"fmt"
)

// Status code for errors due to a downstream dependency timeout.
const HttpDependencyTimeout = 597

func formatErrMsg(uuid, errMsg string) string {
	if len(uuid) > 0 {
		return fmt.Sprintf("GET /cache uuid=%s: %s", uuid, errMsg)
	}
	return fmt.Sprintf("GET /cache: %s", errMsg)
}

/**************************/
/* Get errors			  */
/**************************/
// Put error wrapper
func NewPrebidCacheGetError(uuid string, err error, statusCode int) *PrebidCacheGetError {
	return &PrebidCacheGetError{
		uuid:   uuid,
		err:    err,
		status: statusCode,
	}
}

type PrebidCacheGetError struct {
	uuid   string
	err    error
	status int
}

func (ce *PrebidCacheGetError) Error() string {
	if len(ce.uuid) > 0 {
		return fmt.Sprintf("GET /cache uuid=%s: %s", ce.uuid, ce.err.Error())
	}
	return fmt.Sprintf("GET /cache: %s", ce.err.Error())
}

func (ce *PrebidCacheGetError) StatusCode() int {
	return ce.status
}

func (ce *PrebidCacheGetError) IsKeyNotFound() bool {
	_, isKeyNotFound := ce.err.(KeyNotFoundError)
	return isKeyNotFound
}

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

// Record exists is the error we assign to the different backend errors
// that describe a failure to put a value under a UUID that is already taken
type RecordExistsError struct{}

func (e RecordExistsError) Error() string {
	return "Record exists with provided key."
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
	return fmt.Sprintf("trying to put %d keys which is more than the allowed number allowed: %d", e.NumValues, e.MaxNumValues)
}

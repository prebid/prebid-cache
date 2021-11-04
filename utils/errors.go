package utils

import (
	"fmt"
)

// Status code for errors due to a downstream dependency timeout.
const HttpDependencyTimeout = 597

/**************************/
/* Error wrapper          */
/**************************/
func NewPrebidCacheError(e error, statusCode int) PrebidCacheError {
	return PrebidCacheError{e, statusCode}
}

type PrebidCacheError struct {
	err    error
	status int
}

func (ce PrebidCacheError) Error() string {
	return ce.err
}

func (ce PrebidCacheError) Code() int {
	return ce.status
}

/**************************/
/* Get errors			  */
/**************************/

// Key not found
type KeyNotFoundError struct {
	msgPrefix string
}

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
type PutBadRequestError struct {
	Body []byte
}

func (e PutBadRequestError) Error() string {
	return "Request body " + string(e.Body) + " is not valid JSON."
}

// Surpassed the number of elemets allowed to be put
type PutMaxNumValuesError struct {
	NumValues, MaxNumValues int
}

func (e PutMaxNumValuesError) Error() string {
	return fmt.Sprintf("Incoming number of keys %d is more than allowed: %d", e.NumValues, e.MaxNumValues)
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

package utils

import "fmt"

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

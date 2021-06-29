package utils

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

type RecordExistsError struct{}

func (e RecordExistsError) Error() string {
	return "A record already exists under provided key."
}

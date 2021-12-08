package utils

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

type RecordExistsError struct{}

func (e RecordExistsError) Error() string {
	return "Record exists with provided key."
}

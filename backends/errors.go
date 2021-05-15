package backends

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

type MissingUuidError struct{}

func (e MissingUuidError) Error() string {
	return "Missing required parameter uuid"
}

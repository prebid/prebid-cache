package backends

type KeyNotFoundError struct {
	msgPrefix string
}

func (e KeyNotFoundError) Error() string {
	return e.msgPrefix + " Key not found"
}

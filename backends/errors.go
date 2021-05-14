package backends

type KeyNotFoundError struct{}

func (e KeyNotFoundError) Error() string {
	return "Key not found"
}

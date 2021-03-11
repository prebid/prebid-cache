package backends

type PBCKeyNotFoundError struct {
	msg string
}

func (e PBCKeyNotFoundError) Error() string {
	return e.msg + "Key not found"
}

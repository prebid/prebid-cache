package constant

const (
	//TODO: Use codes
	// UUIDMissing : UUID not passed in request
	UUIDMissing = "001"

	// InvalidUUID : UUID length is not 36 characters long which is the expected length
	InvalidUUID = "002"

	// InvalidJSON : Invalid JSON sent in request body
	InvalidJSON = "003"

	// KeyCountExceeded : more keys than allowed in request body
	KeyCountExceeded = "004"

	// MaxSizeExceeded : POST /cache element exceeded max size
	MaxSizeExceeded = "005"

	// TimedOut : Timeout writing value to the backend
	TimedOut = "006"

	// UnexpErr : POST /cache had an unexpected error
	UnexpErr = "007"
)

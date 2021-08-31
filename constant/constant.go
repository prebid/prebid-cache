package constant

const (
	//TODO: Use codes
	// UUIDMissing : UUID not passed in request
	UUIDMissing = "101"

	// InvalidUUID : UUID length is not 36 characters long which is the expected length
	InvalidUUID = "102"

	// InvalidJSON : Invalid JSON sent in request body
	InvalidJSON = "103"

	// KeyCountExceeded : more keys than allowed in request body
	KeyCountExceeded = "104"

	// MaxSizeExceeded : POST /cache element exceeded max size
	MaxSizeExceeded = "105"

	// TimedOut : Timeout writing value to the backend
	TimedOut = "106"

	// UnexpErr : POST /cache had an unexpected error
	UnexpErr = "107"

	//Node and Pod names for K8S
	DEFAULT_NODENAME  = "Default_Node"
	DEFAULT_PODNAME   = "Default_Pod"
	ENV_VAR_NODE_NAME = "MY_NODE_NAME"
	ENV_VAR_POD_NAME  = "MY_POD_NAME"
)

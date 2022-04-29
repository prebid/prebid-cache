package utils

// These strings are prefixed onto data put in the backend, to designate its type.
const (
	XML_PREFIX  = "xml"
	JSON_PREFIX = "json"
)

// The following numeric constants serve as configuration defaults
const (
	CASSANDRA_DEFAULT_TTL_SECONDS    = 2400
	REDIS_DEFAULT_EXPIRATION_MINUTES = 60
	RATE_LIMITER_NUM_REQUESTS        = 100
	REQUEST_MAX_SIZE_BYTES           = 10 * 1024
	REQUEST_MAX_NUM_VALUES           = 10
	REQUEST_MAX_TTL_SECONDS          = 3600
)

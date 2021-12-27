package backends

import (
	"context"
)

// Backend interface for storing data
type Backend interface {
	Put(ctx context.Context, key string, value string, ttlSeconds int) error
	Get(ctx context.Context, key string) (string, error)
}

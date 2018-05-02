package backends

import (
	"context"
)

// Backend interface for storing data
type Backend interface {
	Put(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
}

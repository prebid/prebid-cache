package decorators

import (
	"context"
	"strconv"

	"github.com/prebid/prebid-cache/backends"
)

// EnforceSizeLimit rejects payloads over a max size.
// If a payload is too large, the Put() function will return a BadPayloadSize error.
func EnforceSizeLimit(delegate backends.Backend, maxSize int) backends.Backend {
	return &sizeCappedBackend{
		delegate: delegate,
		limit:    maxSize,
	}
}

type sizeCappedBackend struct {
	delegate backends.Backend
	limit    int
}

func (b *sizeCappedBackend) Get(ctx context.Context, key string) (string, error) {
	return b.delegate.Get(ctx, key)
}

func (b *sizeCappedBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	valueLen := len(value)
	if valueLen == 0 || valueLen > b.limit {
		return &BadPayloadSize{
			limit: b.limit,
			size:  valueLen,
		}
	}

	return b.delegate.Put(ctx, key, value, ttlSeconds)
}

type BadPayloadSize struct {
	limit int
	size  int
}

func (p *BadPayloadSize) Error() string {
	return "Payload size " + strconv.Itoa(p.size) + " exceeded max " + strconv.Itoa(p.limit)
}

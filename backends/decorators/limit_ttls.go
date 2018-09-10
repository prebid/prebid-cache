package decorators

import (
	"context"

	"github.com/prebid/prebid-cache/backends"
)

// LimitTTLs wraps the delegate and makes sure that it never gets TTLs which exceed the max.
func LimitTTLs(delegate backends.Backend, maxTTLSeconds int) backends.Backend {
	return ttlLimited{
		Backend:       delegate,
		maxTTLSeconds: maxTTLSeconds,
	}
}

type ttlLimited struct {
	backends.Backend
	maxTTLSeconds int
}

func (l ttlLimited) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if l.maxTTLSeconds > ttlSeconds {
		return l.Backend.Put(ctx, key, value, ttlSeconds)
	}
	return l.Backend.Put(ctx, key, value, l.maxTTLSeconds)
}

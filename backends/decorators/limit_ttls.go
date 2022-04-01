package decorators

import (
	"context"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/utils"
)

// LimitTTLs wraps the delegate and makes sure that it never gets TTLs which exceed the max.
// or are less than zero.
func LimitTTLs(delegate backends.Backend, maxTTLSeconds int) backends.Backend {
	maxTTL := maxTTLSeconds
	if maxTTLSeconds <= 0 {
		maxTTL = utils.REQUEST_MAX_TTL_SECONDS
	}
	return ttlLimited{
		Backend:       delegate,
		maxTTLSeconds: maxTTL,
	}
}

type ttlLimited struct {
	backends.Backend
	maxTTLSeconds int
}

// Put will make the delegate.Put() call with the default l.maxTTLSeconds whenever the
// request-defined ttl value is out of bounds
func (l ttlLimited) Put(ctx context.Context, key string, value string, requestTTLSeconds int) error {
	ttl := l.maxTTLSeconds

	if l.maxTTLSeconds > requestTTLSeconds && requestTTLSeconds > 0 {
		ttl = requestTTLSeconds
	}
	return l.Backend.Put(ctx, key, value, ttl)
}

// Get will somply make the delegate.Get() call given that no TTL check is needed on the GET side
func (l ttlLimited) Get(ctx context.Context, key string) (string, error) {
	return l.Backend.Get(ctx, key)
}

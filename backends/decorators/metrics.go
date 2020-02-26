package decorators

import (
	"context"
	"strings"
	"time"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
)

type backendWithMetrics struct {
	delegate backends.Backend
	metrics  *metrics.Metrics
}

func (b *backendWithMetrics) Get(ctx context.Context, key string) (string, error) {
	b.metrics.RecGetBackendRequest("add", nil)
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		b.metrics.RecGetBackendRequest("", &start)
	} else {
		b.metrics.RecGetBackendRequest("error", nil)
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		b.metrics.RecPutBackendRequest("xml", nil, 0)
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		b.metrics.RecPutBackendRequest("json", nil, 0)
	} else {
		b.metrics.RecPutBackendRequest("invalid_format", nil, 0)
	}
	if ttlSeconds != 0 {
		b.metrics.RecPutBackendRequest("defines_ttl", nil, 0)
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value, ttlSeconds)
	if err == nil {
		b.metrics.RecPutBackendRequest("", &start, 0)
	} else {
		b.metrics.RecPutBackendRequest("error", nil, 0)
	}
	b.metrics.RecPutBackendRequest("", nil, float64(len(value)))
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate: backend,
		metrics:  m,
	}
}

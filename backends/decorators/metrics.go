package decorators

import (
	"context"
	"strings"
	"time"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
)

type backendWithMetrics struct {
	delegate       backends.Backend
	metricsEngines *metrics.Metrics
}

func (b *backendWithMetrics) Get(ctx context.Context, key string) (string, error) {
	b.metricsEngines.RecGetBackendRequest("add", nil)
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		b.metricsEngines.RecGetBackendRequest("", &start)
	} else {
		b.metricsEngines.RecGetBackendRequest("error", nil)
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		b.metricsEngines.RecPutBackendRequest("xml", nil, 0)
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		b.metricsEngines.RecPutBackendRequest("json", nil, 0)
	} else {
		b.metricsEngines.RecPutBackendRequest("invalid_format", nil, 0)
	}
	if ttlSeconds != 0 {
		b.metricsEngines.RecPutBackendRequest("defines_ttl", nil, 0)
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value, ttlSeconds)
	if err == nil {
		b.metricsEngines.RecPutBackendRequest("", &start, 0)
	} else {
		b.metricsEngines.RecPutBackendRequest("error", nil, 0)
	}
	b.metricsEngines.RecPutBackendRequest("", nil, float64(len(value)))
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate:       backend,
		metricsEngines: m,
	}
}

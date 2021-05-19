package decorators

import (
	"context"
	"strings"
	"time"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/utils"
)

type backendWithMetrics struct {
	delegate backends.Backend
	metrics  *metrics.Metrics
}

func (b *backendWithMetrics) Get(ctx context.Context, key string) (string, error) {

	b.metrics.RecordGetBackendTotal()
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		b.metrics.RecordGetBackendDuration(time.Since(start))
	} else {
		if _, isKeyNotFound := err.(utils.KeyNotFoundError); isKeyNotFound {
			b.metrics.RecordKeyNotFoundError()
		} else if _, isMissingUuidError := err.(utils.MissingKeyError); isMissingUuidError {
			b.metrics.RecordMissingKeyError()
		}
		b.metrics.RecordGetBackendError()
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string, ttlSeconds int) error {

	if strings.HasPrefix(value, backends.XML_PREFIX) {
		b.metrics.RecordPutBackendXml()
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		b.metrics.RecordPutBackendJson()
	} else {
		b.metrics.RecordPutBackendInvalid()
	}
	if ttlSeconds != 0 {
		b.metrics.RecordPutBackendDefTTL()
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value, ttlSeconds)
	if err == nil {
		b.metrics.RecordPutBackendDuration(time.Since(start))
	} else {
		b.metrics.RecordPutBackendError()
	}
	b.metrics.RecordPutBackendSize(float64(len(value)))
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate: backend,
		metrics:  m,
	}
}

package decorators

import (
	"context"
	"github.com/Prebid-org/prebid-cache/backends"
	"github.com/Prebid-org/prebid-cache/metrics"
	"time"
)

type backendWithMetrics struct {
	delegate backends.Backend
	puts     *metrics.MetricsEntry
	gets     *metrics.MetricsEntry
}

func (b *backendWithMetrics) Get(ctx context.Context, key string) (string, error) {
	b.gets.Request.Mark(1)
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		b.gets.Duration.UpdateSince(start)
	} else {
		b.gets.Errors.Mark(1)
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string) error {
	b.puts.Request.Mark(1)
	start := time.Now()
	err := b.delegate.Put(ctx, key, value)
	if err == nil {
		b.puts.Duration.UpdateSince(start)
	} else {
		b.puts.Errors.Mark(1)
	}
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate: backend,
		puts:     m.PutsBackend,
		gets:     m.GetsBackend,
	}
}

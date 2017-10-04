package metrics

import (
	"context"
	"github.com/Prebid-org/prebid-cache/backends"
	"time"
)

type backendWithMetrics struct {
	delegate backends.Backend
	puts     *MetricsEntry
	gets     *MetricsEntry
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

func MonitorBackend(backend backends.Backend, metrics *Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate: backend,
		puts:     metrics.PutsBackend,
		gets:     metrics.GetsBackend,
	}
}

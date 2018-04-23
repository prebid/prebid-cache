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
	puts     *metrics.MetricsEntryByFormat
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
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		b.puts.XmlRequest.Mark(1)
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		b.puts.JsonRequest.Mark(1)
	} else {
		b.puts.InvalidRequest.Mark(1)
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value)
	if err == nil {
		b.puts.Duration.UpdateSince(start)
	} else {
		b.puts.Errors.Mark(1)
	}
	b.puts.RequestLength.Update(int64(len(value)))
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate: backend,
		puts:     m.PutsBackend,
		gets:     m.GetsBackend,
	}
}

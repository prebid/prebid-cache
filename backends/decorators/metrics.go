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
	metricsEngines *metrics.CacheMetricsEngines
}

func (b *backendWithMetrics) Get(ctx context.Context, key string) (string, error) {
	//b.gets.Request.Mark(1)
	b.metricEngines.Add("gets.current_url.request_count", nil, "")
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		//b.gets.Duration.UpdateSince(start)
		b.metrics.Add("gets.backend.request_duration", &start, "")
	} else {
		//b.gets.Errors.Mark(1)
		b.metrics.Add("gets.backend.error_count", nil, "")
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		//b.puts.XmlRequest.Mark(1)
		b.metrics.Add("puts.backend.xml_request_count", nil, "")
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		//b.puts.JsonRequest.Mark(1)
		b.metrics.Add("puts.backend.json_request_count", nil, "")
	} else {
		//b.puts.InvalidRequest.Mark(1)
		b.metrics.Add("puts.backend.unknown_request_count", nil, "")
	}
	if ttlSeconds != 0 {
		//b.puts.DefinesTTL.Mark(1)
		b.metrics.Add("puts.backend.defines_ttl", nil, "")
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value, ttlSeconds)
	if err == nil {
		//b.puts.Duration.UpdateSince(start)
		b.metrics.Add("puts.backend.request_duration", &start, "")
	} else {
		//b.puts.Errors.Mark(1)
		b.metrics.Add("puts.backend.error_count", nil, "")
	}
	//b.puts.RequestLength.Update(int64(len(value)))
	b.metrics.Add("puts.backend.request_size_bytes", nil, value)
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.CacheMetricsEngines) backends.Backend {
	return &backendWithMetrics{
		delegate:       backend,
		metricsEngines: m,
	}
}

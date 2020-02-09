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
	//b.gets.Request.Mark(1)
	//b.metricsEngines.Add("gets.current_url.request_count", nil, "")
	b.metricsEngines.RecGetBackendRequest("add", nil)
	start := time.Now()
	val, err := b.delegate.Get(ctx, key)
	if err == nil {
		//b.gets.Duration.UpdateSince(start)
		//b.metricsEngines.Add("gets.backend.request_duration", &start, "")
		b.metricsEngines.RecGetBackendRequest("", &start)
	} else {
		//b.gets.Errors.Mark(1)
		//b.metricsEngines.Add("gets.backend.error_count", nil, "")
		b.metricsEngines.RecGetBackendRequest("error", nil)
	}
	return val, err
}

func (b *backendWithMetrics) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	if strings.HasPrefix(value, backends.XML_PREFIX) {
		//b.puts.XmlRequest.Mark(1)
		//b.metricsEngines.Add("puts.backend.xml_request_count", nil, "")
		b.metricsEngines.RecPutBackendRequest("xml", nil, 0)
	} else if strings.HasPrefix(value, backends.JSON_PREFIX) {
		//b.puts.JsonRequest.Mark(1)
		//b.metricsEngines.Add("puts.backend.json_request_count", nil, "")
		b.metricsEngines.RecPutBackendRequest("json", nil, 0)
	} else {
		//b.puts.InvalidRequest.Mark(1)
		//b.metricsEngines.Add("puts.backend.unknown_request_count", nil, "")
		b.metricsEngines.RecPutBackendRequest("invalid_format", nil, 0)
	}
	if ttlSeconds != 0 {
		//b.puts.DefinesTTL.Mark(1)
		//b.metricsEngines.Add("puts.backend.defines_ttl", nil, "")
		b.metricsEngines.RecPutBackendRequest("defines_ttl", nil, 0)
	}
	start := time.Now()
	err := b.delegate.Put(ctx, key, value, ttlSeconds)
	if err == nil {
		//b.puts.Duration.UpdateSince(start)
		//b.metricsEngines.Add("puts.backend.request_duration", &start, "")
		b.metricsEngines.RecPutBackendRequest("", &start, 0)
	} else {
		//b.puts.Errors.Mark(1)
		//b.metricsEngines.Add("puts.backend.error_count", nil, "")
		b.metricsEngines.RecPutBackendRequest("error", nil, 0)
	}
	//b.puts.RequestLength.Update(int64(len(value)))
	//b.metricsEngines.Add("puts.backend.request_size_bytes", nil, value)
	b.metricsEngines.RecPutBackendRequest("", nil, float64(len(value)))
	return err
}

func LogMetrics(backend backends.Backend, m *metrics.Metrics) backends.Backend {
	return &backendWithMetrics{
		delegate:       backend,
		metricsEngines: m,
	}
}

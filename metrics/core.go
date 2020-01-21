package metrics

import (
	"time"

	"github.com/prebid/prebid-cache/config"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/rcrowley/go-metrics"
)

/* Interface definition                */
type CacheMetrics interface {
	//Export(cfg config.Metrics) {
	//Implement differently for Prometheus and for Influx. This means we'll have to trim the current Inflix implementation a bit
	Export(cfg config.Metrics)

	//Increment()
	// This one is absolutely needed because we are going to substitute `Mark(1)` and `Inc()` with this function. In other words, this function
	// will call `Mark(1)` or `Inc()` depending whether this is an Influx or a Prometheus metric object
	Increment(metricName string, start *time.Time, value string)
}

type CacheMetricsEngines struct {
	Influx     *InfluxMetrics
	Prometheus *PrometheusMetrics
}

func (me *CacheMetricsEngines) Export(cfg config.Configuration) {
	if cfg.Metrics.Influx.Host != "" {
		me.Influx.Export(cfg)
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		me.Prometheus.Export(cfg)
	}
}

func CreateMetrics(cfg config.Configuration) *CacheMetricsEngines {
	// Create a list of metrics engines to use.
	// Capacity of 2, as unlikely to have more than 2 metrics backends, and in the case
	// of 1 we won't use the list so it will be garbage collected.
	returnEngines := CacheMetricsEngines{Influx: nil, Prometheus: nil}

	if cfg.Metrics.Influx.Host != "" {
		returnEngines.Influx = CreateInfluxMetrics()
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		returnEngines.Prometheus = CreatePrometheusMetrics(cfg.Metrics.Prometheus)
	}

	return &returnEngines
}

func (cacheMetrics CacheMetricsEngines) Add(metricName string, start *time.Time, value string) {
	if cacheMetrics.Influx != nil {
		cacheMetrics.Influx.Increment(metricName, start, value)
	}
	if cacheMetrics.Prometheus != nil {
		cacheMetrics.Prometheus.Increment(metricName, start, value)
	}
}

func CreateInfluxMetrics() *InfluxMetrics {
	flushTime := time.Second * 10
	r := metrics.NewPrefixedRegistry("prebidcache.")
	m := &InfluxMetrics{
		Registry:        r,
		Puts:            NewInfluxMetricsEntry("puts.current_url", r),
		Gets:            NewInfluxMetricsEntry("gets.current_url", r),
		PutsBackend:     NewInfluxMetricsEntryBackendPuts("puts.backend", r),
		GetsBackend:     NewInfluxMetricsEntry("gets.backend", r),
		Connections:     NewInfluxConnectionMetrics(r),
		ExtraTTLSeconds: metrics.GetOrRegisterHistogram("extra_ttl_seconds", r, metrics.NewUniformSample(5000)),
	}

	metrics.RegisterDebugGCStats(m.Registry)
	metrics.RegisterRuntimeMemStats(m.Registry)

	go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	go metrics.CaptureDebugGCStats(m.Registry, flushTime)

	return m
}

func CreatePrometheusMetrics(cfg config.PrometheusMetrics) *PrometheusMetrics {
	cacheWriteTimeBuckts := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	registry := prometheus.NewRegistry()
	return &PrometheusMetrics{
		Registry: registry,
		Puts: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"puts.current_url.request_duration",
				"Duration in seconds to write to Prebid Cache labeled by success or failure.", //Ask in the Github comment section if this descriptions are ok
				[]string{"success"},
				cacheWriteTimeBuckts),
			RequestMetrics: newCounterVecWithLabels(cfg, registry,
				"puts.current_url.",
				"Count of put requests that were successful, returned errors, or were simply bad requests",
				[]string{"error_count", "bad_request_count", "request_count"},
			),
		},
		Gets: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"gets.current_url.request_duration",
				"Duration in seconds to read from Prebid Cache labeled by success or failure.",
				[]string{"success"},
				cacheWriteTimeBuckts),
			RequestMetrics: newCounterVecWithLabels(cfg, registry,
				"gets.current_url.",
				"Count of get requests that were successful, returned errors, or were simply bad requests",
				[]string{"error_count", "bad_request_count", "request_count"},
			),
		},
		PutsBackend: &PrometheusMetricsEntryByFormat{
			Duration: newHistogram(cfg, registry,
				"puts.backend.request_duration",
				"Duration in seconds to write to Prebid Cache backend labeled by success or failure.",
				[]string{"success"},
				cacheWriteTimeBuckts),
			BackendPutMetrics: newCounterVecWithLabels(cfg, registry,
				"puts.backend.",
				"Count of backend put requests that came in XML format, JSON format, were bad requests, returned errors, were invalid or defined a time to live limit.",
				[]string{"error_count", "bad_request_count", "json_request_count", "xml_request_count", "defines_ttl", "unknown_request_count"},
			),
			RequestLength: newHistogram(cfg, registry,
				"puts.backend.request_size_bytes",
				"Size in bytes of backend put request.",
				[]string{"success"},
				cacheWriteTimeBuckts),
		},
		GetsBackend: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"gets.backend.request_duration",
				"Seconds to write to Prebid Cache labeled by success or failure.",
				[]string{"success"},
				cacheWriteTimeBuckts),
			RequestMetrics: newCounterVecWithLabels(cfg, registry,
				"gets.backend.",
				"Count of backend get requests that were successful, returned errors, or were simply bad requests",
				[]string{"error_count", "bad_request_count", "request_count"},
			),
		},
		Connections: &PrometheusConnectionMetrics{
			RequestMetrics: newCounterVecWithLabels(cfg, registry,
				"connections.",
				"Count of number of active connections, connection close errors and conection accept errors.",
				[]string{"active_incoming", "accept_errors", "close_errors"},
			),
		},
		ExtraTTLSeconds: newHistogram(cfg, registry,
			"puts.backend.request_duration",
			"Seconds of extra time to live in seconds labeled as success.",
			[]string{"success"},
			cacheWriteTimeBuckts),
	}
}

// A blank metrics engine in case no  metrics service was specified in the configuration file
type DummyMetricsEngine struct{}

func (m *DummyMetricsEngine) CreateMetrics() {
}
func (m *DummyMetricsEngine) Export(cfg config.Metrics) {
}
func (m *DummyMetricsEngine) Increment(metricName string, start *time.Time, value string) {
}

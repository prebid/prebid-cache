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
	Export(cfg config.Configuration)

	//Increment()
	// This one is absolutely needed because we are going to substitute `Mark(1)` and `Inc()` with this function. In other words, this function
	// will call `Mark(1)` or `Inc()` depending whether this is an Influx or a Prometheus metric object
	Increment(metricName string, start *time.Time, value string)

	//Decrement()
	Decrement(metricName string)
}

type CacheMetricsEngines struct {
	Influx     *InfluxMetrics
	Prometheus *PrometheusMetrics
}

func (me *CacheMetricsEngines) Export(cfg config.Configuration) {
	if cfg.Metrics.Influx.Host != "" {
		me.Influx.Export(cfg.Metrics)
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		me.Prometheus.Export(cfg.Metrics)
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

func (cacheMetrics CacheMetricsEngines) Substract(metricName string) {
	if cacheMetrics.Influx != nil {
		cacheMetrics.Influx.Decrement(metricName)
	}
	if cacheMetrics.Prometheus != nil {
		cacheMetrics.Prometheus.Decrement(metricName)
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
	promMetrics := &PrometheusMetrics{
		Registry: registry,
		RequestDurationMetrics: newHistogramVector(cfg, registry,
			"request_duration",
			"Duration in seconds to write to Prebid Cache labeled by get or put method and current URL or backend request type.",
			[]string{"method", "result"},
			cacheWriteTimeBuckts,
		), // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
		MethodToEndpointMetrics: newCounterVecWithLabels(cfg, registry,
			"request_counts",
			"How many get requests, put requests, and get backend requests cathegorized by total requests, bad requests, and error requests.",
			[]string{"method", "count_type"},
		), //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
		RequestSyzeBytes: newHistogramVector(cfg, registry,
			"request_size",
			"Currently implemented only for backend put requests.",
			[]string{"method"},
			cacheWriteTimeBuckts,
		), //{ "puts.backend.request_size_bytes" }
		ConnectionErrorMetrics: newCounterVecWithLabels(cfg, registry,
			"connection_error_counts",
			"How many accept_errors, or close_errors connections",
			[]string{"connections"},
		), // { "connections.accept_errors", "connections.close_errors"}
		ActiveConnections: newGaugeMetric(cfg, registry,
			"connections_active_incoming",
			"How many connections are currenctly opened",
		), // {"connections.active_incoming"}
		ExtraTTLSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "extra_ttl",
			Help:    "Time to live in seconds",
			Buckets: cacheWriteTimeBuckts,
		}), //{"extra_ttl_seconds"}
	}

	promMetrics.ExtraTTLSeconds.Observe(5000.00)

	return promMetrics
}

// A blank metrics engine in case no  metrics service was specified in the configuration file
type DummyMetricsEngine struct{}

func (m *DummyMetricsEngine) CreateMetrics() {
}
func (m *DummyMetricsEngine) Export(cfg config.Metrics) {
}
func (m *DummyMetricsEngine) Increment(metricName string, start *time.Time, value string) {
}

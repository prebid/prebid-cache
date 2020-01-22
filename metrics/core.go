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
			"Request duration",
			"Duration in seconds to write to Prebid Cache labeled by get or put method and current URL or backend request type.",
			[]string{"method", "result"},
			cacheWriteTimeBuckts,
		),
		MethodToEndpointMetrics: newCounterVecWithLabels(cfg, registry,
			"Puts and gets and GetsBackend request counts",
			"How many get requests, put requests, and get backend requests cathegorized by total requests, bad requests, and error requests.",
			[]string{"method", "count_type"},
		),
		RequestSyzeBytes: newHistogramVector(cfg, registry,
			"Request size in bytes",
			"Currently implemented only for backend put requests.",
			[]string{"method"},
			cacheWriteTimeBuckts,
		),
		ConnectionMetrics: newCounterVecWithLabels(cfg, registry,
			"Connection success and error counts",
			"How many active_incoming, accept_errors, or close_errors connections",
			[]string{"connections"},
		),
		//ExtraTTLSeconds:         *prometheus.Histogram
		//ExtraTTLSeconds: newHistogram(cfg, registry,
		//	"puts.backend.request_duration",
		//	"Seconds of extra time to live in seconds labeled as success.",
		//	[]string{"success"},
		//	cacheWriteTimeBuckts,
		//),
		ExtraTTLSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "pond_temperature_celsius",
			Help:    "The temperature of the frog pond.",
			Buckets: cacheWriteTimeBuckts,
		}),
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

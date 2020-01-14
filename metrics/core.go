package metrics

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/config"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
)

/* Interface definition                */
type CacheMetrics interface {
	//NewMetricsEntry(name string, r metrics.Registry) *MetricsEntry {
	// Above is the original function signature that can always be cfound in `metrics/influx.go`, I should probably back this file up so I don't have
	// to `git checkout --` the file, anyways, I believe we should get rid of the metrics registry and the return value because those two objects will be defined
	// in the object implementing the interface and the respective function implementation whould take care of initializing the corresponding object
	// JUST like we did un the promehteus file. That is, initializing EVERYTHING inside. May7be in the case of Influx, we can make
	// NewMetricsEntry(name string), NewMetricsEntryBackendPuts(name string), and NewConnectionMetrics() local functions that only get called for the Influx DB implementation
	//CreateMetrics()

	//Export(cfg config.Metrics) {
	//Implement differently for Prometheus and for Influx. This means we'll have to trim the current Inflix implementation a bit
	Export(cfg config.Metrics)

	//Increment()
	// This one is absolutely needed because we are going to substitute `Mark(1)` and `Inc()` with this function. In other words, this function
	// will call `Mark(1)` or `Inc()` depending whether this is an Influx or a Prometheus metric object
	Increment(metricName string, start *Time, value string)
}

type CacheMetricsEngines struct {
	MetricsEngines []*CacheMetrics
}

func CreateCacheMetricsEngines() *CacheMetricsEngines {
	// Create a list of metrics engines to use.
	// Capacity of 2, as unlikely to have more than 2 metrics backends, and in the case
	// of 1 we won't use the list so it will be garbage collected.
	returnEngines := CacheMetricsEngines{MetricsEngines: make(CacheMetrics, 0, 2)}

	if cfg.Metrics.Influxdb.Host != "" {
		returnEngine.MetricsEngines = append(CreateInfluxMetrics(), returnEngines.MetricsEngines)
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		returnEngine.MetricsEngines = append(CreatePrometheusMetrics(), returnEngines.MetricsEngines)
	}

	return &returnEngines
}

func (cacheMetrics CacheMetricsEngines) Add(metricName string, start *Time, value string) {
	for metricsEngine := range cacheMetrics.MetricsEngines {
		metricsEngine.Increment(metricName, start, value)
	}
}

func CreateInfluxMetrics() *InfluxMetrics {
	flushTime := time.Second * 10
	r := metrics.NewPrefixedRegistry("prebidcache.")
	m := &InfluxMetrics{
		Registry:        r,
		Puts:            NewMetricsEntry("puts.current_url", r),
		Gets:            NewMetricsEntry("gets.current_url", r),
		PutsBackend:     NewMetricsEntryBackendPuts("puts.backend", r),
		GetsBackend:     NewMetricsEntry("gets.backend", r),
		Connections:     NewConnectionMetrics(r),
		ExtraTTLSeconds: metrics.GetOrRegisterHistogram("extra_ttl_seconds", r, metrics.NewUniformSample(5000)),
	}

	metrics.RegisterDebugGCStats(m.Registry)
	metrics.RegisterRuntimeMemStats(m.Registry)

	go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	go metrics.CaptureDebugGCStats(m.Registry, flushTime)

	return m
}

func CreatePrometheusMetrics() *PrometheusMetrics {
	cacheWriteTimeBuckts := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	return &PrometheusMetrics{
		Registry: prometheus.NewRegistry(),
		Puts: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, Registry,
				"puts.current_url.request_duration",
				/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
				/*edit*/ []string{successLabel},
				cacheWriteTimeBuckts),
			Errors: newCounterWithoutLabels(cfg, Registry,
				"puts.current_url.error_count",
				"Count of put request that returned errors"),
			BadRequest: newCounterWithoutLabels(cfg, Registry,
				"puts.current_url.bad_request_count",
				"Count of bad put requests"),
			Request: newCounterWithoutLabels(cfg, Registry,
				"puts.current_url.request_count",
				"Count of number of put requests"),
		},
		Gets: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, Registry,
				"gets.current_url.request_duration",
				/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
				/*edit*/ []string{successLabel},
				cacheWriteTimeBuckts),
			Errors: newCounterWithoutLabels(cfg, Registry,
				"gets.current_url.error_count",
				"Count of get request that returned errors"),
			BadRequest: newCounterWithoutLabels(cfg, Registry,
				"gets.current_url.bad_request_count",
				"Count of bad get requests"),
			Request: newCounterWithoutLabels(cfg, Registry,
				"gets.current_url.request_count",
				"Count of number of get requests"),
		},
		PutsBackend: &PrometheusMetricsEntryByFormat{
			Duration: newHistogram(cfg, Registry,
				/*edit*/ "puts.backend.request_duration",
				/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
				/*edit*/ []string{successLabel},
				/*edit*/ cacheWriteTimeBuckts),
			Errors: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.error_count",
				"Count of get request that returned errors"),
			BadRequest: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.bad_request_count",
				"Count of bad get requests"),
			JsonRequest: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.json_request_count",
				"Count of bad get requests"),
			XmlRequest: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.xml_request_count",
				"Count of bad get requests"),
			DefinesTTL: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.defines_ttl",
				"Count of bad get requests"),
			InvalidRequest: newCounterWithoutLabels(cfg, Registry,
				"puts.backend.unknown_request_count",
				"Count of bad get requests"),
			RequestLength: newHistogram(cfg, Registry,
				/*edit*/ "puts.backend.request_duration",
				/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
				/*edit*/ []string{successLabel},
				/*edit*/ cacheWriteTimeBuckts),
		},
		GetsBackend: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, Registry,
				"gets.backend.request_duration",
				/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
				/*edit*/ []string{successLabel},
				cacheWriteTimeBuckts),
			Errors: newCounterWithoutLabels(cfg, Registry,
				"gets.backend.error_count",
				"Count of backend get requests that returned errors"),
			BadRequest: newCounterWithoutLabels(cfg, Registry,
				"gets.backend.bad_request_count",
				"Count of bad backend get requests"),
			Request: newCounterWithoutLabels(cfg, Registry,
				"gets.backend.request_count",
				"Count of number of backend get requests"),
		},
		Connections: &PrometheusConnectionMetrics{
			ActiveConnections: newCounterWithoutLabels(cfg, Registry,
				"connections.active_incoming",
				"Count of number of active connections"),
			ConnectionCloseErrors: newCounterWithoutLabels(cfg, Registry,
				"connections.accept_errors",
				"Count of number of connections that have accept errors"),
			ConnectionAcceptErrors: newCounterWithoutLabels(cfg, Registry,
				"connections.close_errors",
				"Count of number of connections that have close errors"),
		},
		ExtraTTLSeconds: newHistogram(cfg, Registry,
			/*edit*/ "puts.backend.request_duration",
			/*edit*/ "Seconds to write to Prebid Cache labeled by success or failure. Failure timing is limited by Prebid Server enforced timeouts.",
			/*edit*/ []string{successLabel},
			/*edit*/ cacheWriteTimeBuckts),
	}
}

// A blank metrics engine in case no  metrics service was specified in the configuration file
type DummyMetricsEngine struct{}

func (m *DummyMetricsEngine) CreateMetrics() {
}
func (m *DummyMetricsEngine) Export(cfg config.Metrics) {
}
func (m *DummyMetricsEngine) Increment(metricName string, start *Time, value string) {
}

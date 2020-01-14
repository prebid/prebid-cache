package metrics

import (
	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
)

/**************************************************
 *	Still missing:
 **************************************************
 *	> Register methods for Prometheus metrics in metrics/prometheus.go
 *	> Modify all histograms
 *	> Tests for config.go
 *	> Tests for metrics/prometheus.go
 *	> Upload so they can see progress
 *	> Make the option betewwn Prometheus and InfluxDB configurable
 **************************************************/
/**************************************************
 *	Object definition
 **************************************************/

type PrometheusMetrics struct {
	Registry        *prometheus.Registry
	Puts            *PrometheusMetricsEntry
	Gets            *PrometheusMetricsEntry
	PutsBackend     *PrometheusMetricsEntryByFormat
	GetsBackend     *PrometheusMetricsEntry
	Connections     *PrometheusConnectionMetrics
	ExtraTTLSeconds *prometheus.HistogramVec
}
type PrometheusMetricsEntry struct {
	Duration   *prometheus.HistogramVec
	Errors     prometheus.Counter
	BadRequest prometheus.Counter
	Request    prometheus.Counter
}

type PrometheusMetricsEntryByFormat struct {
	Duration       metrics.Histogram
	Errors         prometheus.Counter
	BadRequest     prometheus.Counter
	JsonRequest    prometheus.Counter
	XmlRequest     prometheus.Counter
	DefinesTTL     prometheus.Counter
	InvalidRequest prometheus.Counter
	RequestLength  metrics.Histogram
}

type PrometheusConnectionMetrics struct {
	ActiveConnections      prometheus.Counter
	ConnectionCloseErrors  prometheus.Counter
	ConnectionAcceptErrors prometheus.Counter
}

/**************************************************
 *	Init functions
 **************************************************/

func newCounterWithoutLabels(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string) prometheus.Counter {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	counter := prometheus.NewCounter(opts)
	registry.MustRegister(counter)
	return counter
}

func newHistogram(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	histogram := prometheus.NewHistogramVec(opts, labels)
	registry.MustRegister(histogram)
	return histogram
}

/**************************************************
 *	Functions to record metrics
 *	> How does Influx records? based on what values?
 *	> Once we know this, just translate to `promMetric.Inc()` and we are all set
 **************************************************/
// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m PrometheusMetrics) Export(cfg config.Metrics) {
	logrus.Infof("Metrics will be exported to Influx with host=%s, db=%s, username=%s", cfg.Influx.Host, cfg.Influx.Database, cfg.Influx.Username)
	influxdb.InfluxDB(
		m.Registry,          // metrics registry
		time.Second*10,      // interval
		cfg.Influx.Host,     // the InfluxDB url
		cfg.Influx.Database, // your InfluxDB database
		cfg.Influx.Username, // your InfluxDB user
		cfg.Influx.Password, // your InfluxDB password
	)
	return
}

func (m PrometheusMetrics) Increment(metricName string, start *Time, value string) {
	switch metricName {
	case "puts.current_url.request_duration":
		m.Puts.Duration.With(prometheus.Labels{
			successLabel: strconv.FormatBool("success"),
		}).Observe(time.Since(*start).Seconds())
	case "puts.current_url.error_count":
		m.Puts.Errors.Inc()
	case "puts.current_url.bad_request_count":
		m.Puts.BadRequest.Inc()
	case "puts.current_url.request_count":
		m.Puts.Request.Inc()
	case "gets.current_url.request_duration":
		m.Gets.Duration.With(prometheus.Labels{
			successLabel: strconv.FormatBool("success"),
		}).Observe(time.Since(*start).Seconds())
	case "gets.current_url.error_count":
		m.Gets.Errors.Inc()
	case "gets.current_url.bad_request_count":
		m.Gets.BadRequest.Inc()
	case "gets.current_url.request_count":
		m.Gets.Request.Inc()
	case "puts.backend.request_duration":
		m.PutsBackend.Duration.With(prometheus.Labels{
			successLabel: strconv.FormatBool("success"),
		}).Observe(time.Since(*start).Seconds())
	case "puts.backend.error_count":
		m.PutsBackend.Errors.Inc()
	case "puts.backend.bad_request_count":
		m.PutsBackend.BadRequest.Inc()
	case "puts.backend.json_request_count":
		m.PutsBackend.JsonRequest.Inc()
	case "puts.backend.xml_request_count":
		m.PutsBackend.XmlRequest.Inc()
	case "puts.backend.defines_ttl":
		m.PutsBackend.DefinesTTL.Inc()
	case "puts.backend.unknown_request_count":
		m.PutsBackend.InvalidRequest.Inc()
	case "puts.backend.request_size_bytes":
		m.PutsBackend.RequestLength.With(prometheus.Labels{
			successLabel: strconv.FormatBool("request_size_bytes"),
		}).Inc(uint64(len(value)))
	case "gets.backend.request_duration":
		m.GetsBackend.Duration.With(prometheus.Labels{
			successLabel: strconv.FormatBool("success"),
		}).Observe(time.Since(*start).Seconds())
	case "gets.backend.error_count":
		m.GetsBackend.Errors.Inc()
	case "gets.backend.bad_request_count":
		m.GetsBackend.BadRequest.Inc()
	case "gets.backend.request_count":
		m.GetsBackend.Request.Inc()
	case "connections.active_incoming":
		m.Connections.ActiveConnections.Inc()
	case "connections.accept_errors":
		m.Connections.ConnectionCloseErrors.Inc()
	case "connections.close_errors":
		m.Connections.ConnectionAcceptErrors.Inc()
	default:
		//error
	}
}

/*  To record:
m.Puts.Duration.With(prometheus.Labels{ successLabel: strconv.FormatBool("success"), }).Observe(time.Since(*start).Seconds())
m.Gets.Duration.With(prometheus.Labels{ successLabel: strconv.FormatBool("success"), }).Observe(time.Since(*start).Seconds())
m.PutsBackend.Duration.With(prometheus.Labels{ successLabel: strconv.FormatBool("success"), }).Observe(time.Since(*start).Seconds())
m.GetsBackend.Duration.With(prometheus.Labels{ successLabel: strconv.FormatBool("success"), }).Observe(time.Since(*start).Seconds())
*/

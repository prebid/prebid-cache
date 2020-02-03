package metrics

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"time"
)

/**************************************************
 *	Object definition
 **************************************************/
type PrometheusMetrics struct {
	Registry        *prometheus.Registry
	Puts            *PrometheusRequestStatusMetric
	Gets            *PrometheusRequestStatusMetric
	PutsBackend     *PrometheusRequestStatusMetricByFormat
	GetsBackend     *PrometheusRequestStatusMetric
	Connections     *PrometheusConnectionMetrics
	ExtraTTLSeconds *PrometheusExtraTTLMetrics
}

type PrometheusRequestStatusMetric struct {
	Duration      prometheus.Histogram   //Non vector
	RequestStatus *prometheus.CounterVec // CounterVec "status": "ok", "error", or "bad_request"
}

type PrometheusRequestStatusMetricByFormat struct {
	RequestLength      metrics.Histogram      //Non vector
	PutBackendRequests *prometheus.CounterVec // CounterVec "format": "json" or  "xml","status": "ok", "error", or "bad_request","definesTimeToLive": "TTL_present", or "TTL_missing"
	RequestLength      metrics.Histogram      //Non vector
}

type PrometheusConnectionMetrics struct {
	ConnectionsErrors *prometheus.CounterVec // the "Connection_error" label will hold the values "accept" or "close"
}

type PrometheusExtraTTLMetrics struct {
	ExtraTTLSeconds *prometheus.Histogram
}

/**************************************************
 *	Init functions
 **************************************************/
func CreatePrometheusMetrics(cfg config.PrometheusMetrics) *PrometheusMetrics {
	cacheWriteTimeBuckts := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	requestSizeBuckts := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1} // TODO: tweak
	registry := prometheus.NewRegistry()
	promMetrics := &PrometheusMetrics{
		//Registry        *prometheus.Registry
		Registry: registry,
		//Puts            *PrometheusMetricsEntry
		Puts: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"puts.current_url.request_duration", //modify according to InfluxDB name
				"Duration in seconds Prebid Cache takes to process put requests.",
				cacheWriteTimeBuckts,
			), // {"gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"puts.current_url",
				"Count of total requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "ok", "error", or "bad_request"
			), //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
		},
		//Gets            *PrometheusMetricsEntry
		Gets: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"gets.current_url.request_duration",
				"Duration in seconds Prebid Cache takes to process get requests.",
				cacheWriteTimeBuckts,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets.current_url",
				"Count of total get requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "ok", "error", or "bad_request"
			), //{"gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count"}
		},
		//PutsBackend     *PrometheusMetricsEntryByFormat
		PutsBackend: &PrometheusMetricsEntryByFormat{
			Duration: newHistogram(cfg, registry,
				"puts.backend.request_duration",
				"Duration in seconds Prebid Cache takes to process backend put requests.",
				cacheWriteTimeBuckts,
			),
			//PutBackendRequests *prometheus.CounterVec
			PutBackendRequests: newCounterVecWithLabels(cfg, registry,
				"puts.backend",
				"Count of total requests to Prebid Cache labeled by format, status and whether or not it comes with TTL",
				[]string{"format", "status", "definesTimeToLive"},
			), // CounterVec "format": "json" or  "xml","status": "ok", "error", or "bad_request","definesTimeToLive": "TTL_present", or "TTL_missing"
			//{"puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count"}
			RequestLength: newHistogram(cfg, registry,
				"puts.backend.request_size_bytes",
				"Size in bytes of a backend put request.",
				requestSizeBuckts,
			),
		},
		//GetsBackend     *PrometheusMetricsEntry
		GetsBackend: &PrometheusMetricsEntry{
			Duration: newHistogram(cfg, registry,
				"gets.backend.request_duration",
				"Duration in seconds Prebid Cache takes to process backend get requests.",
				cacheWriteTimeBuckts,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets.backend",
				"Count of total backend get requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "ok", "error", or "bad_request"
			), //{"gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}

		},
		//Connections     *PrometheusConnectionMetrics
		Connections: &PrometheusConnectionMetrics{
			ConnectionsOpened: newSingleCounter(cfg, registry,
				"connections",
				"Count of total number of connectionsbackend get requests to Prebid Server labeled by status.",
			),
			ConnectionsErrors: newCounterVecWithLabels(cfg, registry,
				"connection_error",
				"Count the number of connection accept errors or connection close errors",
				[]string{"connection_error"},
			), // "connection_error" = {"accept", "close"}
		},

		//ExtraTTLSeconds *prometheus.HistogramVec
		ExtraTTLSeconds: &PrometheusExtraTTLMetrics{
			newHistogram(cfg, registry,
				"extra_ttl_seconds",
				"Extra time to live in seconds specified",
				cacheWriteTimeBuckts,
			),
		},
	}
	promMetrics.ExtraTTLSeconds.Observe(5000.00)

	return promMetrics
}

/**************************************************
 *	Helper Init functions
 **************************************************/
func newCounterVecWithLabels(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string, labels []string) *prometheus.CounterVec {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	counterVec := prometheus.NewCounterVec(opts, labels)
	registry.MustRegister(counterVec)
	return counterVec
}

func newSingleCounter(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string) *prometheus.Counter {
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

func newHistogram(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	histogram := prometheus.NewHistogram(opts)
	registry.MustRegister(histogram)
	return histogram
}

func newGaugeMetric(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string) prometheus.Gauge {
	opts := prometheus.GaugeOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	gauge := prometheus.NewGauge(opts)
	registry.MustRegister(gauge)
	return gauge
}

func newHistogramVector(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
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
 *	DEPECRATED Functions to record metrics
 **************************************************/
// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m PrometheusMetrics) Export(cfg config.Metrics) {
	logrus.Infof("Metrics will be exported to Prometheus with host=%s, db=%s, username=%s", cfg.Influx.Host, cfg.Influx.Database, cfg.Influx.Username)
	//influxdb.InfluxDB(
	//	m.Registry,          // metrics registry
	//	time.Second*10,      // interval
	//	cfg.Influx.Host,     // the InfluxDB url
	//	cfg.Influx.Database, // your InfluxDB database
	//	cfg.Influx.Username, // your InfluxDB user
	//	cfg.Influx.Password, // your InfluxDB password
	//)
	return
}

func (m PrometheusMetrics) Increment(metricName string, start *time.Time, value string) {
	metricNameTokens := strings.Split(metricName, ".")

	if len(metricNameTokens) == 2 && metricNameTokens[0] == "connections" {
		switch metricNameTokens[1] {
		case "close_errors":
			fallthrough
		case "accept_errors":
			m.ConnectionErrorMetrics.With(prometheus.Labels{
				metricNameTokens[0]: metricNameTokens[1], // { "connections.accept_errors", "connections.close_errors"}
			}).Inc()
		case "active_incoming":
			m.ActiveConnections.Inc() //{ "connections.active_incoming"}
		}
	} else if len(metricNameTokens) == 3 {
		label := fmt.Sprintf("%s.%s", metricNameTokens[0], metricNameTokens[1])
		if metricNameTokens[0] == "gets" || metricNameTokens[0] == "puts" {
			if metricNameTokens[2] == "request_duration" {
				m.RequestDurationMetrics.With(prometheus.Labels{"method": label, "result": metricNameTokens[2]}).Observe(time.Since(*start).Seconds())
				// {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
			} else if metricNameTokens[2] == "request_size_bytes" {
				m.RequestSyzeBytes.With(prometheus.Labels{
					"method": fmt.Sprintf("%s.%s", label, metricNameTokens[2]), // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
				}).Observe(float64(len(value)))
			} else {
				m.MethodToEndpointMetrics.With(prometheus.Labels{
					"method": label, "count_type": metricNameTokens[2], //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
				}).Inc()
			}
		}
	}
}

func (m PrometheusMetrics) Decrement(metricName string) {
	switch metricName {
	case "connections.active_incoming":
		m.ActiveConnections.Dec()
	default:
		//error
	}
}

/**************************************************
 *	NEW Functions to record metrics
 **************************************************/
func (metricObj *PrometheusRequestStatusMetric) RecordRequestMetric(status string, duration *time.Time) {
	//Duration      prometheus.Histogram   //Non vector
	//RequestStatus *prometheus.CounterVec // CounterVec "status": "ok", "error", or "bad_request"
	switch status {
	case "ok":
		fallthrough
	case "error":
		fallthrough
	case "bad_request":
		metricObj.RequestStatus.With(prometheus.Labels{
			"status": status,
		}).Inc()
	case "duration":
		metricObj.Duration.Observe(duration.Seconds())
	default:
		//err := &errortypes.AnError{
		//	Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		//}
	}
}
func (metricByFormat *PrometheusRequestStatusMetricByFormat) RecordRequestMetricByFormat(status string, duration *time.Time, sizeInBytes float64) {
	//Duration      metrics.Histogram
	//PutBackendRequests *prometheus.CounterVec // CounterVec "format": "json", "xml", "invalid_format", or "defines_ttl"
	//RequestLength      metrics.Histogram
	switch status {
	case "json":
		fallthrough
	case "xml":
		fallthrough
	case "invalid_format":
		fallthrough
	case "defines_ttl":
		metricByFormat.RequestStatus.With(prometheus.Labels{
			"format": status,
		}).Inc()
	case "duration":
		metricByFormat.Duration.Observe(duration.Seconds())
	case "size_bytes":
		metricByFormat.RequestLength.Observe(sizeInBytes)
	default:
		//err := &errortypes.AnError{
		//	Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		//}
	}
}
func (metricObj *PrometheusConnectionMetrics) RecordConnectionMetrics(accept bool) {
	//ConnectionsErrors *prometheus.CounterVec // the "Connection_error" label will hold the values "accept" or "close"
	var labelValue string
	if success {
		labelValue = "accept"
	} else {
		labelValue = "close"
	}
	metricObj.ConnectionsErrors.With(prometheus.Labels{
		"connection_error": labelValue,
	}).Inc()
}

func (m *Metrics) RecordExtraTTLSeconds(success bool) {
	//ExtraTTLSeconds *prometheus.HistogramVec
}

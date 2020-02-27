package metrics

import (
	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

// Constants and global variables
var TenSeconds time.Duration = time.Second * 10
var AddLabel string = "add"
var ErrorLabel string = "error"
var BadRequestLabel string = "bad_request"
var JsonLabel string = "json"
var XmlLabel string = "xml"
var DefinesTTLLabel string = "defines_ttl"
var InvFormatLabel string = "invalid_format"
var SubstractLabel string = "substract"
var CloseLabel string = "close"
var AcceptLabel string = "accept"

//	Object definition
type PrometheusMetrics struct {
	Registry    *prometheus.Registry
	Puts        *PrometheusRequestStatusMetric
	Gets        *PrometheusRequestStatusMetric
	PutsBackend *PrometheusRequestStatusMetricByFormat
	GetsBackend *PrometheusRequestStatusMetric
	Connections *PrometheusConnectionMetrics
	ExtraTTL    *PrometheusExtraTTLMetrics
}

type PrometheusRequestStatusMetric struct {
	Duration      prometheus.Histogram   //Non vector
	RequestStatus *prometheus.CounterVec // CounterVec "status": "add", "error", or "bad_request"
}

type PrometheusRequestStatusMetricByFormat struct {
	Duration           prometheus.Histogram   //Non vector
	PutBackendRequests *prometheus.CounterVec // CounterVec "label": "json" or  "xml","status": "add", "error", or "bad_request","definesTimeToLive": "TTL_present", or "TTL_missing"
	RequestLength      prometheus.Histogram   //Non vector
}

type PrometheusConnectionMetrics struct {
	ConnectionsOpened prometheus.Gauge
	ConnectionsErrors *prometheus.CounterVec // the "Connection_error" label will hold the values "accept" or "close"
}

type PrometheusExtraTTLMetrics struct {
	ExtraTTLSeconds prometheus.Histogram
}

//	Init functions
func CreatePrometheusMetrics(cfg config.PrometheusMetrics) *PrometheusMetrics {
	cacheWriteTimeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	requestSizeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1} // TODO: tweak
	registry := prometheus.NewRegistry()
	promMetrics := &PrometheusMetrics{
		Registry: registry,
		Puts: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"puts.current_url.request_duration", //modify according to InfluxDB name
				"Duration in seconds Prebid Cache takes to process put requests.",
				cacheWriteTimeBuckets,
			), // {"gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"puts.current_url",
				"Count of total requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "add", "error", or "bad_request"
			), //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
		},
		Gets: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"gets.current_url.request_duration",
				"Duration in seconds Prebid Cache takes to process get requests.",
				cacheWriteTimeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets.current_url",
				"Count of total get requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "add", "error", or "bad_request"
			), //{"gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count"}
		},
		PutsBackend: &PrometheusRequestStatusMetricByFormat{
			Duration: newHistogram(cfg, registry,
				"puts.backend.request_duration",
				"Duration in seconds Prebid Cache takes to process backend put requests.",
				cacheWriteTimeBuckets,
			),
			PutBackendRequests: newCounterVecWithLabels(cfg, registry,
				"puts.backend",
				"Count of total requests to Prebid Cache labeled by format, status and whether or not it comes with TTL",
				[]string{"label"},
			), // CounterVec "label": "json" or  "xml","status": "add", "error", or "bad_request","definesTimeToLive": "TTL_present", or "TTL_missing"
			//{"puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count"}
			RequestLength: newHistogram(cfg, registry,
				"puts.backend.request_size_bytes",
				"Size in bytes of a backend put request.",
				requestSizeBuckets,
			),
		},
		GetsBackend: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"gets.backend.request_duration",
				"Duration in seconds Prebid Cache takes to process backend get requests.",
				cacheWriteTimeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets.backend",
				"Count of total backend get requests to Prebid Server labeled by status.",
				[]string{"status"}, // CounterVec labels --> "status": "add", "error", or "bad_request"
			), //{"gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}

		},
		Connections: &PrometheusConnectionMetrics{
			ConnectionsOpened: newGaugeMetric(cfg, registry,
				"connections",
				"Count of total number of connectionsbackend get requests to Prebid Server labeled by status.",
			),
			ConnectionsErrors: newCounterVecWithLabels(cfg, registry,
				"connection_error",
				"Count the number of connection accept errors or connection close errors",
				[]string{"connection_error"},
			), // "connection_error" = {"accept", "close"}
		},
		ExtraTTL: &PrometheusExtraTTLMetrics{
			ExtraTTLSeconds: newHistogram(cfg, registry,
				"extra_ttl_seconds",
				"Extra time to live in seconds specified",
				cacheWriteTimeBuckets,
			),
		},
	}
	promMetrics.ExtraTTL.ExtraTTLSeconds.Observe(5000.00)

	return promMetrics
}

//	Helper Init functions
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

func newSingleCounter(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string) prometheus.Counter {
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

func newHistogram(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, buckets []float64) prometheus.Histogram {
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

//	Functions to record metrics
func (m PrometheusMetrics) Export(cfg config.Metrics) {
}

func (m *PrometheusMetrics) RecordPutError() {
	m.Puts.RequestStatus.With(prometheus.Labels{"status": "error"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBadRequest() {
	m.Puts.RequestStatus.With(prometheus.Labels{"status": "bad_request"}).Inc()
}

func (m *PrometheusMetrics) RecordPutTotal() {
	m.Puts.RequestStatus.With(prometheus.Labels{"status": "add"}).Inc()
}

func (m *PrometheusMetrics) RecordPutDuration(duration *time.Time) {
	m.Puts.Duration.Observe(time.Since(*duration).Seconds())
}

func (m *PrometheusMetrics) RecordGetError() {
	m.Gets.RequestStatus.With(prometheus.Labels{"status": "error"}).Inc()
}

func (m *PrometheusMetrics) RecordGetBadRequest() {
	m.Gets.RequestStatus.With(prometheus.Labels{"status": "bad_request"}).Inc()
}

func (m *PrometheusMetrics) RecordGetTotal() {
	m.Gets.RequestStatus.With(prometheus.Labels{"status": "add"}).Inc()
}

func (m *PrometheusMetrics) RecordGetDuration(duration *time.Time) {
	m.Gets.Duration.Observe(time.Since(*duration).Seconds())
}

func (m *PrometheusMetrics) RecordPutBackendXml() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{"label": "xml"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendJson() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{"label": "json"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendInvalid() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{"label": "invalid_format"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendDefTTL() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{"label": "defines_ttl"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendDuration(duration *time.Time) {
	m.PutsBackend.Duration.Observe(time.Since(*duration).Seconds())
}

func (m *PrometheusMetrics) RecordPutBackendError() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{"label": "error"}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendSize(sizeInBytes float64) {
	m.PutsBackend.RequestLength.Observe(sizeInBytes)
}

func (m *PrometheusMetrics) RecordGetBackendTotal() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{"label": "add"}).Inc()
}

func (m *PrometheusMetrics) RecordGetBackendDuration(duration *time.Time) {
	m.GetsBackend.Duration.Observe(time.Since(*duration).Seconds())
}

func (m *PrometheusMetrics) RecordGetBackendError() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{"status": "error"}).Inc()
}

func (m *PrometheusMetrics) IncreaseOpenConnections() {
	m.Connections.ConnectionsOpened.Inc()
}

func (m *PrometheusMetrics) DecreaseOpenConnections() {
	m.Connections.ConnectionsOpened.Dec()
}

func (m *PrometheusMetrics) RecordCloseConnectionErrors() {
	m.Connections.ConnectionsErrors.With(prometheus.Labels{"connection_error": "close"}).Inc()
}

func (m *PrometheusMetrics) RecordAcceptConnectionErrors() {
	m.Connections.ConnectionsErrors.With(prometheus.Labels{"connection_error": "accept"}).Inc()
}

func (m *PrometheusMetrics) RecordExtraTTLSeconds(value float64) {
	m.ExtraTTL.ExtraTTLSeconds.Observe(value)
}

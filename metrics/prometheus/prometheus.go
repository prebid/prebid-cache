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

//NEW Functions to record metrics
func (m *PrometheusMetrics) RecordPutRequest(status string, duration *time.Time) {
	incCounterInVector(m.Puts.RequestStatus, "status", status, map[string]bool{AddLabel: true, ErrorLabel: true, BadRequestLabel: true})
	incDuration(m.Puts.Duration, duration)
}

func (m *PrometheusMetrics) RecordGetRequest(status string, duration *time.Time) {
	incCounterInVector(m.Gets.RequestStatus, "status", status, map[string]bool{AddLabel: true, ErrorLabel: true, BadRequestLabel: true})
	incDuration(m.Gets.Duration, duration)
	/*
		b.metrics.RecPutBackendRequest("error", nil, 0); m.Gets.Errors.Mark(1)
		b.metrics.RecGetBackendRequest("bad_request", nil); m.Gets.BadRequest.Mark(1)
		b.metrics.RecGetBackendRequest("add", nil); m.Gets.Request.Mark(1)
		b.metrics.RecPutBackendRequest("", &start, 0); m.Gets.Duration.UpdateSince(*duration)
	*/
}

func (m *PrometheusMetrics) RecordPutBackendRequest(status string, duration *time.Time, sizeInBytes float64) {
	incDuration(m.PutsBackend.Duration, duration)
	incCounterInVector(m.PutsBackend.PutBackendRequests, "label", status, map[string]bool{AddLabel: true, JsonLabel: true, XmlLabel: true, InvFormatLabel: true, DefinesTTLLabel: true, ErrorLabel: true})
	incSize(m.PutsBackend.RequestLength, sizeInBytes)
	/*
		m.PutsBackend.Request.Mark(1); b.metrics.RecPutBackendRequest("add", nil, 0)
		m.PutsBackend.XmlRequest.Mark(1); b.metrics.RecPutBackendRequest("xml", nil, 0)
		m.PutsBackend.JsonRequest.Mark(1); b.metrics.RecPutBackendRequest("json", nil, 0)
		m.PutsBackend.InvalidRequest.Mark(1);b.metrics.RecPutBackendRequest("invalid_format", nil, 0)
		m.PutsBackend.DefinesTTL.Mark(1); b.metrics.RecPutBackendRequest("defines_ttl", nil, 0)
		m.PutsBackend.Duration.UpdateSince(*duration); b.metrics.RecPutBackendRequest("", &start, 0)
		m.PutsBackend.Errors.Mark(1);b.metrics.RecPutBackendRequest("error", nil, 0)
		m.PutsBackend.RequestLength.Update(int64(sizeInBytes)); b.metrics.RecPutBackendRequest("", nil, float64(len(value)))
		m.PutsBackend.BadRequest.Mark(1); b.metrics.RecPutBackendRequest("bad_request", nil, 0)
	*/
}

func (m *PrometheusMetrics) RecordGetBackendRequest(status string, duration *time.Time) {
	incCounterInVector(m.GetsBackend.RequestStatus, "status", status, map[string]bool{AddLabel: true, ErrorLabel: true, BadRequestLabel: true})
	incDuration(m.GetsBackend.Duration, duration)
	/*
		m.GetsBackend.Request.Mark(1); b.metrics.RecGetBackendRequest("add", nil)
		m.GetsBackend.Duration.UpdateSince(*duration); b.metrics.RecGetBackendRequest("", &start)
		m.GetsBackend.Errors.Mark(1); b.metrics.RecGetBackendRequest("error", nil)
			m.GetsBackend.BadRequest.Mark(1)
	*/
}

func (m *PrometheusMetrics) RecordConnectionMetrics(label string) {
	if label == AddLabel {
		m.Connections.ConnectionsOpened.Inc()
	} else if label == SubstractLabel {
		m.Connections.ConnectionsOpened.Dec()
	}
	incCounterInVector(m.Connections.ConnectionsErrors, "connection_error", label, map[string]bool{AcceptLabel: true, CloseLabel: true})
}

func (m *PrometheusMetrics) RecordExtraTTLSeconds(value float64) {
	m.ExtraTTL.ExtraTTLSeconds.Observe(value)
}

//	Auxiliary functions to record metrics
func incCounterInVector(counter *prometheus.CounterVec, label string, status string, labelMap map[string]bool) {
	if labelMap[status] {
		counter.With(prometheus.Labels{label: status}).Inc()
	}
}

func incDuration(histogram prometheus.Histogram, duration *time.Time) {
	if duration != nil {
		histogram.Observe(time.Since(*duration).Seconds())
	}
}

func incSize(m prometheus.Histogram, sizeInBytes float64) {
	if sizeInBytes > 0 {
		m.Observe(sizeInBytes)
	}
}

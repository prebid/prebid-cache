package metrics

import (
	"fmt"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusKey    string = "status"
	FormatKey    string = "format"
	ConnErrorKey string = "connection_error"

	ConnOpenedKey string = "connection_opened"
	ConnClosedKey string = "connection_closed"

	TotalsVal     string = "total"
	ErrorVal      string = "error"
	BadRequestVal string = "bad_request"
	JsonVal       string = "json"
	XmlVal        string = "xml"
	DefinesTTLVal string = "defines_ttl"
	InvFormatVal  string = "invalid_format"
	CloseVal      string = "close"
	AcceptVal     string = "accept"

	MetricsPrometheus = "Prometheus"
)

type PrometheusMetrics struct {
	Registry    *prometheus.Registry
	Puts        *PrometheusRequestStatusMetric
	Gets        *PrometheusRequestStatusMetric
	PutsBackend *PrometheusRequestStatusMetricByFormat
	GetsBackend *PrometheusRequestStatusMetric
	Connections *PrometheusConnectionMetrics
	ExtraTTL    *PrometheusExtraTTLMetrics
	MetricsName string
}

type PrometheusRequestStatusMetric struct {
	Duration      prometheus.Histogram
	RequestStatus *prometheus.CounterVec
}

type PrometheusRequestStatusMetricByFormat struct {
	Duration           prometheus.Histogram
	PutBackendRequests *prometheus.CounterVec
	RequestLength      prometheus.Histogram
}

type PrometheusConnectionMetrics struct {
	ConnectionsErrors *prometheus.CounterVec
	ConnectionsClosed prometheus.Counter
	ConnectionsOpened prometheus.Counter
}

type PrometheusExtraTTLMetrics struct {
	ExtraTTLSeconds prometheus.Histogram
}

func CreatePrometheusMetrics(cfg config.PrometheusMetrics) *PrometheusMetrics {
	timeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	requestSizeBuckets := []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	registry := prometheus.NewRegistry()
	promMetrics := &PrometheusMetrics{
		Registry: registry,
		Puts: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"puts_request_duration",
				"Duration in seconds Prebid Cache takes to process put requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"puts_request",
				"Count of total requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		Gets: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"gets_request_duration",
				"Duration in seconds Prebid Cache takes to process get requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets_request",
				"Count of total get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		PutsBackend: &PrometheusRequestStatusMetricByFormat{
			Duration: newHistogram(cfg, registry,
				"puts_backend_duration",
				"Duration in seconds Prebid Cache takes to process backend put requests.",
				timeBuckets,
			),
			PutBackendRequests: newCounterVecWithLabels(cfg, registry,
				"puts_backend",
				"Count of total requests to Prebid Cache labeled by format, status and whether or not it comes with TTL",
				[]string{FormatKey},
			),
			RequestLength: newHistogram(cfg, registry,
				"puts_backend_request_size_bytes",
				"Size in bytes of a backend put request.",
				requestSizeBuckets,
			),
		},
		GetsBackend: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"gets_backend_duration",
				"Duration in seconds Prebid Cache takes to process backend get requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets_backend",
				"Count of total backend get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		Connections: &PrometheusConnectionMetrics{
			ConnectionsClosed: newSingleCounter(cfg, registry, ConnOpenedKey, "Count the number of open connections"),
			ConnectionsOpened: newSingleCounter(cfg, registry, ConnClosedKey, "Count the number of closed connections"),
			ConnectionsErrors: newCounterVecWithLabels(cfg, registry,
				ConnErrorKey,
				"Count the number of connection accept errors or connection close errors",
				[]string{ConnErrorKey},
			),
		},
		ExtraTTL: &PrometheusExtraTTLMetrics{
			ExtraTTLSeconds: newHistogram(cfg, registry,
				"extra_ttl_seconds",
				"Extra time to live in seconds specified",
				timeBuckets,
			),
		},
		MetricsName: MetricsPrometheus,
	}

	// Should be the equivalent of the following influx collectors
	// go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	// go metrics.CaptureDebugGCStats(m.Registry, flushTime)
	collectorNamespace := fmt.Sprintf("%s_%s", cfg.Namespace, cfg.Subsystem)
	promMetrics.Registry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{Namespace: collectorNamespace}),
	)

	preloadLabelValues(promMetrics)
	return promMetrics
}

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

func (m PrometheusMetrics) Export(cfg config.Metrics) {
}

func (m *PrometheusMetrics) GetMetricsEngineName() string {
	return m.MetricsName
}

func (m *PrometheusMetrics) GetEngineRegistry() interface{} {
	return m.Registry
}

func (m *PrometheusMetrics) RecordPutError() {
	m.Puts.RequestStatus.With(prometheus.Labels{StatusKey: ErrorVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBadRequest() {
	m.Puts.RequestStatus.With(prometheus.Labels{StatusKey: BadRequestVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutTotal() {
	m.Puts.RequestStatus.With(prometheus.Labels{StatusKey: TotalsVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutDuration(duration time.Duration) {
	m.Puts.Duration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordGetError() {
	m.Gets.RequestStatus.With(prometheus.Labels{StatusKey: ErrorVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetBadRequest() {
	m.Gets.RequestStatus.With(prometheus.Labels{StatusKey: BadRequestVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetTotal() {
	m.Gets.RequestStatus.With(prometheus.Labels{StatusKey: TotalsVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetDuration(duration time.Duration) {
	m.Gets.Duration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordPutBackendXml() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{FormatKey: XmlVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendJson() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{FormatKey: JsonVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendInvalid() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{FormatKey: InvFormatVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendDefTTL() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{FormatKey: DefinesTTLVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendDuration(duration time.Duration) {
	m.PutsBackend.Duration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordPutBackendError() {
	m.PutsBackend.PutBackendRequests.With(prometheus.Labels{FormatKey: ErrorVal}).Inc()
}

func (m *PrometheusMetrics) RecordPutBackendSize(sizeInBytes float64) {
	m.PutsBackend.RequestLength.Observe(sizeInBytes)
}

func (m *PrometheusMetrics) RecordGetBackendTotal() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{StatusKey: TotalsVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetBackendDuration(duration time.Duration) {
	m.GetsBackend.Duration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordGetBackendError() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{StatusKey: ErrorVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetBackendBadRequest() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{StatusKey: BadRequestVal}).Inc()
}

func (m *PrometheusMetrics) RecordConnectionOpen() {
	m.Connections.ConnectionsOpened.Inc()
}

func (m *PrometheusMetrics) RecordConnectionClosed() {
	m.Connections.ConnectionsClosed.Inc()
}

func (m *PrometheusMetrics) RecordCloseConnectionErrors() {
	m.Connections.ConnectionsErrors.With(prometheus.Labels{ConnErrorKey: CloseVal}).Inc()
}

func (m *PrometheusMetrics) RecordAcceptConnectionErrors() {
	m.Connections.ConnectionsErrors.With(prometheus.Labels{ConnErrorKey: AcceptVal}).Inc()
}

func (m *PrometheusMetrics) RecordExtraTTLSeconds(value float64) {
	m.ExtraTTL.ExtraTTLSeconds.Observe(value)
}

package metrics

import (
	"fmt"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Label keys
	StatusKey    string = "status"
	FormatKey    string = "format"
	ConnErrorKey string = "connection_error"
	TypeKey      string = "type"

	// Label values
	TotalsVal      string = "total"
	ErrorVal       string = "error"
	KeyNotFoundVal string = "key_not_found"
	MissingKeyVal  string = "missing_key"
	BadRequestVal  string = "bad_request"
	JsonVal        string = "json"
	XmlVal         string = "xml"
	CustomKey      string = "custom_key"
	InvFormatVal   string = "invalid_format"
	CloseVal       string = "close"
	AcceptVal      string = "accept"

	// Metric names
	PutRequestMet  string = "puts_request"
	PutReqDurMet   string = "puts_request_duration"
	GetRequestMet  string = "gets_request"
	GetReqDurMet   string = "gets_request_duration"
	PutBackendMet  string = "puts_backend"
	PutBackDurMet  string = "puts_backend_duration"
	PutBackSizeMet string = "puts_backend_request_size_bytes"
	PutTTLSeconds  string = "puts_backend_request_ttl"
	GetBackendMet  string = "gets_backend"
	GetBackendErr  string = "gets_backend_error"
	GetBackDurMet  string = "gets_backend_duration"
	ConnOpenedMet  string = "connection_opened"
	ConnClosedMet  string = "connection_closed"

	MetricsPrometheus = "Prometheus"
)

type PrometheusMetrics struct {
	Registry    *prometheus.Registry
	Puts        *PrometheusRequestStatusMetric
	Gets        *PrometheusRequestStatusMetric
	PutsBackend *PrometheusRequestStatusMetricByFormat
	GetsBackend *PrometheusRequestStatusMetric
	Connections *PrometheusConnectionMetrics
	MetricsName string
}

type PrometheusRequestStatusMetric struct {
	Duration      prometheus.Histogram
	RequestStatus *prometheus.CounterVec
	ErrorsByType  *prometheus.CounterVec
}

type PrometheusRequestStatusMetricByFormat struct {
	Duration           prometheus.Histogram
	PutBackendRequests *prometheus.CounterVec
	RequestLength      prometheus.Histogram
	RequestTTLDuration prometheus.Histogram
}

type PrometheusConnectionMetrics struct {
	ConnectionsErrors *prometheus.CounterVec
	ConnectionsClosed prometheus.Counter
	ConnectionsOpened prometheus.Counter
}

func CreatePrometheusMetrics(cfg config.PrometheusMetrics) *PrometheusMetrics {
	timeBuckets := []float64{0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 1}
	// TTL seconds buckets for 1 second, half a minute as well as one, ten, fifteen, thirty minutes and 1, 2, and 3 and 10 hours
	ttlBuckets := []float64{0.001, 1, 30, 60, 600, 900, 1800, 3600, 7200, 10800, 36000}
	requestSizeBuckets := []float64{0, 4096, 8192, 16384, 32768, 65536, 131072, 262144, 524288, 1048576}
	registry := prometheus.NewRegistry()
	promMetrics := &PrometheusMetrics{
		Registry: registry,
		Puts: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				PutReqDurMet,
				"Duration in seconds Prebid Cache takes to process put requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				PutRequestMet,
				"Count of total requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		Gets: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				GetReqDurMet,
				"Duration in seconds Prebid Cache takes to process get requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				GetRequestMet,
				"Count of total get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		PutsBackend: &PrometheusRequestStatusMetricByFormat{
			Duration: newHistogram(cfg, registry,
				PutBackDurMet,
				"Duration in seconds Prebid Cache takes to process backend put requests.",
				timeBuckets,
			),
			PutBackendRequests: newCounterVecWithLabels(cfg, registry,
				PutBackendMet,
				"Count of total requests to Prebid Cache labeled by format, status and whether or not it comes with TTL",
				[]string{FormatKey},
			),
			RequestLength: newHistogram(cfg, registry,
				PutBackSizeMet,
				"Size in bytes of a backend put request.",
				requestSizeBuckets,
			),
			RequestTTLDuration: newHistogram(cfg, registry,
				PutTTLSeconds,
				"Time-to-live duration in seconds specified in put request body's ttl_seconds field",
				ttlBuckets,
			),
		},
		GetsBackend: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				GetBackDurMet,
				"Duration in seconds Prebid Cache takes to process backend get requests.",
				timeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				GetBackendMet,
				"Count of total backend get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
			ErrorsByType: newCounterVecWithLabels(cfg, registry,
				GetBackendErr,
				"Account for the most frequent type of get errors in the backend",
				[]string{TypeKey},
			),
		},
		Connections: &PrometheusConnectionMetrics{
			ConnectionsClosed: newSingleCounter(cfg, registry, ConnClosedMet, "Count the number of closed connections"),
			ConnectionsOpened: newSingleCounter(cfg, registry, ConnOpenedMet, "Count the number of open connections"),
			ConnectionsErrors: newCounterVecWithLabels(cfg, registry,
				ConnErrorKey,
				"Count the number of connection accept errors or connection close errors",
				[]string{ConnErrorKey},
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

func (m *PrometheusMetrics) RecordPutKeyProvided() {
	m.Puts.RequestStatus.With(prometheus.Labels{StatusKey: CustomKey}).Inc()
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

func (m *PrometheusMetrics) RecordPutBackendDuration(duration time.Duration) {
	m.PutsBackend.Duration.Observe(duration.Seconds())
}

func (m *PrometheusMetrics) RecordPutBackendTTLSeconds(duration time.Duration) {
	m.PutsBackend.RequestTTLDuration.Observe(duration.Seconds())
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

func (m *PrometheusMetrics) RecordKeyNotFoundError() {
	m.GetsBackend.ErrorsByType.With(prometheus.Labels{TypeKey: KeyNotFoundVal}).Inc()
}

func (m *PrometheusMetrics) RecordMissingKeyError() {
	m.GetsBackend.ErrorsByType.With(prometheus.Labels{TypeKey: MissingKeyVal}).Inc()
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

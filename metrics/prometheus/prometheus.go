package metrics

import (
	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

// Constants and global variables
const (
	StatusKey    string = "status"
	FormatKey    string = "format"
	ConnErrorKey string = "connection_error"

	TotalsVal     string = "total"
	ErrorVal      string = "error"
	BadRequestVal string = "bad_request"
	JsonVal       string = "json"
	XmlVal        string = "xml"
	DefinesTTLVal string = "defines_ttl"
	InvFormatVal  string = "invalid_format"
	CloseVal      string = "close"
	AcceptVal     string = "accept"
)

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
	Duration      prometheus.Histogram
	RequestStatus *prometheus.CounterVec
}

type PrometheusRequestStatusMetricByFormat struct {
	Duration           prometheus.Histogram
	PutBackendRequests *prometheus.CounterVec
	RequestLength      prometheus.Histogram
}

type PrometheusConnectionMetrics struct {
	ConnectionsOpened prometheus.Gauge
	ConnectionsErrors *prometheus.CounterVec
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
				"puts_current_url_duration",
				"Duration in seconds Prebid Cache takes to process put requests.",
				cacheWriteTimeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"puts_current_url",
				"Count of total requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		Gets: &PrometheusRequestStatusMetric{
			Duration: newHistogram(cfg, registry,
				"gets_current_url_duration",
				"Duration in seconds Prebid Cache takes to process get requests.",
				cacheWriteTimeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets_current_url",
				"Count of total get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		PutsBackend: &PrometheusRequestStatusMetricByFormat{
			Duration: newHistogram(cfg, registry,
				"puts_backend_duration",
				"Duration in seconds Prebid Cache takes to process backend put requests.",
				cacheWriteTimeBuckets,
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
				cacheWriteTimeBuckets,
			),
			RequestStatus: newCounterVecWithLabels(cfg, registry,
				"gets_backend",
				"Count of total backend get requests to Prebid Server labeled by status.",
				[]string{StatusKey},
			),
		},
		Connections: &PrometheusConnectionMetrics{
			ConnectionsOpened: newGaugeMetric(cfg, registry,
				"connections",
				"Count of total number of connectionsbackend get requests to Prebid Server labeled by status.",
			),
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

//	`CacheMetrics` interface implementation
func (m PrometheusMetrics) Export(cfg config.Metrics) {
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

func (m *PrometheusMetrics) RecordPutDuration(duration *time.Time) {
	m.Puts.Duration.Observe(time.Since(*duration).Seconds())
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

func (m *PrometheusMetrics) RecordGetDuration(duration *time.Time) {
	m.Gets.Duration.Observe(time.Since(*duration).Seconds())
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

func (m *PrometheusMetrics) RecordPutBackendDuration(duration *time.Time) {
	m.PutsBackend.Duration.Observe(time.Since(*duration).Seconds())
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

func (m *PrometheusMetrics) RecordGetBackendDuration(duration *time.Time) {
	m.GetsBackend.Duration.Observe(time.Since(*duration).Seconds())
}

func (m *PrometheusMetrics) RecordGetBackendError() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{StatusKey: ErrorVal}).Inc()
}

func (m *PrometheusMetrics) RecordGetBackendBadRequest() {
	m.GetsBackend.RequestStatus.With(prometheus.Labels{StatusKey: BadRequestVal}).Inc()
}

func (m *PrometheusMetrics) IncreaseOpenConnections() {
	m.Connections.ConnectionsOpened.Inc()
}

func (m *PrometheusMetrics) DecreaseOpenConnections() {
	m.Connections.ConnectionsOpened.Dec()
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

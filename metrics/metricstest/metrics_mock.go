package metricstest

import (
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"time"
)

// Define Mock metrics
var MockHistograms map[string]float64
var MockCounters map[string]int64

func CreateMockMetrics() *metrics.Metrics {
	MockHistograms = make(map[string]float64, 6)
	MockHistograms["puts.current_url.duration"] = 0.00
	MockHistograms["gets.current_url.duration"] = 0.00
	MockHistograms["puts.backends.request_duration"] = 0.00
	MockHistograms["puts.backends.request_size_bytes"] = 0.00
	MockHistograms["gets.backends.duration"] = 0.00
	MockHistograms["connections.connections_opened"] = 0.00
	MockHistograms["extra_ttl_seconds"] = 0.00

	MockCounters = make(map[string]int64, 16)
	MockCounters["puts.current_url.request.total"] = 0
	MockCounters["puts.current_url.request.error"] = 0
	MockCounters["puts.current_url.request.bad_request"] = 0
	MockCounters["gets.current_url.request.total"] = 0
	MockCounters["gets.current_url.request.error"] = 0
	MockCounters["gets.current_url.request.bad_request"] = 0
	MockCounters["puts.backends.add"] = 0
	MockCounters["puts.backends.json"] = 0
	MockCounters["puts.backends.xml"] = 0
	MockCounters["puts.backends.invalid_format"] = 0
	MockCounters["puts.backends.defines_ttl"] = 0
	MockCounters["puts.backends.request.error"] = 0
	MockCounters["puts.backends.request.bad_request"] = 0
	MockCounters["gets.backends.request.total"] = 0
	MockCounters["gets.backends.request.error"] = 0
	MockCounters["gets.backends.request.bad_request"] = 0
	MockCounters["connections.connection_error.accept"] = 0
	MockCounters["connections.connection_error.close"] = 0

	return &metrics.Metrics{MetricEngines: []metrics.CacheMetrics{&MockMetrics{}}}
}

type MockMetrics struct{}

func (m *MockMetrics) Export(cfg config.Metrics) {
}
func (m *MockMetrics) RecordPutError() {
	MockCounters["puts.current_url.request.error"] = MockCounters["puts.current_url.request.error"] + 1
}
func (m *MockMetrics) RecordPutBadRequest() {
	MockCounters["puts.current_url.request.bad_request"] = MockCounters["puts.current_url.request.bad_request"] + 1
}
func (m *MockMetrics) RecordPutTotal() {
	MockCounters["puts.current_url.request.total"] = MockCounters["puts.current_url.request.total"] + 1
}
func (m *MockMetrics) RecordPutDuration(duration *time.Time) {
	MockHistograms["puts.current_url.duration"] = time.Since(*duration).Seconds()
}
func (m *MockMetrics) RecordGetError() {
	MockCounters["gets.current_url.request.error"] = MockCounters["gets.current_url.request.error"] + 1
}
func (m *MockMetrics) RecordGetBadRequest() {
	MockCounters["gets.current_url.request.bad_request"] = MockCounters["gets.current_url.request.bad_request"] + 1
}
func (m *MockMetrics) RecordGetTotal() {
	MockCounters["gets.current_url.request.total"] = MockCounters["gets.current_url.request.total"] + 1
}
func (m *MockMetrics) RecordGetDuration(duration *time.Time) {
	MockHistograms["gets.current_url.duration"] = time.Since(*duration).Seconds()
}
func (m *MockMetrics) RecordPutBackendXml() {
	MockCounters["puts.backends.xml"] = MockCounters["puts.backends.xml"] + 1
}
func (m *MockMetrics) RecordPutBackendJson() {
	MockCounters["puts.backends.json"] = MockCounters["puts.backends.json"] + 1
}
func (m *MockMetrics) RecordPutBackendInvalid() {
	MockCounters["puts.backends.invalid_format"] = MockCounters["puts.backends.invalid_format"] + 1
}
func (m *MockMetrics) RecordPutBackendDefTTL() {
	MockCounters["puts.backends.defines_ttl"] = MockCounters["puts.backends.defines_ttl"] + 1
}
func (m *MockMetrics) RecordPutBackendDuration(duration *time.Time) {
	MockHistograms["puts.backends.request_duration"] = time.Since(*duration).Seconds()
}
func (m *MockMetrics) RecordPutBackendError() {
	MockCounters["puts.backends.request.error"] = MockCounters["puts.backends.request.error"] + 1
}
func (m *MockMetrics) RecordPutBackendSize(sizeInBytes float64) {
	MockHistograms["puts.backends.request_size_bytes"] = sizeInBytes
}
func (m *MockMetrics) RecordGetBackendDuration(duration *time.Time) {
	MockHistograms["gets.backends.duration"] = time.Since(*duration).Seconds()
}
func (m *MockMetrics) RecordGetBackendTotal() {
	MockCounters["gets.backends.request.total"] = MockCounters["gets.backends.request.total"] + 1
}
func (m *MockMetrics) RecordGetBackendError() {
	MockCounters["gets.backends.request.error"] = MockCounters["gets.backends.request.error"] + 1
}
func (m *MockMetrics) IncreaseOpenConnections() {
	MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] + 1
}
func (m *MockMetrics) DecreaseOpenConnections() {
	MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] - 1
}
func (m *MockMetrics) RecordCloseConnectionErrors() {
	MockCounters["connections.connection_error.close"] = MockCounters["connections.connection_error.close"] + 1
}
func (m *MockMetrics) RecordAcceptConnectionErrors() {
	MockCounters["connections.connection_error.accept"] = MockCounters["connections.connection_error.accept"] + 1
}
func (m *MockMetrics) RecordExtraTTLSeconds(value float64) {
	MockHistograms["extra_ttl_seconds"] = value
}

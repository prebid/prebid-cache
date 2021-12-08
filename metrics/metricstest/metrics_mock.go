package metricstest

import (
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
)

const mockDuration time.Duration = time.Second

var MockHistograms map[string]float64
var MockCounters map[string]int64

func CreateMockMetrics() *metrics.Metrics {
	MockHistograms = make(map[string]float64, 6)
	MockHistograms["puts.current_url.duration"] = 0.00
	MockHistograms["gets.current_url.duration"] = 0.00
	MockHistograms["puts.backends.request_duration"] = 0.00
	MockHistograms["puts.backends.request_size_bytes"] = 0.00
	MockHistograms["puts.backends.request_ttl_seconds"] = 0.00
	MockHistograms["gets.backends.duration"] = 0.00
	MockHistograms["connections.connections_opened"] = 0.00

	MockCounters = make(map[string]int64, 16)
	MockCounters["puts.current_url.request.total"] = 0
	MockCounters["puts.current_url.request.error"] = 0
	MockCounters["puts.current_url.request.bad_request"] = 0
	MockCounters["puts.current_url.request.custom_key"] = 0
	MockCounters["gets.current_url.request.total"] = 0
	MockCounters["gets.current_url.request.error"] = 0
	MockCounters["gets.current_url.request.bad_request"] = 0
	MockCounters["puts.backends.add"] = 0
	MockCounters["puts.backends.json"] = 0
	MockCounters["puts.backends.xml"] = 0
	MockCounters["puts.backends.invalid_format"] = 0
	MockCounters["puts.backends.request.error"] = 0
	MockCounters["puts.backends.request.bad_request"] = 0
	MockCounters["gets.backends.request.total"] = 0
	MockCounters["gets.backends.request.error"] = 0
	MockCounters["gets.backends.request.bad_request"] = 0
	MockCounters["gets.backend_error.key_not_found"] = 0
	MockCounters["gets.backend_error.missing_key"] = 0
	MockCounters["connections.connection_error.accept"] = 0
	MockCounters["connections.connection_error.close"] = 0

	return &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&MockMetrics{
				MetricsName: "Mockmetrics",
			},
		},
	}
}

type MockMetrics struct {
	MetricsName string
}

func (m *MockMetrics) Export(cfg config.Metrics) {
}
func (m *MockMetrics) GetEngineRegistry() interface{} {
	return nil
}
func (m *MockMetrics) GetMetricsEngineName() string {
	return ""
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
func (m *MockMetrics) RecordPutDuration(duration time.Duration) {
	MockHistograms["puts.current_url.duration"] = mockDuration.Seconds()
}
func (m *MockMetrics) RecordPutKeyProvided() {
	MockCounters["puts.current_url.request.custom_key"] = MockCounters["puts.current_url.request.custom_key"] + 1
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
func (m *MockMetrics) RecordGetDuration(duration time.Duration) {
	MockHistograms["gets.current_url.duration"] = mockDuration.Seconds()
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
func (m *MockMetrics) RecordPutBackendDuration(duration time.Duration) {
	MockHistograms["puts.backends.request_duration"] = mockDuration.Seconds()
}
func (m *MockMetrics) RecordPutBackendError() {
	MockCounters["puts.backends.request.error"] = MockCounters["puts.backends.request.error"] + 1
}
func (m *MockMetrics) RecordPutBackendSize(sizeInBytes float64) {
	MockHistograms["puts.backends.request_size_bytes"] = sizeInBytes
}
func (m *MockMetrics) RecordPutBackendTTLSeconds(duration time.Duration) {
	MockHistograms["puts.backends.request_ttl_seconds"] = mockDuration.Seconds()
}
func (m *MockMetrics) RecordGetBackendDuration(duration time.Duration) {
	MockHistograms["gets.backends.duration"] = mockDuration.Seconds()
}
func (m *MockMetrics) RecordGetBackendTotal() {
	MockCounters["gets.backends.request.total"] = MockCounters["gets.backends.request.total"] + 1
}
func (m *MockMetrics) RecordGetBackendError() {
	MockCounters["gets.backends.request.error"] = MockCounters["gets.backends.request.error"] + 1
}
func (m *MockMetrics) RecordKeyNotFoundError() {
	MockCounters["gets.backend_error.key_not_found"] = MockCounters["gets.backend_error.key_not_found"] + 1
}
func (m *MockMetrics) RecordMissingKeyError() {
	MockCounters["gets.backend_error.missing_key"] = MockCounters["gets.backend_error.missing_key"] + 1
}
func (m *MockMetrics) RecordConnectionOpen() {
	MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] + 1
}
func (m *MockMetrics) RecordConnectionClosed() {
	MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] - 1
}
func (m *MockMetrics) RecordCloseConnectionErrors() {
	MockCounters["connections.connection_error.close"] = MockCounters["connections.connection_error.close"] + 1
}
func (m *MockMetrics) RecordAcceptConnectionErrors() {
	MockCounters["connections.connection_error.accept"] = MockCounters["connections.connection_error.accept"] + 1
}

package metricstest

import (
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/mock"
)

const mockDuration time.Duration = time.Second

var MockHistograms map[string]float64
var MockCounters map[string]int64

// Rename to NewMockMetricsEngine
func CreateMockMetrics(mockMetrics MockMetrics) *metrics.Metrics {
	//// Put metrics
	//mockMetrics.On("RecordPutTotal")
	//mockMetrics.On("RecordPutKeyProvided")
	//mockMetrics.On("RecordPutBadRequest")
	//mockMetrics.On("RecordPutError")
	//mockMetrics.On("RecordPutDuration", mock.Anything)
	//mockMetrics.On("RecordPutBackendXml")
	//mockMetrics.On("RecordPutBackendJson")
	//mockMetrics.On("RecordPutBackendError")
	//mockMetrics.On("RecordPutBackendInvalid")
	//mockMetrics.On("RecordPutBackendSize", mock.Anything)
	//mockMetrics.On("RecordPutBackendTTLSeconds", mock.Anything)
	//mockMetrics.On("RecordPutBackendDuration", mock.Anything)

	//// Get metrics
	//mockMetrics.On("RecordGetError")
	//mockMetrics.On("RecordGetBadRequest")
	//mockMetrics.On("RecordGetTotal")
	//mockMetrics.On("RecordGetDuration", mock.Anything)
	//mockMetrics.On("RecordGetBackendDuration", mock.Anything)
	//mockMetrics.On("RecordGetBackendTotal")
	//mockMetrics.On("RecordGetBackendError")

	// Other metrics
	//mockMetrics.On("RecordKeyNotFoundError")
	//mockMetrics.On("RecordMissingKeyError")
	//mockMetrics.On("RecordConnectionOpen")
	//mockMetrics.On("RecordConnectionClosed")
	//mockMetrics.On("RecordCloseConnectionErrors")
	//mockMetrics.On("RecordAcceptConnectionErrors")

	return &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
}

type MockMetrics struct {
	mock.Mock
}

func (m *MockMetrics) Export(cfg config.Metrics) {}
func (m *MockMetrics) GetEngineRegistry() interface{} {
	return nil
}
func (m *MockMetrics) GetMetricsEngineName() string {
	return ""
}

func (m *MockMetrics) RecordPutError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBadRequest() {
	//MockCounters["puts.current_url.request.bad_request"] = MockCounters["puts.current_url.request.bad_request"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutTotal() {
	//MockCounters["puts.current_url.request.total"] = MockCounters["puts.current_url.request.total"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutDuration(duration time.Duration) {
	//MockHistograms["puts.current_url.duration"] = float64(mockDuration.Seconds())
	m.Called()
	return
}
func (m *MockMetrics) RecordPutKeyProvided() {
	//MockCounters["puts.current_url.request.custom_key"] = MockCounters["puts.current_url.request.custom_key"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordGetError() {
	//MockCounters["gets.current_url.request.error"] = MockCounters["gets.current_url.request.error"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBadRequest() {
	//MockCounters["gets.current_url.request.bad_request"] = MockCounters["gets.current_url.request.bad_request"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordGetTotal() {
	//MockCounters["gets.current_url.request.total"] = MockCounters["gets.current_url.request.total"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordGetDuration(duration time.Duration) {
	//MockHistograms["gets.current_url.duration"] = mockDuration.Seconds()
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendXml() {
	//MockCounters["puts.backends.xml"] = MockCounters["puts.backends.xml"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendJson() {
	//MockCounters["puts.backends.json"] = MockCounters["puts.backends.json"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendInvalid() {
	//MockCounters["puts.backends.invalid_format"] = MockCounters["puts.backends.invalid_format"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendDuration(duration time.Duration) {
	//MockHistograms["puts.backends.request_duration"] = mockDuration.Seconds()
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendError() {
	//MockCounters["puts.backends.request.error"] = MockCounters["puts.backends.request.error"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendSize(sizeInBytes float64) {
	//MockHistograms["puts.backends.request_size_bytes"] = sizeInBytes
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendTTLSeconds(duration time.Duration) {
	//MockHistograms["puts.backends.request_ttl_seconds"] = mockDuration.Seconds()
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendDuration(duration time.Duration) {
	//MockHistograms["gets.backends.duration"] = mockDuration.Seconds()
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendTotal() {
	//MockCounters["gets.backends.request.total"] = MockCounters["gets.backends.request.total"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendError() {
	//MockCounters["gets.backends.request.error"] = MockCounters["gets.backends.request.error"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordKeyNotFoundError() {
	//MockCounters["gets.backend_error.key_not_found"] = MockCounters["gets.backend_error.key_not_found"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordMissingKeyError() {
	//MockCounters["gets.backend_error.missing_key"] = MockCounters["gets.backend_error.missing_key"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordConnectionOpen() {
	//MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordConnectionClosed() {
	//MockHistograms["connections.connections_opened"] = MockHistograms["connections.connections_opened"] - 1
	m.Called()
	return
}
func (m *MockMetrics) RecordCloseConnectionErrors() {
	//MockCounters["connections.connection_error.close"] = MockCounters["connections.connection_error.close"] + 1
	m.Called()
	return
}
func (m *MockMetrics) RecordAcceptConnectionErrors() {
	//MockCounters["connections.connection_error.accept"] = MockCounters["connections.connection_error.accept"] + 1
	m.Called()
	return
}

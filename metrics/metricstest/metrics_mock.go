package metricstest

import (
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/stretchr/testify/mock"
)

func AssertMetrics(t *testing.T, expectedMetrics []string, actualMetrics MockMetrics) {
	t.Helper()

	// All the names of our metric interface methods
	allMetrics := map[string]struct{}{
		"RecordAcceptConnectionErrors": {},
		"RecordCloseConnectionErrors":  {},
		"RecordConnectionClosed":       {},
		"RecordConnectionOpen":         {},
		"RecordGetBackendDuration":     {},
		"RecordGetBackendError":        {},
		"RecordGetBackendTotal":        {},
		"RecordGetBadRequest":          {},
		"RecordGetDuration":            {},
		"RecordGetError":               {},
		"RecordGetTotal":               {},
		"RecordKeyNotFoundError":       {},
		"RecordMissingKeyError":        {},
		"RecordPutBackendDuration":     {},
		"RecordPutBackendError":        {},
		"RecordPutBackendInvalid":      {},
		"RecordPutBackendJson":         {},
		"RecordPutBackendSize":         {},
		"RecordPutBackendTTLSeconds":   {},
		"RecordPutBackendXml":          {},
		"RecordPutBadRequest":          {},
		"RecordPutDuration":            {},
		"RecordPutError":               {},
		"RecordPutKeyProvided":         {},
		"RecordPutTotal":               {},
	}

	// Assert the metrics found in the expectedMetrics array where called. If a given element is not a known metric, throw error.
	for _, metricName := range expectedMetrics {
		_, exists := allMetrics[metricName]
		if exists {
			actualMetrics.AssertCalled(t, metricName)
			delete(allMetrics, metricName)
		} else {
			t.Errorf("Cannot assert unrecognized metric '%s' was called", metricName)
		}
	}

	// Assert the metrics not found in the expectedMetrics array where not called
	for metricName := range allMetrics {
		actualMetrics.AssertNotCalled(t, metricName)
	}
}

// MetricsRecorded is a structure used to document the exepected metrics to be recorded when running unit tests
type MetricsRecorded struct {
	// Connection metrics
	RecordAcceptConnectionErrors int64 `json:"RecordAcceptConnectionErrors"`
	RecordCloseConnectionErrors  int64 `json:"RecordCloseConnectionErrors"`
	RecordConnectionClosed       int64 `json:"RecordConnectionClosed"`
	RecordConnectionOpen         int64 `json:"RecordConnectionOpen"`

	// Get metrics
	RecordGetBackendDuration float64 `json:"RecordGetBackendDuration"`
	RecordGetBackendError    int64   `json:"RecordGetBackendError"`
	RecordGetBackendTotal    int64   `json:"RecordGetBackendTotal"`
	RecordGetBadRequest      int64   `json:"RecordGetBadRequest"`
	RecordGetDuration        float64 `json:"RecordGetDuration"`
	RecordGetError           int64   `json:"RecordGetError"`
	RecordGetTotal           int64   `json:"RecordGetTotal"`

	// Put metrics
	RecordKeyNotFoundError     int64   `json:"RecordKeyNotFoundError"`
	RecordMissingKeyError      int64   `json:"RecordMissingKeyError"`
	RecordPutBackendDuration   float64 `json:"RecordPutBackendDuration"`
	RecordPutBackendError      int64   `json:"RecordPutBackendError"`
	RecordPutBackendInvalid    int64   `json:"RecordPutBackendInvalid"`
	RecordPutBackendJson       int64   `json:"RecordPutBackendJson"`
	RecordPutBackendSize       float64 `json:"RecordPutBackendSize"`
	RecordPutBackendTTLSeconds float64 `json:"RecordPutBackendTTLSeconds"`
	RecordPutBackendXml        int64   `json:"RecordPutBackendXml"`
	RecordPutBadRequest        int64   `json:"RecordPutBadRequest"`
	RecordPutDuration          float64 `json:"RecordPutDuration"`
	RecordPutError             int64   `json:"RecordPutError"`
	RecordPutKeyProvided       int64   `json:"RecordPutKeyProvided"`
	RecordPutTotal             int64   `json:"RecordPutTotal"`
}

func CreateMockMetrics() MockMetrics {
	mockMetrics := MockMetrics{}

	mockMetrics.On("RecordAcceptConnectionErrors")
	mockMetrics.On("RecordCloseConnectionErrors")
	mockMetrics.On("RecordConnectionClosed")
	mockMetrics.On("RecordConnectionOpen")
	mockMetrics.On("RecordGetBackendDuration", mock.Anything)
	mockMetrics.On("RecordGetBackendError")
	mockMetrics.On("RecordGetBackendTotal")
	mockMetrics.On("RecordGetBadRequest")
	mockMetrics.On("RecordGetDuration", mock.Anything)
	mockMetrics.On("RecordGetError")
	mockMetrics.On("RecordGetTotal")
	mockMetrics.On("RecordKeyNotFoundError")
	mockMetrics.On("RecordMissingKeyError")
	mockMetrics.On("RecordPutBackendDuration", mock.Anything)
	mockMetrics.On("RecordPutBackendError")
	mockMetrics.On("RecordPutBackendInvalid")
	mockMetrics.On("RecordPutBackendJson")
	mockMetrics.On("RecordPutBackendSize", mock.Anything)
	mockMetrics.On("RecordPutBackendTTLSeconds", mock.Anything)
	mockMetrics.On("RecordPutBackendXml")
	mockMetrics.On("RecordPutBadRequest")
	mockMetrics.On("RecordPutDuration", mock.Anything)
	mockMetrics.On("RecordPutError")
	mockMetrics.On("RecordPutKeyProvided")
	mockMetrics.On("RecordPutTotal")

	return mockMetrics
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
	m.Called()
	return
}
func (m *MockMetrics) RecordPutTotal() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutDuration(duration time.Duration) {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutKeyProvided() {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBadRequest() {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetTotal() {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetDuration(duration time.Duration) {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendXml() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendJson() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendInvalid() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendDuration(duration time.Duration) {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendSize(sizeInBytes float64) {
	m.Called()
	return
}
func (m *MockMetrics) RecordPutBackendTTLSeconds(duration time.Duration) {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendDuration(duration time.Duration) {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendTotal() {
	m.Called()
	return
}
func (m *MockMetrics) RecordGetBackendError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordKeyNotFoundError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordMissingKeyError() {
	m.Called()
	return
}
func (m *MockMetrics) RecordConnectionOpen() {
	m.Called()
	return
}
func (m *MockMetrics) RecordConnectionClosed() {
	m.Called()
	return
}
func (m *MockMetrics) RecordCloseConnectionErrors() {
	m.Called()
	return
}
func (m *MockMetrics) RecordAcceptConnectionErrors() {
	m.Called()
	return
}

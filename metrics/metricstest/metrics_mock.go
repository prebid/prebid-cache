package metricstest

import (
	"reflect"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/mock"
)

func AssertMetrics(t *testing.T, expectedMetrics []string, actualMetrics MockMetrics, testFile string) {
	t.Helper()

	m := metrics.Metrics{}
	mt := reflect.TypeOf(m)
	allMetricsNames := make(map[string]struct{}, mt.NumMethod())
	metricsLogged := make(map[string]struct{}, mt.NumMethod())

	// List methods of the Metrics interface into map
	for i := 0; i < mt.NumMethod(); i++ {
		allMetricsNames[mt.Method(i).Name] = struct{}{}
	}

	// Assert the metrics found in the expectedMetrics array where called. If a given element is not a known metric, throw error.
	for _, metricName := range expectedMetrics {
		_, exists := allMetricsNames[metricName]
		if exists {
			actualMetrics.AssertCalled(t, metricName)
			metricsLogged[metricName] = struct{}{}
		} else {
			t.Errorf("%s. Cannot assert unrecognized metric '%s' was called", testFile, metricName)
		}
	}

	// Assert the metrics not found in the expectedMetrics array where not called
	for metric := range allMetricsNames {
		// Assert that metrics not found in metricsLogged were effectively not logged
		if _, metricWasLogged := metricsLogged[metric]; !metricWasLogged {
			actualMetrics.AssertNotCalled(t, metric)
		}
	}
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

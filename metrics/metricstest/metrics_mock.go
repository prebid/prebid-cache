package metricstest

import (
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/stretchr/testify/mock"
)

func AssertMetrics(t *testing.T, expectedMetrics MetricsRecorded, actualMetrics MockMetrics) {
	t.Helper()
	if expectedMetrics.RecordAcceptConnectionErrors > 0 {
		actualMetrics.AssertCalled(t, "RecordAcceptConnectionErrors")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordAcceptConnectionErrors")
	}
	if expectedMetrics.RecordCloseConnectionErrors > 0 {
		actualMetrics.AssertCalled(t, "RecordCloseConnectionErrors")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordCloseConnectionErrors")
	}
	if expectedMetrics.RecordConnectionClosed > 0 {
		actualMetrics.AssertCalled(t, "RecordConnectionClosed")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordConnectionClosed")
	}
	if expectedMetrics.RecordConnectionOpen > 0 {
		actualMetrics.AssertCalled(t, "RecordConnectionOpen")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordConnectionOpen")
	}
	if expectedMetrics.RecordGetBackendDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordGetBackendDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendDuration")
	}
	if expectedMetrics.RecordGetBackendError > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBackendError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendError")
	}
	if expectedMetrics.RecordGetBackendTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBackendTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendTotal")
	}
	if expectedMetrics.RecordGetBadRequest > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBadRequest")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBadRequest")
	}
	if expectedMetrics.RecordGetDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordGetDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetDuration")
	}
	if expectedMetrics.RecordGetError > 0 {
		actualMetrics.AssertCalled(t, "RecordGetError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetError")
	}
	if expectedMetrics.RecordGetTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordGetTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetTotal")
	}
	if expectedMetrics.RecordKeyNotFoundError > 0 {
		actualMetrics.AssertCalled(t, "RecordKeyNotFoundError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordKeyNotFoundError")
	}
	if expectedMetrics.RecordMissingKeyError > 0 {
		actualMetrics.AssertCalled(t, "RecordMissingKeyError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordMissingKeyError")
	}
	if expectedMetrics.RecordPutBackendDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendDuration")
	}
	if expectedMetrics.RecordPutBackendError > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendError")
	}
	if expectedMetrics.RecordPutBackendInvalid > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendInvalid")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendInvalid")
	}
	if expectedMetrics.RecordPutBackendJson > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendJson")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendJson")
	}
	if expectedMetrics.RecordPutBackendSize > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendSize")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendSize")
	}
	if expectedMetrics.RecordPutBackendTTLSeconds > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendTTLSeconds")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendTTLSeconds")
	}
	if expectedMetrics.RecordPutBackendXml > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendXml")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendXml")
	}
	if expectedMetrics.RecordPutBadRequest > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBadRequest")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBadRequest")
	}
	if expectedMetrics.RecordPutDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutDuration")
	}
	if expectedMetrics.RecordPutError > 0 {
		actualMetrics.AssertCalled(t, "RecordPutError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutError")
	}
	if expectedMetrics.RecordPutKeyProvided > 0 {
		actualMetrics.AssertCalled(t, "RecordPutKeyProvided")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutKeyProvided")
	}
	if expectedMetrics.RecordPutTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordPutTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutTotal")
	}
}

// MetricsRecorded is a structure used to document the exepected metrics to be recorded when running unit tests
type MetricsRecorded struct {
	// Connection metrics
	RecordAcceptConnectionErrors int64 `json:"acceptConnectionErrors"`
	RecordCloseConnectionErrors  int64 `json:"closeConnectionErrors"`
	RecordConnectionClosed       int64 `json:"connectionClosed"`
	RecordConnectionOpen         int64 `json:"connectionOpen"`

	// Get metrics
	RecordGetBackendDuration float64 `json:"recordGetBackendDuration"`
	RecordGetBackendError    int64   `json:"recordGetBackendError"`
	RecordGetBackendTotal    int64   `json:"recordGetBackendTotal"`
	RecordGetBadRequest      int64   `json:"recordGetBadrequest"`
	RecordGetDuration        float64 `json:"recordGetDuration"`
	RecordGetError           int64   `json:"recordGetError"`
	RecordGetTotal           int64   `json:"recordGetTotal"`

	// Put metrics
	RecordKeyNotFoundError     int64   `json:"keyNotFoundError"`
	RecordMissingKeyError      int64   `json:"missingKeyError"`
	RecordPutBackendDuration   float64 `json:"putBackendDuration"`
	RecordPutBackendError      int64   `json:"putBackendError"`
	RecordPutBackendInvalid    int64   `json:"putBackendInvalid"`
	RecordPutBackendJson       int64   `json:"totalJsonRequests"`
	RecordPutBackendSize       float64 `json:"putBackendSize"`
	RecordPutBackendTTLSeconds float64 `json:"putBackendTTLSeconds"`
	RecordPutBackendXml        int64   `json:"totalXmlRequests"`
	RecordPutBadRequest        int64   `json:"putBadRequest"`
	RecordPutDuration          float64 `json:"putDuration"`
	RecordPutError             int64   `json:"putError"`
	RecordPutKeyProvided       int64   `json:"putKeyProvided"`
	RecordPutTotal             int64   `json:"putTotal"`
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

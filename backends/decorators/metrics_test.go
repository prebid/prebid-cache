package decorators

import (
	"context"
	"errors"
	"testing"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type failedBackend struct {
	returnError error
}

func (b *failedBackend) Get(ctx context.Context, key string) (string, error) {
	return "", b.returnError
}

func (b *failedBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return b.returnError
}

func createMockMetrics() metricstest.MockMetrics {
	mockMetrics := metricstest.MockMetrics{}
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

type metricsRecorded struct {
	//// Put metrics
	//RecordPutTotal             int64   `json:"putTotal"`
	//RecordPutKeyProvided       int64   `json:"putKeyProvided"`
	//RecordPutBackendXml        int64   `json:"totalXmlRequests"`
	//RecordPutBackendJson       int64   `json:"totalJsonRequests"`
	//RecordPutBadRequest        int64   `json:"putBadRequest"`
	//RecordPutError             int64   `json:"putError"`
	//RecordPutBackendError      int64   `json:"putBackendError"`
	//RecordPutBackendInvalid    int64   `json:"putBackendInvalid"`
	//RecordPutDuration          float64 `json:"putDuration"`
	//RecordPutBackendSize       float64 `json:"putBackendSize"`
	//RecordPutBackendTTLSeconds float64 `json:"putBackendTTLSeconds"`
	//RecordPutBackendDuration   float64 `json:"putBackendDuration"`

	//// Get metrics
	//RecordGetError           int64   `json:"recordGetError"`
	//RecordGetBadRequest      int64   `json:"recordGetBadrequest"`
	//RecordGetTotal           int64   `json:"recordGetTotal"`
	//RecordGetDuration        float64 `json:"recordGetDuration"`
	//RecordGetBackendDuration float64 `json:"recordGetBackendDuration"`
	//RecordGetBackendTotal    int64   `json:"recordGetBackendTotal"`
	//RecordGetBackendError    int64   `json:"recordGetBackendError"`
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

func assertMetrics(t *testing.T, expectedMetrics metricsRecorded, actualMetrics metricstest.MockMetrics) {
	t.Helper()

	//if expectedMetrics.RecordPutTotal > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutTotal")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutTotal")
	//}
	//if expectedMetrics.RecordPutKeyProvided > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutKeyProvided")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutKeyProvided")
	//}
	//if expectedMetrics.RecordPutBadRequest > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBadRequest")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBadRequest")
	//}
	//if expectedMetrics.RecordPutError > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutError")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutError")
	//}
	//if expectedMetrics.RecordPutDuration > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutDuration")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutDuration")
	//}
	//if expectedMetrics.RecordPutBackendXml > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendXml")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendXml")
	//}
	//if expectedMetrics.RecordPutBackendJson > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendJson")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendJson")
	//}
	//if expectedMetrics.RecordPutBackendError > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendError")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendError")
	//}
	//if expectedMetrics.RecordPutBackendInvalid > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendInvalid")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendInvalid")
	//}
	//if expectedMetrics.RecordPutBackendSize > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendSize")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendSize")
	//}
	//if expectedMetrics.RecordPutBackendTTLSeconds > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendTTLSeconds")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendTTLSeconds")
	//}
	//if expectedMetrics.RecordPutBackendDuration > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendDuration")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendDuration")
	//}
	// ---
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

func TestGetBackendMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricsRecorded{
		RecordGetBackendTotal:    1,
		RecordGetBackendDuration: 1.00,
	}

	// Test setup
	mockMetrics := createMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}

	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "xml<vast></vast>", 0)
	backendWithMetrics := LogMetrics(rawBackend, m)

	// Run test
	backendWithMetrics.Get(context.Background(), "foo")

	// Assert
	assertMetrics(t, expectedMetrics, mockMetrics)
}

func TestGetBackendErrorMetrics(t *testing.T) {

	type testCase struct {
		desc            string
		expectedMetrics metricsRecorded
		expectedError   error
	}
	testGroups := []struct {
		name  string
		tests []testCase
	}{
		{
			"Special backend storage GET errors",
			[]testCase{
				{
					"Failed get backend request should be accounted as a key not found error",
					metricsRecorded{
						RecordGetBackendError:  1,
						RecordKeyNotFoundError: 1,
						RecordGetBackendTotal:  1,
					},
					utils.NewPBCError(utils.KEY_NOT_FOUND),
				},
				{
					"Failed get backend request should be accounted as a missing key (uuid) error",
					metricsRecorded{
						RecordGetBackendError: 1,
						RecordMissingKeyError: 1,
						RecordGetBackendTotal: 1,
					},
					utils.NewPBCError(utils.MISSING_KEY),
				},
			},
		},
		{
			"Other backend error",
			[]testCase{
				{
					"Failed get backend request should be accounted under the error label",
					metricsRecorded{
						RecordGetBackendError: 1,
						RecordGetBackendTotal: 1,
					},
					errors.New("some backend storage service error"),
				},
			},
		},
	}

	for _, group := range testGroups {
		for _, test := range group.tests {
			// Fresh mock metrics
			mockMetrics := createMockMetrics()
			m := &metrics.Metrics{
				MetricEngines: []metrics.CacheMetrics{
					&mockMetrics,
				},
			}
			// Create backend with a mock storage that will fail and record metrics
			backend := LogMetrics(&failedBackend{test.expectedError}, m)

			// Run test
			retrievedValue, err := backend.Get(context.Background(), "foo")

			// Assertions
			assert.Empty(t, retrievedValue, "%s - %s", group.name, test.desc)
			assert.Equal(t, test.expectedError, err, "%s - %s", group.name, test.desc)
			assertMetrics(t, test.expectedMetrics, mockMetrics)
		}
	}
}

func TestPutSuccessMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricsRecorded{
		RecordPutBackendDuration:   1.00,
		RecordPutBackendXml:        1,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendSize:       1,
	}

	// Test setup
	mockMetrics := createMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 60)

	// Assert
	assertMetrics(t, expectedMetrics, mockMetrics)
}

func TestPutErrorMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricsRecorded{
		RecordPutBackendError:      1,
		RecordPutBackendXml:        1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
	}

	// Test setup
	mockMetrics := createMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(&failedBackend{errors.New("Failure")}, m)

	// Run test
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	// Assert
	assertMetrics(t, expectedMetrics, mockMetrics)
}

func TestJsonPayloadMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricsRecorded{
		RecordPutBackendJson:       1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendDuration:   1.00,
	}

	// Test setup
	mockMetrics := createMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "json{\"key\":\"value\"", 0)

	// Assert
	assertMetrics(t, expectedMetrics, mockMetrics)
}

func TestInvalidPayloadMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricsRecorded{
		RecordPutBackendInvalid:    1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendDuration:   1.00,
	}

	// Test setup
	mockMetrics := createMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "bar", 0)

	// Assert
	assertMetrics(t, expectedMetrics, mockMetrics)
}

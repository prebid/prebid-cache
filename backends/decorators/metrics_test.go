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

func TestGetBackendMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricstest.MetricsRecorded{
		RecordGetBackendTotal:    1,
		RecordGetBackendDuration: 1.00,
	}

	// Test setup
	mockMetrics := metricstest.CreateMockMetrics()
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
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestGetBackendErrorMetrics(t *testing.T) {

	type testCase struct {
		desc            string
		expectedMetrics metricstest.MetricsRecorded
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
					metricstest.MetricsRecorded{
						RecordGetBackendError:  1,
						RecordKeyNotFoundError: 1,
						RecordGetBackendTotal:  1,
					},
					utils.NewPBCError(utils.KEY_NOT_FOUND),
				},
				{
					"Failed get backend request should be accounted as a missing key (uuid) error",
					metricstest.MetricsRecorded{
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
					metricstest.MetricsRecorded{
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
			mockMetrics := metricstest.CreateMockMetrics()
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
			metricstest.AssertMetrics(t, test.expectedMetrics, mockMetrics)
		}
	}
}

func TestPutSuccessMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricstest.MetricsRecorded{
		RecordPutBackendDuration:   1.00,
		RecordPutBackendXml:        1,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendSize:       1,
	}

	// Test setup
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 60)

	// Assert
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestPutErrorMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricstest.MetricsRecorded{
		RecordPutBackendError:      1,
		RecordPutBackendXml:        1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
	}

	// Test setup
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(&failedBackend{errors.New("Failure")}, m)

	// Run test
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	// Assert
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestJsonPayloadMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricstest.MetricsRecorded{
		RecordPutBackendJson:       1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendDuration:   1.00,
	}

	// Test setup
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "json{\"key\":\"value\"", 0)

	// Assert
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

func TestInvalidPayloadMetrics(t *testing.T) {
	// Expected values
	expectedMetrics := metricstest.MetricsRecorded{
		RecordPutBackendInvalid:    1,
		RecordPutBackendSize:       1.00,
		RecordPutBackendTTLSeconds: 1.00,
		RecordPutBackendDuration:   1.00,
	}

	// Test setup
	mockMetrics := metricstest.CreateMockMetrics()
	m := &metrics.Metrics{
		MetricEngines: []metrics.CacheMetrics{
			&mockMetrics,
		},
	}
	backend := LogMetrics(backends.NewMemoryBackend(), m)

	// Run test
	backend.Put(context.Background(), "foo", "bar", 0)

	// Assert
	metricstest.AssertMetrics(t, expectedMetrics, mockMetrics)
}

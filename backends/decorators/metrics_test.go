package decorators

import (
	"context"
	"errors"
	"testing"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/prebid/prebid-cache/utils"
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

func TestGetSuccessMetrics(t *testing.T) {
	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "xml<vast></vast>", 0)
	backend := LogMetrics(rawBackend, m)
	backend.Get(context.Background(), "foo")

	mockMetrics.AssertCalled(t, "RecordGetTotal")
	mockMetrics.AssertCalled(t, "RecordGetDuration")
	//assert.Equalf(t, int64(1), metricstest.MockCounters["gets.backends.request.total"], "Successful backend request been accounted for in the total get backend request count")
	//assert.Greater(t, metricstest.MockHistograms["gets.backends.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestGetErrorMetrics(t *testing.T) {

	type testCase struct {
		desc         string
		inMetricName string
		outError     error
	}
	testGroups := []struct {
		groupName string
		tests     []testCase
	}{
		{
			"Any other error",
			[]testCase{
				{
					"Failed get backend request should be accounted under the error label",
					"RecordGetBackendError",
					errors.New("other backend error"),
				},
			},
		},
		{
			"Special errors",
			[]testCase{
				{
					"Failed get backend request should be accounted as a key not found error",
					"RecordKeyNotFoundError",
					utils.NewPBCError(utils.KEY_NOT_FOUND),
				},
				{
					"Failed get backend request should be accounted as a missing key (uuid) error",
					"RecordMissingKeyError",
					utils.NewPBCError(utils.MISSING_KEY),
				},
			},
		},
	}

	// Create metrics
	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)

	errsTotal := 0
	for _, group := range testGroups {
		for _, test := range group.tests {
			// Create backend with a mock storage that will fail and assign metrics
			backend := LogMetrics(&failedBackend{test.outError}, m)

			// Run test
			backend.Get(context.Background(), "foo")
			errsTotal++

			// Assert
			mockMetrics.AssertCalled(t, test.inMetricName)
			mockMetrics.AssertCalled(t, "RecordGetBackendError")
			mockMetrics.AssertCalled(t, "RecordGetTotal")
		}
	}
}

func TestPutSuccessMetrics(t *testing.T) {

	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 60)

	mockMetrics.AssertCalled(t, "RecordPutBackendDuration")
	mockMetrics.AssertCalled(t, "RecordPutBackendXml")
	mockMetrics.AssertCalled(t, "RecordPutBackendTTLSeconds")
}

func TestPutErrorMetrics(t *testing.T) {

	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	backend := LogMetrics(&failedBackend{errors.New("Failure")}, m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	mockMetrics.AssertCalled(t, "RecordPutBackendXml")
	mockMetrics.AssertCalled(t, "RecordPutBackendError")
}

func TestJsonPayloadMetrics(t *testing.T) {

	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "json{\"key\":\"value\"", 0)
	backend.Get(context.Background(), "foo")

	mockMetrics.AssertCalled(t, "RecordPutBackendJson")
}

func TestPutSizeSampling(t *testing.T) {

	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	payload := `json{"key":"value"}`
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", payload, 0)

	mockMetrics.AssertCalled(t, "RecordPutBackendSize")
}

func TestInvalidPayloadMetrics(t *testing.T) {

	mockMetrics := metricstest.MockMetrics{}
	m := metricstest.CreateMockMetrics(mockMetrics)
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "bar", 0)
	backend.Get(context.Background(), "foo")

	mockMetrics.AssertCalled(t, "RecordPutBackendInvalid")
}

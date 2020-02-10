package decorators

import (
	"context"
	"fmt"
	"testing"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/assert"
)

type failedBackend struct{}

func (b *failedBackend) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("Failure")
}

func (b *failedBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return fmt.Errorf("Failure")
}

func TestGetSuccessMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "xml<vast></vast>", 0)
	backend := LogMetrics(rawBackend, m)
	backend.Get(context.Background(), "foo")

	assert.Equalf(t, int64(1), metrics.HT2["gets.backends.request.total"], "Successful backend request been accounted for in the total get backend request count")
	assert.Greater(t, metrics.HT1["gets.backends.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestGetErrorMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Get(context.Background(), "foo")

	assert.Equal(t, int64(1), metrics.HT2["gets.backends.request.error"], "Failed get backend request should have been accounted under the error label")
	assert.Equal(t, int64(1), metrics.HT2["gets.backends.request.total"], "Failed get backend request should have been accounted in the request totals")
}

func TestPutSuccessMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	assert.Greater(t, metrics.HT1["puts.backends.request_duration"], 0.00, "Successful put request duration should be greater than zero")
	assert.Equal(t, int64(1), metrics.HT2["puts.backends.xml"], "An xml request should have been logged.")
	assert.Equal(t, int64(0), metrics.HT2["puts.backends.defines_ttl"], "An event for TTL defined shouldn't be logged if the TTL was 0")
}

func TestTTLDefinedMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 1)

	assert.Equal(t, int64(1), metrics.HT2["puts.backends.defines_ttl"], "An event for TTL defined shouldn't be logged if the TTL was 0")
}

func TestPutErrorMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	assert.Equal(t, int64(1), metrics.HT2["puts.backends.xml"], "An xml request should have been logged.")
	assert.Equal(t, int64(1), metrics.HT2["puts.backends.request.error"], "Failed get backend request should have been accounted under the error label")
}

func TestJsonPayloadMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "json{\"key\":\"value\"", 0)
	backend.Get(context.Background(), "foo")

	assert.Equal(t, int64(1), metrics.HT2["puts.backends.json"], "A json request should have been logged.")
}

func TestPutSizeSampling(t *testing.T) {
	m := metrics.CreateMockMetrics()
	payload := `json{"key":"value"}`
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", payload, 0)

	assert.Greater(t, metrics.HT1["puts.backends.request_size_bytes"], 0.00, "Successful put request size should be greater than zero")
}

func TestInvalidPayloadMetrics(t *testing.T) {
	m := metrics.CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "bar", 0)
	backend.Get(context.Background(), "foo")

	assert.Equal(t, int64(1), metrics.HT2["puts.backends.invalid_format"], "A Put request of invalid format should have been logged.")
}

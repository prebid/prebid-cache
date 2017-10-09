package decorators

import (
	"context"
	"fmt"
	"github.com/Prebid-org/prebid-cache/backends"
	"github.com/Prebid-org/prebid-cache/metrics"
	"github.com/Prebid-org/prebid-cache/metrics/metricstest"
	"testing"
)

type failedBackend struct{}

func (b *failedBackend) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("Failure")
}

func (b *failedBackend) Put(ctx context.Context, key string, value string) error {
	return fmt.Errorf("Failure")
}

func TestGetSuccessMetrics(t *testing.T) {
	m := metrics.CreateMetrics()
	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "bar")
	backend := LogMetrics(rawBackend, m)
	backend.Get(context.Background(), "foo")

	metricstest.AssertSuccessMetricsExist(t, m.GetsBackend)
}

func TestGetErrorMetrics(t *testing.T) {
	m := metrics.CreateMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Get(context.Background(), "foo")

	metricstest.AssertErrorMetricsExist(t, m.GetsBackend)
}

func TestPutSuccessMetrics(t *testing.T) {
	m := metrics.CreateMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "bar")

	metricstest.AssertSuccessMetricsExist(t, m.PutsBackend)
}

func TestPutErrorMetrics(t *testing.T) {
	m := metrics.CreateMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Put(context.Background(), "foo", "bar")

	if m.PutsBackend.Request.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
	if m.PutsBackend.Duration.Count() != 0 {
		t.Errorf("The request duration should not have been counted.")
	}
	if m.PutsBackend.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if m.PutsBackend.Errors.Count() != 1 {
		t.Errorf("An Error should have been counted.")
	}
}

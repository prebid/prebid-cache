package metrics

import (
	"context"
	"fmt"
	"github.com/Prebid-org/prebid-cache/backends"
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
	metrics := CreateMetrics()
	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "bar")
	backend := MonitorBackend(rawBackend, metrics)
	backend.Get(context.Background(), "foo")

	assertSuccessMetricsExist(t, metrics.GetsBackend)
}

func TestGetErrorMetrics(t *testing.T) {
	metrics := CreateMetrics()
	backend := MonitorBackend(&failedBackend{}, metrics)
	backend.Get(context.Background(), "foo")

	assertErrorMetricsExist(t, metrics.GetsBackend)
}

func TestPutSuccessMetrics(t *testing.T) {
	metrics := CreateMetrics()
	backend := MonitorBackend(backends.NewMemoryBackend(), metrics)
	backend.Put(context.Background(), "foo", "bar")

	assertSuccessMetricsExist(t, metrics.PutsBackend)
}

func TestPutErrorMetrics(t *testing.T) {
	metrics := CreateMetrics()
	backend := MonitorBackend(&failedBackend{}, metrics)
	backend.Put(context.Background(), "foo", "bar")

	if metrics.PutsBackend.Request.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
	if metrics.PutsBackend.Duration.Count() != 0 {
		t.Errorf("The request duration should not have been counted.")
	}
	if metrics.PutsBackend.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if metrics.PutsBackend.Errors.Count() != 1 {
		t.Errorf("An Error should have been counted.")
	}
}

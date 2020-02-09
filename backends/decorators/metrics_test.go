package decorators

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/assert"
	//"github.com/prebid/prebid-cache/metrics/metricstest"
)

type failedBackend struct{}

func (b *failedBackend) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("Failure")
}

func (b *failedBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return fmt.Errorf("Failure")
}

func TestGetSuccessMetrics(t *testing.T) {
	m := CreateMockMetrics()
	rawBackend := backends.NewMemoryBackend()
	rawBackend.Put(context.Background(), "foo", "xml<vast></vast>", 0)
	backend := LogMetrics(rawBackend, m)
	backend.Get(context.Background(), "foo")

	//metricstest.AssertSuccessMetricsExist(t, m.GetsBackend)
	actualRequestDuration, _ := HT1["gets.backends.duration"]
	actualRequestCount, _ := HT2["gets.backends.request.total"]
	assert.Equalf(t, int64(1), actualRequestCount, "Successful backend request been accounted for in the total get backend request count, expected = 1; actual = %d\n", actualRequestCount)
	assert.Greater(t, actualRequestDuration, 0.00, "Successful put request duration should be greater than zero")
}

func TestGetErrorMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Get(context.Background(), "foo")

	//metricstest.AssertErrorMetricsExist(t, m.GetsBackend)
	actualErrorCount, _ := HT2["gets.backends.request.error"]
	actualRequestCount, _ := HT2["gets.backends.request.total"]
	assert.Equal(t, int64(1), actualErrorCount, "Failed get backend request should have been accounted under the error label")
	assert.Equal(t, int64(1), actualRequestCount, "Failed get backend request should have been accounted in the request totals")
}

func TestPutSuccessMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	//assertSuccessMetricsExist(t, m.PutsBackend)
	actualRequestDuration, _ := HT1["puts.backend.duration"]
	actualRequestCount, _ := HT2["puts.backends.request.total"]
	actualXMLRequestCount, _ := HT2["puts.backends.xml"]
	actualTTLRequestCount, _ := HT2["puts.backends.defines_ttl"]

	assert.Equal(t, int64(1), actualRequestCount, "Successful put backend request should have been accounted in the request totals")
	assert.Greater(t, actualRequestDuration, 0.00, "Successful put request duration should be greater than zero")
	assert.Equal(t, int64(1), actualXMLRequestCount, "An xml request should have been logged.")
	assert.Equal(t, int64(0), actualTTLRequestCount, "An event for TTL defined shouldn't be logged if the TTL was 0")
}

/*
func TestTTLDefinedMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 1)
	if m.PutsBackend.DefinesTTL.Count() != 1 {
		t.Errorf("An event for TTL defined should be logged if the TTL is not 0")
	}
}

func TestPutErrorMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(&failedBackend{}, m)
	backend.Put(context.Background(), "foo", "xml<vast></vast>", 0)

	assertErrorMetricsExist(t, m.PutsBackend)
	if m.PutsBackend.XmlRequest.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
}

func TestJsonPayloadMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "json{\"key\":\"value\"", 0)
	backend.Get(context.Background(), "foo")

	if m.PutsBackend.JsonRequest.Count() != 1 {
		t.Errorf("A json Put should have been logged.")
	}
}

func TestPutSizeSampling(t *testing.T) {
	m := CreateMockMetrics()
	payload := `json{"key":"value"}`
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", payload, 0)

	if m.PutsBackend.RequestLength.Count() != 1 {
		t.Errorf("A request size sample should have been logged.")
	}
}

func TestInvalidPayloadMetrics(t *testing.T) {
	m := CreateMockMetrics()
	backend := LogMetrics(backends.NewMemoryBackend(), m)
	backend.Put(context.Background(), "foo", "bar", 0)
	backend.Get(context.Background(), "foo")

	if m.PutsBackend.InvalidRequest.Count() != 1 {
		t.Errorf("A Put request of invalid format should have been logged.")
	}
}

func assertSuccessMetricsExist(t *testing.T, entry *metrics.MetricsEntryByFormat) {
	t.Helper()
	if entry.Duration.Count() != 1 {
		t.Errorf("The request duration should have been counted.")
	}
	if entry.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if entry.Errors.Count() != 0 {
		t.Errorf("No Errors should have been counted.")
	}
}

func assertErrorMetricsExist(t *testing.T, entry *metrics.MetricsEntryByFormat) {
	t.Helper()
	if entry.Duration.Count() != 0 {
		t.Errorf("The request duration should not have been counted.")
	}
	if entry.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if entry.Errors.Count() != 1 {
		t.Errorf("An Error should have been counted.")
	}
}
*/
/*Define Mock metrics        */
var HT1 map[string]float64

//var HT1 map[string]float64 = map[string]float64{
//	"puts.current_url.duration":        0.00,
//	"gets.current_url.duration":        0.00,
//	"puts.backends.request_duration":   0.00,
//	"puts.backends.request_size_bytes": 0.00,
//	"gets.backends.duration":           0.00,
//	"connections.connections_opened":   0.00,
//	"extra_ttl_seconds":                0.00,
//}

var HT2 map[string]int64

//var HT2 map[string]int64 = map[string]int64{
//	"puts.current_url.request.total":       0,
//	"puts.current_url.request.error":       0,
//	"puts.current_url.request.bad_request": 0,
//	"gets.current_url.request.total":       0,
//	"gets.current_url.request.error":       0,
//	"gets.current_url.request.bad_request": 0,
//	"puts.backends.add":                    0,
//	"puts.backends.json":                   0,
//	"puts.backends.xml":                    0,
//	"puts.backends.invalid_format":         0,
//	"puts.backends.defines_ttl":            0,
//	"puts.backends.request.error":          0,
//	"gets.backends.request.total":          0,
//	"gets.backends.request.error":          0,
//	"gets.backends.request.bad_request":    0,
//	"connections.connection_error.accept":  0,
//	"connections.connection_error.close":   0,
//}

func CreateMockMetrics() *metrics.Metrics {
	HT1 = make(map[string]float64, 6)
	HT1["puts.current_url.duration"] = 0.00
	HT1["gets.current_url.duration"] = 0.00
	HT1["puts.backends.request_duration"] = 0.00
	HT1["puts.backends.request_size_bytes"] = 0.00
	HT1["gets.backends.duration"] = 0.00
	HT1["connections.connections_opened"] = 0.00
	HT1["extra_ttl_seconds"] = 0.00

	HT2 = make(map[string]int64, 16)
	HT2["puts.current_url.request.total"] = 0
	HT2["puts.current_url.request.error"] = 0
	HT2["puts.current_url.request.bad_request"] = 0
	HT2["gets.current_url.request.total"] = 0
	HT2["gets.current_url.request.error"] = 0
	HT2["gets.current_url.request.bad_request"] = 0
	HT2["puts.backends.add"] = 0
	HT2["puts.backends.json"] = 0
	HT2["puts.backends.xml"] = 0
	HT2["puts.backends.invalid_format"] = 0
	HT2["puts.backends.defines_ttl"] = 0
	HT2["puts.backends.request.error"] = 0
	HT2["gets.backends.request.total"] = 0
	HT2["gets.backends.request.error"] = 0
	HT2["gets.backends.request.bad_request"] = 0
	HT2["connections.connection_error.accept"] = 0
	HT2["connections.connection_error.close"] = 0

	return &metrics.Metrics{MetricEngines: []metrics.CacheMetrics{&MockMetrics{}}}
}

type MockMetrics struct{}

func (m *MockMetrics) RecordPutRequest(status string, duration *time.Time) {
	if duration != nil {
		HT1["puts.current_url.duration"] = time.Since(*duration).Seconds()
	} else {
		switch status {
		case "add":
			HT2["puts.current_url.request.total"] = HT2["puts.current_url.request.total"] + 1
		case "error":
			HT2["puts.current_url.request.error"] = HT2["puts.current_url.request.error"] + 1
		case "bad_request":
			HT2["puts.current_url.request.bad_request"] = HT2["puts.current_url.request.bad_request"] + 1
		}
	}
}

func (m *MockMetrics) RecordGetRequest(status string, duration *time.Time) {
	if duration != nil {
		HT1["gets.current_url.duration"] = time.Since(*duration).Seconds()
	} else {
		switch status {
		case "add":
			HT2["gets.current_url.request.total"] = HT2["gets.current_url.request.total"] + 1
		case "error":
			HT2["gets.current_url.request.error"] = HT2["gets.current_url.request.error"] + 1
		case "bad_request":
			HT2["gets.current_url.request.bad_request"] = HT2["gets.current_url.request.bad_request"] + 1
		}
	}
}
func (m *MockMetrics) RecordPutBackendRequest(status string, duration *time.Time, sizeInBytes float64) {
	if duration != nil {
		HT1["puts.backends.request_duration"] = time.Since(*duration).Seconds()
	} else if sizeInBytes > 0 {
		HT1["puts.backends.request_size_bytes"] = sizeInBytes
	} else {
		switch status {
		case "add":
			HT2["puts.backends.request.total"] = HT2["puts.backends.request.total"] + 1
		case "json":
			HT2["puts.backends.json"] = HT2["puts.backends.json"] + 1
		case "xml":
			HT2["puts.backends.xml"] = HT2["puts.backends.xml"] + 1
		case "invalid_format":
			HT2["puts.backends.invalid_format"] = HT2["puts.backends.invalid_format"] + 1
		case "defines_ttl":
			HT2["puts.backends.defines_ttl"] = HT2["puts.backends.defines_ttl"] + 1
		case "error":
			HT2["puts.backends.request.error"] = HT2["puts.backends.request.error"] + 1
		}
	}
}

func (m *MockMetrics) RecordGetBackendRequest(status string, duration *time.Time) {
	if duration != nil {
		HT1["gets.backends.duration"] = time.Since(*duration).Seconds()
	} else {
		switch status {
		case "add":
			HT2["gets.backends.request.total"] = HT2["gets.backends.request.total"] + 1
		case "error":
			HT2["gets.backends.request.error"] = HT2["gets.backends.request.error"] + 1
		case "bad_request":
			HT2["gets.backends.request.bad_request"] = HT2["gets.backends.request.bad_request"] + 1
		}
	}
}
func (m *MockMetrics) RecordConnectionMetrics(label string) {
	switch label {
	case "add":
		HT1["connections.connections_opened"] = HT1["connections.connections_opened"] + 1
	case "substract":
		HT1["connections.connections_opened"] = HT1["connections.connections_opened"] - 1
	case "accept":
		HT2["connections.connection_error.accept"] = HT2["connections.connection_error.accept"] + 1
	case "close":
		HT2["connections.connection_error.close"] = HT2["connections.connection_error.close"] + 1
	}
}
func (m *MockMetrics) RecordExtraTTLSeconds(aVar float64) {
	HT1["extra_ttl_seconds"] = aVar
}
func (m *MockMetrics) Export(cfg config.Metrics) {
	//
}

//func (m *MockMetrics) getFloatEntry(string key) float64 {
//	if val, ok := HT1[string]; ok {
//		return val
//	}
//	return float64(-1)
//}
//func (m *MockMetrics) getIntEntry(string key) int {
//	if val, ok := HT2[string]; ok {
//		return val
//	}
//	return -1
//}

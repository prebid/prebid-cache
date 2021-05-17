package metrics

import (
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestRegisteredInfluxMetrics(t *testing.T) {
	m := CreateInfluxMetrics()

	// Test cases
	testCases := []struct {
		metricName, expectedMetricObject string
	}{
		// Puts:
		{"puts.current_url.request_duration", "Timer"},
		{"puts.current_url.error_count", "Meter"},
		{"puts.current_url.bad_request_count", "Meter"},
		{"puts.current_url.request_count", "Meter"},
		// Gets:
		{"gets.current_url.request_duration", "Timer"},
		{"gets.current_url.error_count", "Meter"},
		{"gets.current_url.bad_request_count", "Meter"},
		{"gets.current_url.request_count", "Meter"},
		// PutsBackend:
		{"puts.backend.request_duration", "Timer"},
		{"puts.backend.error_count", "Meter"},
		{"puts.backend.bad_request_count", "Meter"},
		{"puts.backend.json_request_count", "Meter"},
		{"puts.backend.xml_request_count", "Meter"},
		{"puts.backend.defines_ttl", "Meter"},
		{"puts.backend.unknown_request_count", "Meter"},
		{"puts.backend.request_size_bytes", "Histogram"},
		// GetsBackend:
		{"gets.backend.request_duration", "Timer"},
		{"gets.backend.error_count", "Meter"},
		{"gets.backend.bad_request_count", "Meter"},
		{"gets.backend.request_count", "Meter"},
		// GetsBackErr:
		{"gets.backend_error.key_not_found", "Meter"},
		{"gets.backend_error.missing_key", "Meter"},
		// Connections:
		{"connections.active_incoming", "Counter"},
		{"connections.accept_errors", "Meter"},
		{"connections.close_errors", "Meter"},
		// ExtraTTL:
		{"extra_ttl_seconds", "Histogram"},
	}

	// Assertions
	for _, test := range testCases {
		actualMetricObject := m.Registry.Get(test.metricName)
		assert.NotNil(t, actualMetricObject, "Metric %s was expected to be registered but it isn't", test.metricName)

		var correctMetricType bool
		switch test.expectedMetricObject {
		case "Timer":
			_, correctMetricType = actualMetricObject.(metrics.Timer)
		case "Meter":
			_, correctMetricType = actualMetricObject.(metrics.Meter)
		case "Counter":
			_, correctMetricType = actualMetricObject.(metrics.Counter)
		case "Histogram":
			_, correctMetricType = actualMetricObject.(metrics.Histogram)
		}
		assert.True(t, correctMetricType, "Metric %s was expected to be of type %s but it isn't", test.metricName, test.expectedMetricObject)
	}
}

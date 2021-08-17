package metrics

import (
	"strings"
	"testing"
	"time"

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
		{"puts.backend.request_ttl_seconds", "Timer"},
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

func TestDurationRecorders(t *testing.T) {
	var fiveSeconds time.Duration = time.Second * 5

	m := CreateInfluxMetrics()

	type testCase struct {
		description    string
		runTest        func(im *InfluxMetrics)
		metricToAssert interface{}
	}

	testGroups := []struct {
		groupDesc string
		testCases []testCase
	}{
		{
			"m.Puts",
			[]testCase{
				{
					description:    "Five second RecordPutDuration",
					runTest:        func(im *InfluxMetrics) { im.RecordPutDuration(fiveSeconds) },
					metricToAssert: m.Puts.Duration,
				},
				{
					description:    "record a generic put error with RecordPutError",
					runTest:        func(im *InfluxMetrics) { im.RecordPutError() },
					metricToAssert: m.Puts.Errors,
				},
				{
					description:    "record an incoming bad put request with RecordPutBadRequest",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBadRequest() },
					metricToAssert: m.Puts.BadRequest,
				},
				{
					description:    "record an incoming non-bad put request with RecordPutTotal",
					runTest:        func(im *InfluxMetrics) { im.RecordPutTotal() },
					metricToAssert: m.Puts.Request,
				},
			},
		},
		{
			"m.Gets",
			[]testCase{
				{
					description:    "Five second RecordGetDuration",
					runTest:        func(im *InfluxMetrics) { im.RecordGetDuration(fiveSeconds) },
					metricToAssert: m.Gets.Duration,
				},
				{
					description:    "record a generic put error with RecordGetError",
					runTest:        func(im *InfluxMetrics) { im.RecordGetError() },
					metricToAssert: m.Gets.Errors,
				},
				{
					description:    "record an incoming bad put request with RecordGetBadRequest",
					runTest:        func(im *InfluxMetrics) { im.RecordGetBadRequest() },
					metricToAssert: m.Gets.BadRequest,
				},
				{
					description:    "record an incoming non-bad put request with RecordGetTotal",
					runTest:        func(im *InfluxMetrics) { im.RecordGetTotal() },
					metricToAssert: m.Gets.Request,
				},
			},
		},
		{
			"m.PutsBackend",
			[]testCase{
				{
					description:    "Five second RecordPutBackendDuration",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendDuration(fiveSeconds) },
					metricToAssert: m.PutsBackend.Duration,
				},
				{
					description:    "record a generic put error with RecordPutBackendError",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendError() },
					metricToAssert: m.PutsBackend.Errors,
				},
				{
					description:    "record a valid XML put request with RecordPutBackendXml",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendXml() },
					metricToAssert: m.PutsBackend.XmlRequest,
				},
				{
					description:    "record a valid JSON put request with RecordPutBackendJson",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendJson() },
					metricToAssert: m.PutsBackend.JsonRequest,
				},
				{
					description:    "record an invalid put request with RecordPutBackendInvalid",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendInvalid() },
					metricToAssert: m.PutsBackend.InvalidRequest,
				},
				{
					description:    "valid put request specifies its time to live. Keep count of the number of requests that do so with RecordPutBackendDefTTL",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendDefTTL() },
					metricToAssert: m.PutsBackend.DefinesTTL,
				},
				{
					description:    "valid put request record the size of its value field in bytes with RecordPutBackendSize",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendSize(float64(1)) },
					metricToAssert: m.PutsBackend.RequestLength,
				},
				{
					description:    "valid put request comes with a non-zero value in the ttlseconds field. Record with RecordPutBackendSize",
					runTest:        func(im *InfluxMetrics) { im.RecordPutBackendTTLSeconds(fiveSeconds) },
					metricToAssert: m.PutsBackend.RequestTTL,
				},
			},
		},
		{
			"m.GetsBackend",
			[]testCase{
				{
					description:    "Five second RecordGetBackendDuration",
					runTest:        func(im *InfluxMetrics) { im.RecordGetBackendDuration(fiveSeconds) },
					metricToAssert: m.GetsBackend.Duration,
				},
				{
					description:    "record a generic get error with RecordGetBackendError",
					runTest:        func(im *InfluxMetrics) { im.RecordGetBackendError() },
					metricToAssert: m.GetsBackend.Errors,
				},
				{
					description:    "record an incoming, valid get request with RecordGetBackendTotal",
					runTest:        func(im *InfluxMetrics) { im.RecordGetBackendTotal() },
					metricToAssert: m.GetsBackend.Request,
				},
			},
		},
		{
			"m.GetsBackErr",
			[]testCase{
				{
					description:    "record a key not found get request error with RecordKeyNotFoundError",
					runTest:        func(im *InfluxMetrics) { im.RecordKeyNotFoundError() },
					metricToAssert: m.GetsErr.KeyNotFoundErrors,
				},
				{
					description:    "record a missing key, get request error with RecordMissingKeyError",
					runTest:        func(im *InfluxMetrics) { im.RecordMissingKeyError() },
					metricToAssert: m.GetsErr.MissingKeyErrors,
				},
			},
		},
		{
			"m.Connections",
			[]testCase{
				{
					description:    "Increase counter when a connection opens",
					runTest:        func(im *InfluxMetrics) { im.RecordConnectionOpen() },
					metricToAssert: m.Connections.ActiveConnections,
				},
				{
					description:    "Decrease counter when a connection closes",
					runTest:        func(im *InfluxMetrics) { im.RecordConnectionClosed() },
					metricToAssert: m.Connections.ActiveConnections,
				},
				{
					description:    "record a connection that suddenly closed with an error",
					runTest:        func(im *InfluxMetrics) { im.RecordCloseConnectionErrors() },
					metricToAssert: m.Connections.ConnectionCloseErrors,
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, test := range group.testCases {
			test.runTest(m)

			// In order to assert, find out what's the type of Influx metric
			if timer, isTimer := test.metricToAssert.(metrics.Timer); isTimer {
				assert.Equal(t, fiveSeconds.Nanoseconds(), timer.Sum(), "Group '%s'. Desc: %s", group.groupDesc, test.description)

			} else if meter, isMeter := test.metricToAssert.(metrics.Meter); isMeter {
				assert.Equal(t, int64(1), meter.Count(), "Group '%s'. Desc: %s", group.groupDesc, test.description)

			} else if histogram, isHistogram := test.metricToAssert.(metrics.Histogram); isHistogram {
				assert.Equal(t, int64(1), histogram.Sum(), "Group '%s'. Desc: %s", group.groupDesc, test.description)

			} else if counter, isCounter := test.metricToAssert.(metrics.Counter); isCounter {
				if strings.HasPrefix(test.description, "Increase") {
					assert.Equal(t, int64(1), counter.Count(), "Group '%s'. Desc: %s", group.groupDesc, test.description)
				} else {
					assert.Equal(t, int64(0), counter.Count(), "Group '%s'. Desc: %s", group.groupDesc, test.description)
				}
			}
		}
	}
}

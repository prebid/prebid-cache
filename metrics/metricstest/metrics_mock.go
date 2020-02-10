package metricstest

import (
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"time"
)

/*Define Mock metrics        */
var HT1 map[string]float64
var HT2 map[string]int64

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
}

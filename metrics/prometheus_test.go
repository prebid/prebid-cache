package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestMetricCountGatekeeping(t *testing.T) {
	m := createPrometheusMetricsForTesting()

	// Gather All Metrics
	//metricFamilies, err := m.Registry.Gather()
	_, err := m.Registry.Gather()
	assert.NoError(t, err, "gather metics")

}

/**************************************************
 *	The test case object
type PrometheusMetrics struct {

	for the `RequestDurationMetrics` field:  *prometheus.HistogramVec
	"puts.current_url.request_duration"
	"gets.current_url.request_duration"
	"puts.backend.request_duration"
	"gets.backend.request_duration"

	for the `MethodToEndpointMetrics` field: *prometheus.CounterVec
	"puts.current_url.error_count"
	"puts.current_url.bad_request_count"
	"puts.current_url.request_count"
	"gets.current_url.error_count"
	"gets.current_url.bad_request_count"
	"gets.current_url.request_count"
	"puts.backend.error_count"
	"puts.backend.bad_request_count"
	"puts.backend.json_request_count"
	"puts.backend.xml_request_count"
	"puts.backend.defines_ttl"
	"puts.backend.unknown_request_count"
	"gets.backend.error_count"
	"gets.backend.bad_request_count"
	"gets.backend.request_count"

	for the `RequestSyzeBytes` field:        *prometheus.HistogramVec
	"puts.backend.request_size_bytes" }

	for the `ConnectionErrorMetrics` field:  *prometheus.CounterVec
	"connections.accept_errors"
	"connections.close_errors"}

	for the `ActiveConnections` field:       prometheus.Gauge
	"connections.active_incoming"}

	for the `ExtraTTLSeconds` field:         prometheus.Histogram
	extra_ttl_seconds"}
}
 **************************************************/

// This test case will try to assert the functionality of `PrometheusMetrics.RequestDurationMetrics` which is a `HistogramVector`
// with the following labels: {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
func TestRequestDurationMetrics(t *testing.T) {
	testCases := []struct {
		testDescription          string
		testObservations         map[string][]int
		expectedHistogramEntries map[string]map[string]uint64
	}{
		{
			testDescription: "Test to log an initial time into every histogogram in the vector of histograms",
			testObservations: map[string][]int{
				"puts.current_url.request_duration": []int{10, 10, 10, 10},
				"gets.current_url.request_duration": []int{5, 10, 5},
				"puts.backend.request_duration":     []int{5, 0, 5},
				"gets.backend.request_duration":     []int{},
			},
			expectedHistogramEntries: map[string]map[string]uint64{
				"puts.current_url.request_duration": map[string]uint64{
					"sample_count": 4,
					"sample_sum":   40,
				},
				"gets.current_url.request_duration": map[string]uint64{
					"sample_count": 3,
					"sample_sum":   20,
				},
				"puts.backend.request_duration": map[string]uint64{
					"sample_count": 3,
					"sample_sum":   10,
				},
				"gets.backend.request_duration": map[string]uint64{
					"sample_count": 0,
					"sample_sum":   0,
				},
			},
		},
	}

	cacheMetrics := createPrometheusMetricsForTesting()
	timeStamp := time.Now()
	for _, test := range testCases {
		//per `test`, log observations to `PrometheusMetrics.RequestDurationMetrics` which is a `HistogramVector`
		for metricLabel, reqDurations := range test.testObservations {
			//add to metric
			for dur := range reqDurations {
				timeEntry := timeStamp.Add(time.Second * time.Duration(dur))
				cacheMetrics.Increment(metricLabel, &timeEntry, "")
			}

		}

		//assert results of each Histogram in the vector
		for metricName, results := range test.expectedHistogramEntries {
			tokens := strings.Split(metricName, ".")
			promLabelKey := tokens[0] + "." + tokens[1]

			thisHistogram := getHistogramFromHistogramVec(cacheMetrics.RequestDurationMetrics, promLabelKey, tokens[2])

			assert.NotNil(t, results, "No histogram found")

			// Assert sample count and sample sum
			assertHistogram(t, metricName, thisHistogram, results["sample_count"], float64(results["sample_sum"]))
			break
		}
	}
}

// This test case will try to assert the functionality of `ConnectionErrorMetrics` which is a `CounterVec` because we expect
// its numbers to only increase and will use the following labels: { "connections.accept_errors", "connections.close_errors"}
func TestConnectionMetrics(t *testing.T) {
	testCases := []struct {
		description                       string
		testFunction                      func(m *PrometheusMetrics)
		expectedConnectionsAcceptErrors   float64
		expectedConnectionsCloseErrors    float64
		expectedConnectionsActiveIncoming float64
	}{
		{
			description: "Test number of connection accept errors",
			testFunction: func(m *PrometheusMetrics) {
				m.Increment("connections.accept_errors", nil, "")
			},
			expectedConnectionsAcceptErrors:   1,
			expectedConnectionsCloseErrors:    0,
			expectedConnectionsActiveIncoming: 0,
		},
		{
			description: "Test number of connection close errors",
			testFunction: func(m *PrometheusMetrics) {
				m.Increment("connections.close_errors", nil, "")
			},
			expectedConnectionsAcceptErrors:   0,
			expectedConnectionsCloseErrors:    1,
			expectedConnectionsActiveIncoming: 0,
		},
		{
			description: "Active connections test",
			testFunction: func(m *PrometheusMetrics) {
				m.Increment("connections.active_incoming", nil, "")
				m.Increment("connections.active_incoming", nil, "")
				m.Decrement("connections.active_incoming")
			},
			expectedConnectionsAcceptErrors:   0,
			expectedConnectionsCloseErrors:    0,
			expectedConnectionsActiveIncoming: 1,
		},
	}

	for _, test := range testCases {
		m := createPrometheusMetricsForTesting()

		test.testFunction(m)

		ConnectionsAcceptErrorsCounter := m.ConnectionErrorMetrics.With(prometheus.Labels{"connections": "accept_errors"})
		ConnectionsCloseErrorsCounter := m.ConnectionErrorMetrics.With(prometheus.Labels{"connections": "close_errors"})

		assertCounterValue(t, test.description, ConnectionsAcceptErrorsCounter, test.expectedConnectionsAcceptErrors)
		assertCounterValue(t, test.description, ConnectionsCloseErrorsCounter, test.expectedConnectionsCloseErrors)
		assertGaugeValue(t, test.description, m.ActiveConnections, test.expectedConnectionsActiveIncoming)
	}
}

func createPrometheusMetricsForTesting() *PrometheusMetrics {
	promConfig := config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "server",
	}
	return CreatePrometheusMetrics(promConfig)
}

func assertCounterValue(t *testing.T, description string, counter prometheus.Counter, expected float64) {
	m := dto.Metric{}
	counter.Write(&m)
	actual := *m.GetCounter().Value

	assert.Equal(t, expected, actual, description)
}

func assertGaugeValue(t *testing.T, description string, gauge prometheus.Gauge, expected float64) {
	m := dto.Metric{}
	gauge.Write(&m)
	actual := *m.GetGauge().Value

	assert.Equal(t, expected, actual, description)
}

func assertCounterVecValue(t *testing.T, description string, counterVec *prometheus.CounterVec, expected float64, labels prometheus.Labels) {
	counter := counterVec.With(labels)
	assertCounterValue(t, description, counter, expected)
}

func getHistogramFromHistogramVec(histogram *prometheus.HistogramVec, labelKey, labelValue string) dto.Histogram {
	var result dto.Histogram
	processMetrics(histogram, func(m dto.Metric) {
		for _, label := range m.GetLabel() {
			if label.GetName() == labelKey && label.GetValue() == labelValue {
				result = *m.GetHistogram()
			}
		}
	})
	return result
}

func processMetrics(collector prometheus.Collector, handler func(m dto.Metric)) {
	collectorChan := make(chan prometheus.Metric)
	go func() {
		collector.Collect(collectorChan)
		close(collectorChan)
	}()

	for metric := range collectorChan {
		dtoMetric := dto.Metric{}
		metric.Write(&dtoMetric)
		handler(dtoMetric)
	}
}

func assertHistogram(t *testing.T, name string, histogram dto.Histogram, expectedCount uint64, expectedSum float64) {
	assert.Equal(t, expectedCount, histogram.GetSampleCount(), name+":count")
	assert.Equal(t, expectedSum, histogram.GetSampleSum(), name+":sum")
}

//func TestRequestMetric(t *testing.T) {
//func TestRequestMetricWithoutCookie(t *testing.T) {
//func TestAccountMetric(t *testing.T) {
//func TestImpressionsMetric(t *testing.T) {
//func TestLegacyImpressionsMetric(t *testing.T) {
//func TestRequestTimeMetric(t *testing.T) {
//func TestAdapterBidReceivedMetric(t *testing.T) {
//func TestRecordAdapterPriceMetric(t *testing.T) {
//func TestAdapterRequestMetrics(t *testing.T) {
//func TestAdapterRequestErrorMetrics(t *testing.T) {
//func TestAdapterTimeMetric(t *testing.T) {
//func TestAdapterCookieSyncMetric(t *testing.T) {
//func TestUserIDSetMetric(t *testing.T) {
//func TestUserIDSetMetricWhenBidderEmpty(t *testing.T) {
//func TestAdapterPanicMetric(t *testing.T) {
//func TestStoredReqCacheResultMetric(t *testing.T) {
//func TestStoredImpCacheResultMetric(t *testing.T) {
//func TestCookieMetric(t *testing.T) {
//func TestPrebidCacheRequestTimeMetric(t *testing.T) {
//func TestMetricAccumulationSpotCheck(t *testing.T) {

package metrics

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

const TenSeconds time.Duration = time.Second * 10

func createPrometheusMetricsForTesting() *PrometheusMetrics {
	return CreatePrometheusMetrics(config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "cache",
	})
}

func assertCounterVecValue(t *testing.T, description string, counterVec *prometheus.CounterVec, expected float64, labels prometheus.Labels) {
	counter := counterVec.With(labels)
	assertCounterValue(t, description, counter, expected)
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

func assertHistogram(t *testing.T, name string, histogram prometheus.Histogram, expectedCount uint64, expectedSum float64) {
	m := dto.Metric{}
	histogram.Write(&m)
	actual := *m.GetHistogram()

	assert.Equal(t, expectedCount, actual.GetSampleCount(), name+":count")
	assert.Equal(t, expectedSum, actual.GetSampleSum(), name+":sum")
}

func TestPrometheusGetMetricsEngineName(t *testing.T) {
	m := createPrometheusMetricsForTesting()
	engineName := m.GetMetricsEngineName()

	assert.Equal(t, "Prometheus", engineName, "Prometheus engine name should be 'Prometheus', actual: %s \n", engineName)
}

func TestPrometheusGetEngineRegistry(t *testing.T) {
	m := createPrometheusMetricsForTesting()

	registry := m.GetEngineRegistry()

	_, ok := registry.(*prometheus.Registry)

	assert.True(t, ok, "Prometheus engine registry should be of type *prometheus.Registry")
}

func TestPrometheusRequestStatusMetric(t *testing.T) {
	m := createPrometheusMetricsForTesting()

	type testCaseObject struct {
		description      string
		expDuration      float64
		expRequestTotals float64
		expRequestErrors float64
		expBadRequests   float64
		testCase         func(pm *PrometheusMetrics)
	}

	testGroups := map[*PrometheusRequestStatusMetric][]testCaseObject{
		m.Puts: {
			{
				description: "Log put request duration",
				testCase: func(pm *PrometheusMetrics) {
					pm.RecordPutDuration(TenSeconds)
				},
				expDuration:      10,
				expRequestTotals: 0, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count put request total",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordPutTotal() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count put request error",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordPutError() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 0,
			},
			{
				description:      "Count put request bad request",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordPutBadRequest() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 1,
			},
		},
		m.Gets: {
			{
				description: "Log get request duration",
				testCase: func(pm *PrometheusMetrics) {
					pm.RecordGetDuration(TenSeconds)
				},
				expDuration:      10,
				expRequestTotals: 0, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count get request total",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetTotal() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count get request error",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetError() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 0,
			},
			{
				description:      "Count get request bad request",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetBadRequest() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 1,
			},
		},
		m.GetsBackend: {
			{
				description: "Log get backend request duration",
				testCase: func(pm *PrometheusMetrics) {
					pm.RecordGetBackendDuration(TenSeconds)
				},
				expDuration:      10,
				expRequestTotals: 0, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count get backend request total",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetBackendTotal() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 0, expBadRequests: 0,
			},
			{
				description:      "Count get backend request error",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetBackendError() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 0,
			},
			{
				description:      "Count get backend request bad request",
				testCase:         func(pm *PrometheusMetrics) { pm.RecordGetBackendBadRequest() },
				expDuration:      10,
				expRequestTotals: 1, expRequestErrors: 1, expBadRequests: 1,
			},
		},
	}

	for prometheusMetric, testCaseArray := range testGroups {
		for _, test := range testCaseArray {
			test.testCase(m)

			assertHistogram(t, test.description, prometheusMetric.Duration, 1, test.expDuration)
			assertCounterVecValue(t, test.description, prometheusMetric.RequestStatus, test.expRequestTotals, prometheus.Labels{StatusKey: TotalsVal})
			assertCounterVecValue(t, test.description, prometheusMetric.RequestStatus, test.expRequestErrors, prometheus.Labels{StatusKey: ErrorVal})
			assertCounterVecValue(t, test.description, prometheusMetric.RequestStatus, test.expBadRequests, prometheus.Labels{StatusKey: BadRequestVal})
		}
	}
}

func TestGetsBackendErrorsByType(t *testing.T) {

	m := createPrometheusMetricsForTesting()

	testCaseArray := []struct {
		description          string
		expKeyNotFoundErrors float64
		expMissingKeyErrors  float64
		recordMetric         func(pm *PrometheusMetrics)
	}{
		{
			description:          "Add to the get backend key not found error counter",
			expKeyNotFoundErrors: 1,
			expMissingKeyErrors:  0,
			recordMetric:         func(pm *PrometheusMetrics) { pm.RecordKeyNotFoundError() },
		},
		{
			description:          "Add to the get backend missing key error",
			expKeyNotFoundErrors: 1,
			expMissingKeyErrors:  1,
			recordMetric:         func(pm *PrometheusMetrics) { pm.RecordMissingKeyError() },
		},
	}

	for _, test := range testCaseArray {
		test.recordMetric(m)

		assertCounterVecValue(t, test.description, m.GetsBackend.ErrorsByType, test.expKeyNotFoundErrors, prometheus.Labels{TypeKey: KeyNotFoundVal})
		assertCounterVecValue(t, test.description, m.GetsBackend.ErrorsByType, test.expMissingKeyErrors, prometheus.Labels{TypeKey: MissingKeyVal})
	}
}

func TestPutBackendMetrics(t *testing.T) {
	m := createPrometheusMetricsForTesting()

	type testCaseObject struct {
		description string
		testCase    func(pm *PrometheusMetrics)

		//counters
		expXmlCount     float64
		expJsonCount    float64
		expInvalidCount float64
		expDefTTLCount  float64
		expErrorCount   float64

		//Duration and sixe in bytes
		expDuration      float64
		expSizeHistSum   float64
		expSizeHistCount uint64
	}

	testCases := []testCaseObject{
		{
			description: "Log put backend request duration",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordPutBackendDuration(TenSeconds)
			},
			expDuration: 10,
		},
		{
			description: "Count put backend xml request",
			testCase:    func(pm *PrometheusMetrics) { pm.RecordPutBackendXml() },
			expDuration: 10,
			expXmlCount: 1,
		},
		{
			description:  "Count put backend json request",
			testCase:     func(pm *PrometheusMetrics) { pm.RecordPutBackendJson() },
			expDuration:  10,
			expXmlCount:  1,
			expJsonCount: 1,
		},
		{
			description:     "Count put backend invalid request",
			testCase:        func(pm *PrometheusMetrics) { pm.RecordPutBackendInvalid() },
			expDuration:     10,
			expXmlCount:     1,
			expJsonCount:    1,
			expInvalidCount: 1,
		},
		{
			description:     "Count put backend of requests that define TTL",
			testCase:        func(pm *PrometheusMetrics) { pm.RecordPutBackendDefTTL() },
			expDuration:     10,
			expXmlCount:     1,
			expJsonCount:    1,
			expInvalidCount: 1,
			expDefTTLCount:  1,
		},
		{
			description:     "Count put backend request errors",
			testCase:        func(pm *PrometheusMetrics) { pm.RecordPutBackendError() },
			expDuration:     10,
			expXmlCount:     1,
			expJsonCount:    1,
			expInvalidCount: 1,
			expDefTTLCount:  1,
			expErrorCount:   1,
		},
		{
			description: "Log put backend request duration",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordPutBackendSize(16)
			},
			expDuration:      10,
			expXmlCount:      1,
			expJsonCount:     1,
			expInvalidCount:  1,
			expDefTTLCount:   1,
			expErrorCount:    1,
			expSizeHistSum:   16,
			expSizeHistCount: 1,
		},
	}

	for _, test := range testCases {
		test.testCase(m)

		assertHistogram(t, test.description, m.PutsBackend.Duration, 1, test.expDuration)
		assertCounterVecValue(t, test.description, m.PutsBackend.PutBackendRequests, test.expXmlCount, prometheus.Labels{FormatKey: XmlVal})
		assertCounterVecValue(t, test.description, m.PutsBackend.PutBackendRequests, test.expJsonCount, prometheus.Labels{FormatKey: JsonVal})
		assertCounterVecValue(t, test.description, m.PutsBackend.PutBackendRequests, test.expInvalidCount, prometheus.Labels{FormatKey: InvFormatVal})
		assertCounterVecValue(t, test.description, m.PutsBackend.PutBackendRequests, test.expDefTTLCount, prometheus.Labels{FormatKey: DefinesTTLVal})
		assertCounterVecValue(t, test.description, m.PutsBackend.PutBackendRequests, test.expErrorCount, prometheus.Labels{FormatKey: ErrorVal})
		assertHistogram(t, test.description, m.PutsBackend.RequestLength, test.expSizeHistCount, test.expSizeHistSum)
	}
}

func TestConnectionMetrics(t *testing.T) {
	testCases := []struct {
		description                    string
		testCase                       func(pm *PrometheusMetrics)
		expectedOpenedConnectionCount  float64
		expectedClosedConnectionCount  float64
		expectedAcceptConnectionErrors float64
		expectedCloseConnectionErrors  float64
	}{
		{
			description: "Add a connection to the open connection count",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordConnectionOpen()
			},
			expectedOpenedConnectionCount:  1,
			expectedClosedConnectionCount:  0,
			expectedAcceptConnectionErrors: 0,
			expectedCloseConnectionErrors:  0,
		},
		{
			description: "Remove a connection from the open connection count",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordConnectionClosed()
			},
			expectedOpenedConnectionCount:  1,
			expectedClosedConnectionCount:  1,
			expectedAcceptConnectionErrors: 0,
			expectedCloseConnectionErrors:  0,
		},
		{
			description: "Count a close connection error",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordCloseConnectionErrors()
			},
			expectedOpenedConnectionCount:  1,
			expectedClosedConnectionCount:  1,
			expectedAcceptConnectionErrors: 0,
			expectedCloseConnectionErrors:  1,
		},
		{
			description: "Count an accept connection error",
			testCase: func(pm *PrometheusMetrics) {
				pm.RecordCloseConnectionErrors()
				pm.RecordAcceptConnectionErrors()
			},
			expectedOpenedConnectionCount:  1,
			expectedClosedConnectionCount:  1,
			expectedAcceptConnectionErrors: 1,
			expectedCloseConnectionErrors:  2,
		},
	}

	m := createPrometheusMetricsForTesting()

	for _, test := range testCases {
		test.testCase(m)

		assertCounterValue(t, test.description, m.Connections.ConnectionsOpened, test.expectedOpenedConnectionCount)
		assertCounterValue(t, test.description, m.Connections.ConnectionsClosed, test.expectedClosedConnectionCount)

		assertCounterVecValue(t, test.description, m.Connections.ConnectionsErrors, test.expectedAcceptConnectionErrors, prometheus.Labels{ConnErrorKey: AcceptVal})
		assertCounterVecValue(t, test.description, m.Connections.ConnectionsErrors, test.expectedCloseConnectionErrors, prometheus.Labels{ConnErrorKey: CloseVal})
	}
}

func TestExtraTTLMetrics(t *testing.T) {
	m := createPrometheusMetricsForTesting()

	m.RecordExtraTTLSeconds(5)
	assertHistogram(t, "Assert the extra time to live in seconds was logged", m.ExtraTTL.ExtraTTLSeconds, 1, 5.00)
}

func TestMetricCountGatekeeping(t *testing.T) {
	expectedCardinalityCount := 26
	actualCardinalityCount := 0

	// Run test
	m := createPrometheusMetricsForTesting()
	metricFamilies, err := m.Registry.Gather()
	assert.NoError(t, err, "gather metics")

	// Assertions
	for _, metricFamily := range metricFamilies {
		actualCardinalityCount += len(metricFamily.GetMetric())
	}

	// This assertion provides a warning for newly added high-cardinality non-adapter specific metrics. The hardcoded limit
	//   is an arbitrary soft ceiling. Thought should be given as to the value of the new metrics if you find yourself
	//   needing to increase this number.
	assert.True(t, actualCardinalityCount <= expectedCardinalityCount, "General Cardinality doesn't match")
}

func TestListedMetrics(t *testing.T) {
	type metricData struct {
		metricDesc        string
		metricLabelValues []string
	}
	expectedMetricList := map[string]metricData{
		//Puts metrics
		PutRequestMet: metricData{
			metricDesc: "Count of total requests to Prebid Server labeled by status.",
			metricLabelValues: []string{
				StatusKey + " = " + BadRequestVal,
				StatusKey + " = " + ErrorVal,
				StatusKey + " = " + TotalsVal,
			},
		},
		PutReqDurMet: metricData{
			metricDesc: "Duration in seconds Prebid Cache takes to process put requests.",
		},

		//Gets metrics
		GetRequestMet: metricData{
			metricDesc: "Count of total get requests to Prebid Server labeled by status.",
			metricLabelValues: []string{
				StatusKey + " = " + BadRequestVal,
				StatusKey + " = " + ErrorVal,
				StatusKey + " = " + TotalsVal,
			},
		},
		GetReqDurMet: metricData{
			metricDesc: "Duration in seconds Prebid Cache takes to process get requests.",
		},

		//PutsBackend metrics
		PutBackendMet: metricData{
			metricDesc: "Count of total requests to Prebid Cache labeled by format, status and whether or not it comes with TTL",
			metricLabelValues: []string{
				FormatKey + " = " + DefinesTTLVal,
				FormatKey + " = " + ErrorVal,
				FormatKey + " = " + InvFormatVal,
				FormatKey + " = " + JsonVal,
				FormatKey + " = " + XmlVal,
			},
		},
		PutBackDurMet: metricData{
			metricDesc: "Duration in seconds Prebid Cache takes to process backend put requests.",
		},
		PutBackSizeMet: metricData{
			metricDesc: "Size in bytes of a backend put request.",
		},

		//GetsBackend metrics
		GetBackendMet: metricData{
			metricDesc: "Count of total backend get requests to Prebid Server labeled by status.",
			metricLabelValues: []string{
				StatusKey + " = " + BadRequestVal,
				StatusKey + " = " + ErrorVal,
				StatusKey + " = " + TotalsVal,
			},
		},
		GetBackendErr: metricData{
			metricDesc: "Account for the most frequent type of get errors in the backend",
			metricLabelValues: []string{
				TypeKey + " = " + KeyNotFoundVal,
				TypeKey + " = " + MissingKeyVal,
			},
		},
		GetBackDurMet: metricData{
			metricDesc: "Duration in seconds Prebid Cache takes to process backend get requests.",
		},

		//Connections metrics
		ConnClosedMet: metricData{
			metricDesc: "Count the number of closed connections",
		},
		ConnErrorKey: metricData{
			metricDesc: "Count the number of connection accept errors or connection close errors",
			metricLabelValues: []string{
				ConnErrorKey + " = accept",
				ConnErrorKey + " = close",
			},
		},
		ConnOpenedMet: metricData{
			metricDesc: "Count the number of open connections",
		},

		//ExtraTTL metrics
		ExtraTTLMet: metricData{
			metricDesc: "Extra time to live in seconds specified",
		},
	}

	// Run test
	m := createPrometheusMetricsForTesting()

	// Assert results
	metricFamilies, err := m.Registry.Gather()
	assert.NoError(t, err, "gather metics")

	for _, metricFamily := range metricFamilies {
		metName := strings.TrimPrefix(metricFamily.GetName(), "prebid_cache_")
		mData, found := expectedMetricList[metName]

		if assert.True(t, found, "Metric %s not found in the actual metric list", metName) {
			assert.Equal(t, mData.metricDesc, metricFamily.GetHelp(), "Metric %s help data does not match.", metName)
		}

		if len(mData.metricLabelValues) > 0 {
			metrics := metricFamily.GetMetric()
			actualMetricList := []string{}

			for _, metric := range metrics {
				metricLabels := metric.GetLabel()

				for _, labelPair := range metricLabels {
					actualMetricList = append(actualMetricList, fmt.Sprintf("%s = %s", labelPair.GetName(), labelPair.GetValue()))
				}
			}
			assert.ElementsMatch(t, mData.metricLabelValues, actualMetricList, "Metric %s. Label list does not match expected", metName)
		}
	}
}

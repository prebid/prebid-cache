package metrics

import (
	"testing"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

//func TestMetricCountGatekeeping(t *testing.T) {
//func TestConnectionMetrics(t *testing.T) {
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

func createPrometheusMetricsForTesting() *PrometheusMetrics {
	promConfig := config.PrometheusMetrics{
		Port:      8080,
		Namespace: "prebid",
		Subsystem: "server",
	}
	return CreatePrometheusMetrics(promConfig)
}

func assertCounterValue(t *testing.T, description, name string, counter prometheus.Counter, expected float64) {
	m := prometheus.Metric{}
	counter.Write(&m)
	actual := *m.GetCounter().Value

	assert.Equal(t, expected, actual, description)
}

func assertCounterVecValue(t *testing.T, description, name string, counterVec *prometheus.CounterVec, expected float64, labels prometheus.Labels) {
	counter := counterVec.With(labels)
	assertCounterValue(t, description, name, counter, expected)
}

func getHistogramFromHistogramVec(histogram *prometheus.HistogramVec, labelKey, labelValue string) prometheus.Histogram {
	var result prometheus.Histogram
	processMetrics(histogram, func(m prometheus.Metric) {
		for _, label := range m.GetLabel() {
			if label.GetName() == labelKey && label.GetValue() == labelValue {
				result = *m.GetHistogram()
			}
		}
	})
	return result
}

func processMetrics(collector prometheus.Collector, handler func(m prometheus.Metric)) {
	collectorChan := make(chan prometheus.Metric)
	go func() {
		collector.Collect(collectorChan)
		close(collectorChan)
	}()

	for metric := range collectorChan {
		dtoMetric := prometheus.Metric{}
		metric.Write(&dtoMetric)
		handler(dtoMetric)
	}
}

func assertHistogram(t *testing.T, name string, histogram prometheus.Histogram, expectedCount uint64, expectedSum float64) {
	assert.Equal(t, expectedCount, histogram.GetSampleCount(), name+":count")
	assert.Equal(t, expectedSum, histogram.GetSampleSum(), name+":sum")
}

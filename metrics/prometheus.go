package metrics

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"time"
)

/**************************************************
 *	Still missing:
 **************************************************
 *	> Register methods for Prometheus metrics in metrics/prometheus.go
 *	> Modify all histograms
 *	> Tests for config.go
 *	> Tests for metrics/prometheus.go
 *	> Upload so they can see progress
 *	> Make the option betewwn Prometheus and InfluxDB configurable
 **************************************************/
/**************************************************
 *	Object definition
 **************************************************/

type PrometheusMetrics struct {
	Registry                *prometheus.Registry
	RequestDurationMetrics  *prometheus.HistogramVec
	MethodToEndpointMetrics *prometheus.CounterVec
	RequestSyzeBytes        *prometheus.HistogramVec
	ConnectionMetrics       *prometheus.CounterVec
	ExtraTTLSeconds         prometheus.Histogram
}

/**************************************************
 *	Init functions
 **************************************************/

func newCounterVecWithLabels(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string, labels []string) *prometheus.CounterVec {
	opts := prometheus.CounterOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	counterVec := prometheus.NewCounterVec(opts, labels)
	registry.MustRegister(counterVec)
	return counterVec
}

func newHistogramVector(cfg config.PrometheusMetrics, registry *prometheus.Registry, name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	opts := prometheus.HistogramOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}
	histogram := prometheus.NewHistogramVec(opts, labels)
	registry.MustRegister(histogram)
	return histogram
}

/**************************************************
 *	Functions to record metrics
 *	> How does Influx records? based on what values?
 *	> Once we know this, just translate to `promMetric.Inc()` and we are all set
 **************************************************/
// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m PrometheusMetrics) Export(cfg config.Metrics) {
	logrus.Infof("Metrics will be exported to Prometheus with host=%s, db=%s, username=%s", cfg.Influx.Host, cfg.Influx.Database, cfg.Influx.Username)
	//influxdb.InfluxDB(
	//	m.Registry,          // metrics registry
	//	time.Second*10,      // interval
	//	cfg.Influx.Host,     // the InfluxDB url
	//	cfg.Influx.Database, // your InfluxDB database
	//	cfg.Influx.Username, // your InfluxDB user
	//	cfg.Influx.Password, // your InfluxDB password
	//)
	return
}

func (m PrometheusMetrics) Increment(metricName string, start *time.Time, value string) {
	metricNameTokens := strings.Split(metricName, ".")

	if len(metricNameTokens) == 2 && metricNameTokens[0] == "connections" {
		m.ConnectionMetrics.With(prometheus.Labels{
			metricNameTokens[0]: metricNameTokens[2], // {"connections.active_incoming", "connections.accept_errors", "connections.close_errors"}
		}).Inc()
	} else if len(metricNameTokens) == 3 {
		label := fmt.Sprintf("%s.%s", metricNameTokens[0], metricNameTokens[1])
		if metricNameTokens[0] == "gets" || metricNameTokens[0] == "puts" {
			if metricNameTokens[2] == "request_duration" {
				m.RequestDurationMetrics.With(prometheus.Labels{
					label: metricNameTokens[2], // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
				}).Observe(time.Since(*start).Seconds())
			} else if metricNameTokens[2] == "request_size_bytes" {
				m.RequestSyzeBytes.With(prometheus.Labels{
					"method": fmt.Sprintf("%s.%s", label, metricNameTokens[2]), // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
				}).Observe(float64(len(value)))
			} else {
				m.MethodToEndpointMetrics.With(prometheus.Labels{
					label: metricNameTokens[2], // {"connections.active_incoming", "connections.accept_errors", "connections.close_errors"}
				}).Inc()
			}
		}
	}
}

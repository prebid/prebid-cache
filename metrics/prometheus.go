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
 *	Object definition
 **************************************************/

type PrometheusMetrics struct {
	Registry                *prometheus.Registry
	RequestDurationMetrics  *prometheus.HistogramVec // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
	MethodToEndpointMetrics *prometheus.CounterVec   //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
	RequestSyzeBytes        *prometheus.HistogramVec //{ "puts.backend.request_size_bytes" }
	ConnectionErrorMetrics  *prometheus.CounterVec   // {"connections.accept_errors", "connections.close_errors"}
	ActiveConnections       prometheus.Gauge         // {"connections.active_incoming"}
	ExtraTTLSeconds         prometheus.Histogram     //{"extra_ttl_seconds"}
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

func newGaugeMetric(cfg config.PrometheusMetrics, registry *prometheus.Registry, name string, help string) prometheus.Gauge {
	opts := prometheus.GaugeOpts{
		Namespace: cfg.Namespace,
		Subsystem: cfg.Subsystem,
		Name:      name,
		Help:      help,
	}
	gauge := prometheus.NewGauge(opts)
	registry.MustRegister(gauge)
	return gauge
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
		switch metricNameTokens[1] {
		case "close_errors":
			fallthrough
		case "accept_errors":
			m.ConnectionErrorMetrics.With(prometheus.Labels{
				metricNameTokens[0]: metricNameTokens[1], // { "connections.accept_errors", "connections.close_errors"}
			}).Inc()
		case "active_incoming":
			m.ActiveConnections.Inc() //{ "connections.active_incoming"}
		}
	} else if len(metricNameTokens) == 3 {
		label := fmt.Sprintf("%s.%s", metricNameTokens[0], metricNameTokens[1])
		if metricNameTokens[0] == "gets" || metricNameTokens[0] == "puts" {
			if metricNameTokens[2] == "request_duration" {
				m.RequestDurationMetrics.With(prometheus.Labels{"method": label, "result": metricNameTokens[2]}).Observe(time.Since(*start).Seconds())
				// {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
			} else if metricNameTokens[2] == "request_size_bytes" {
				m.RequestSyzeBytes.With(prometheus.Labels{
					"method": fmt.Sprintf("%s.%s", label, metricNameTokens[2]), // {"puts.current_url.request_duration", "gets.current_url.request_duration", "puts.backend.request_duration", "gets.backend.request_duration"}
				}).Observe(float64(len(value)))
			} else {
				m.MethodToEndpointMetrics.With(prometheus.Labels{
					"method": label, "count_type": metricNameTokens[2], //{"puts.current_url.error_count", "puts.current_url.bad_request_count", "puts.current_url.request_count", "gets.current_url.error_count", "gets.current_url.bad_request_count", "gets.current_url.request_count", "puts.backend.error_count", "puts.backend.bad_request_count", "puts.backend.json_request_count", "puts.backend.xml_request_count","puts.backend.defines_ttl", "puts.backend.unknown_request_count", "gets.backend.error_count", "gets.backend.bad_request_count", "gets.backend.request_count"}
				}).Inc()
			}
		}
	}
}

func (m PrometheusMetrics) Decrement(metricName string) {
	switch metricName {
	case "connections.active_incoming":
		m.ActiveConnections.Dec()
	default:
		//error
	}
}

package metrics

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/config"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
)

/**************************************************
 *	Object definition
 **************************************************/
type InfluxMetrics struct {
	Registry        metrics.Registry
	Puts            *MetricsEntry
	Gets            *MetricsEntry
	PutsBackend     *MetricsEntryByFormat
	GetsBackend     *MetricsEntry
	Connections     *ConnectionMetrics
	ExtraTTLSeconds metrics.Histogram
}

type InfluxMetricsEntry struct {
	Duration   metrics.Timer
	Errors     metrics.Meter
	BadRequest metrics.Meter
	Request    metrics.Meter
}

type InfluxMetricsEntryByFormat struct {
	Duration       metrics.Timer
	Errors         metrics.Meter
	BadRequest     metrics.Meter
	JsonRequest    metrics.Meter
	XmlRequest     metrics.Meter
	DefinesTTL     metrics.Meter
	InvalidRequest metrics.Meter
	RequestLength  metrics.Histogram
}

type InfluxConnectionMetrics struct {
	ActiveConnections      metrics.Counter
	ConnectionCloseErrors  metrics.Meter
	ConnectionAcceptErrors metrics.Meter
}

func NewInfluxMetricsEntry(name string, r metrics.Registry) *InfluxMetricsEntry {
	return &InfluxMetricsEntry{
		Duration:   metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
		BadRequest: metrics.GetOrRegisterMeter(fmt.Sprintf("%s.bad_request_count", name), r),
		Request:    metrics.GetOrRegisterMeter(fmt.Sprintf("%s.request_count", name), r),
	}
}

func NewInfluxMetricsEntryBackendPuts(name string, r metrics.Registry) *InfluxMetricsEntryByFormat {
	return &InfluxMetricsEntryByFormat{
		Duration:       metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:         metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
		BadRequest:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.bad_request_count", name), r),
		JsonRequest:    metrics.GetOrRegisterMeter(fmt.Sprintf("%s.json_request_count", name), r),
		XmlRequest:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.xml_request_count", name), r),
		DefinesTTL:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.defines_ttl", name), r),
		InvalidRequest: metrics.GetOrRegisterMeter(fmt.Sprintf("%s.unknown_request_count", name), r),
		RequestLength:  metrics.GetOrRegisterHistogram(name+".request_size_bytes", r, metrics.NewExpDecaySample(1028, 0.015)),
	}
}

func NewInfluxConnectionMetrics(r metrics.Registry) *InfluxConnectionMetrics {
	return &InfluxConnectionMetrics{
		ActiveConnections:      metrics.GetOrRegisterCounter("connections.active_incoming", r),
		ConnectionAcceptErrors: metrics.GetOrRegisterMeter("connections.accept_errors", r),
		ConnectionCloseErrors:  metrics.GetOrRegisterMeter("connections.close_errors", r),
	}
}

func CreateInfluxMetrics() *InfluxMetrics {
	flushTime := time.Second * 10
	r := metrics.NewPrefixedRegistry("prebidcache.")
	m := &InfluxMetrics{
		Registry:        r,
		Puts:            NewInfluxMetricsEntry("puts.current_url", r),
		Gets:            NewInfluxMetricsEntry("gets.current_url", r),
		PutsBackend:     NewInfluxMetricsEntryBackendPuts("puts.backend", r),
		GetsBackend:     NewInfluxMetricsEntry("gets.backend", r),
		Connections:     NewInfluxConnectionMetrics(r),
		ExtraTTLSeconds: metrics.GetOrRegisterHistogram("extra_ttl_seconds", r, metrics.NewUniformSample(5000)),
	}

	metrics.RegisterDebugGCStats(m.Registry)
	metrics.RegisterRuntimeMemStats(m.Registry)

	go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	go metrics.CaptureDebugGCStats(m.Registry, flushTime)

	return m
}

// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m InfluxMetrics) Export(cfg config.Metrics) {
	if cfg.Influx.Host != "" {
		logrus.Infof("Metrics will be exported to Influx with host=%s, db=%s, username=%s", cfg.Influx.Host, cfg.Influx.Database, cfg.Influx.Username)
		influxdb.InfluxDB(
			m.Registry,          // metrics registry
			time.Second*10,      // interval
			cfg.Influx.Host,     // the InfluxDB url
			cfg.Influx.Database, // your InfluxDB database
			cfg.Influx.Username, // your InfluxDB user
			cfg.Influx.Password, // your InfluxDB password
		)
	}
	return
}

//func (m InfluxMetrics) Increment(metricName string, start *time.Time, value string) {
//	switch metricName {
//	case "puts.current_url.request_duration":
//		m.Puts.Duration.UpdateSince(*start)
//	case "puts.current_url.error_count":
//		m.Puts.Errors.Mark(1)
//	case "puts.current_url.bad_request_count":
//		m.Puts.BadRequest.Mark(1)
//	case "puts.current_url.request_count":
//		m.Puts.Request.Mark(1)
//	case "gets.current_url.request_duration":
//		m.Gets.Duration.UpdateSince(*start)
//	case "gets.current_url.error_count":
//		m.Gets.Errors.Mark(1)
//	case "gets.current_url.bad_request_count":
//		m.Gets.BadRequest.Mark(1)
//	case "gets.current_url.request_count":
//		m.Gets.Request.Mark(1)
//	case "puts.backend.request_duration":
//		m.PutsBackend.Duration.UpdateSince(*start)
//	case "puts.backend.error_count":
//		m.PutsBackend.Errors.Mark(1)
//	case "puts.backend.bad_request_count":
//		m.PutsBackend.BadRequest.Mark(1)
//	case "puts.backend.json_request_count":
//		m.PutsBackend.JsonRequest.Mark(1)
//	case "puts.backend.xml_request_count":
//		m.PutsBackend.XmlRequest.Mark(1)
//	case "puts.backend.defines_ttl":
//		m.PutsBackend.DefinesTTL.Mark(1)
//	case "puts.backend.unknown_request_count":
//		m.PutsBackend.InvalidRequest.Mark(1)
//	case "puts.backend.request_size_bytes":
//		m.PutsBackend.RequestLength.Update(int64(len(value)))
//	case "gets.backend.request_duration":
//		m.GetsBackend.Duration.UpdateSince(*start)
//	case "gets.backend.error_count":
//		m.GetsBackend.Errors.Mark(1)
//	case "gets.backend.bad_request_count":
//		m.GetsBackend.BadRequest.Mark(1)
//	case "gets.backend.request_count":
//		m.GetsBackend.Request.Mark(1)
//	case "connections.active_incoming":
//		m.Connections.ActiveConnections.Inc(1)
//	case "connections.close_errors":
//		m.Connections.ConnectionCloseErrors.Mark(1)
//	case "connections.accept_errors":
//		m.Connections.ConnectionAcceptErrors.Mark(1)
//	default:
//		//error
//	}
//}
//
//func (m InfluxMetrics) Decrement(metricName string) {
//	switch metricName {
//	case "connections.active_incoming":
//		m.Connections.ActiveConnections.Dec(1)
//	default:
//		//error
//	}
//}

/**************************************************
 *	NEW Functions to record metrics
 **************************************************/
func (m *InfluxMetrics) RecordPutRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case "error":
			m.Puts.Errors.Mark(1)
		case "bad_request":
			m.Puts.BadRequest.Mark(1)
		case "add":
			m.Puts.Request.Mark(1)
		}
	} else if duration != nil {
		m.Puts.Duration.UpdateSince(*duration)
	}
}

func (m *InfluxMetrics) RecordGetRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case "error":
			m.Gets.Errors.Mark(1)
		case "bad_request":
			m.Gets.BadRequest.Mark(1)
		case "add":
			m.Gets.Request.Mark(1)
		}
	} else if duration != nil {
		m.Gets.Duration.UpdateSince(*duration)
	}
}

func (m *InfluxMetrics) RecordPutBackendRequest(status string, duration *time.Time, sizeInBytes float64) {
	if duration != nil {
		m.PutsBackend.Duration.UpdateSince(*duration)
	}
	switch status {
	case "error":
		m.PutsBackend.Errors.Mark(1)
	case "bad_request":
		m.PutsBackend.BadRequest.Mark(1)
	case "json":
		m.PutsBackend.JsonRequest.Mark(1)
	case "xml":
		m.PutsBackend.XmlRequest.Mark(1)
	case "defines_ttl":
		m.PutsBackend.DefinesTTL.Mark(1)
	case "invalid_format":
		m.PutsBackend.InvalidRequest.Mark(1)
	}
	if value > 0 {
		m.PutsBackend.RequestLength.Update(int64(len(value)))
	}
}

func (m *InfluxMetrics) RecordGetBackendRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case "error":
			m.GetsBackend.Errors.Mark(1)
		case "bad_request":
			m.GetsBackend.BadRequest.Mark(1)
		case "add":
			m.GetsBackend.Request.Mark(1)
		}
	} else if duration != nil {
		m.GetsBackend.Duration.UpdateSince(*duration)
	}
}

func (m *InfluxMetrics) RecordConnectionMetrics(label string) {
	switch label {
	case "add":
		m.Connections.ActiveConnections.Inc(1)
	case "substract":
		m.Connections.ActiveConnections.Dec(1)
	case "close":
		m.Connections.ConnectionCloseErrors.Mark(1)
	case "accept":
		m.Connections.ConnectionAcceptErrors.Mark(1)
	}
}

func (m *InfluxMetrics) RecordExtraTTLSeconds(value float64) {
	//ExtraTTLSeconds *prometheus.HistogramVec
	//m.GetsBackend.Duration.UpdateSince(*start)
	m.PrometheusExtraTTLMetrics.Observe(value)
}

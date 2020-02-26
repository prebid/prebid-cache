package metrics

import (
	"fmt"
	"time"

	"github.com/prebid/prebid-cache/config"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/go-metrics-influxdb"
)

// Constants and global variables
const TenSeconds time.Duration = time.Second * 10
const AddLabel string = "add"
const ErrorLabel string = "error"
const BadRequestLabel string = "bad_request"
const JsonLabel string = "json"
const XmlLabel string = "xml"
const DefinesTTLLabel string = "defines_ttl"
const InvFormatLabel string = "invalid_format"
const SubstractLabel string = "substract"
const CloseLabel string = "close"
const AcceptLabel string = "accept"

//	Object definition
type InfluxMetrics struct {
	Registry    metrics.Registry
	Puts        *InfluxMetricsEntry
	Gets        *InfluxMetricsEntry
	PutsBackend *InfluxMetricsEntryByFormat
	GetsBackend *InfluxMetricsEntry
	Connections *InfluxConnectionMetrics
	ExtraTTL    *InfluxExtraTTL
}

type InfluxMetricsEntry struct {
	Duration   metrics.Timer
	Errors     metrics.Meter
	BadRequest metrics.Meter
	Request    metrics.Meter
}

type InfluxMetricsEntryByFormat struct {
	Duration       metrics.Timer
	Request        metrics.Meter
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
type InfluxExtraTTL struct {
	ExtraTTLSeconds metrics.Histogram
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
		Registry:    r,
		Puts:        NewInfluxMetricsEntry("puts.current_url", r),
		Gets:        NewInfluxMetricsEntry("gets.current_url", r),
		PutsBackend: NewInfluxMetricsEntryBackendPuts("puts.backend", r),
		GetsBackend: NewInfluxMetricsEntry("gets.backend", r),
		Connections: NewInfluxConnectionMetrics(r),
		ExtraTTL:    &InfluxExtraTTL{ExtraTTLSeconds: metrics.GetOrRegisterHistogram("extra_ttl_seconds", r, metrics.NewUniformSample(5000))},
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
			m.Registry,
			TenSeconds,
			cfg.Influx.Host,
			cfg.Influx.Database,
			cfg.Influx.Username,
			cfg.Influx.Password,
		)
	}
	return
}

func (m *InfluxMetrics) RecordPutRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case ErrorLabel:
			m.Puts.Errors.Mark(1)
		case BadRequestLabel:
			m.Puts.BadRequest.Mark(1)
		case AddLabel:
			m.Puts.Request.Mark(1)
		}
	} else if duration != nil {
		m.Puts.Duration.UpdateSince(*duration)
	}
}

func (m *InfluxMetrics) RecordGetRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case ErrorLabel:
			m.Gets.Errors.Mark(1)
		case BadRequestLabel:
			m.Gets.BadRequest.Mark(1)
		case AddLabel:
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
	case AddLabel:
		m.PutsBackend.Request.Mark(1)
	case ErrorLabel:
		m.PutsBackend.Errors.Mark(1)
	case BadRequestLabel:
		m.PutsBackend.BadRequest.Mark(1)
	case JsonLabel:
		m.PutsBackend.JsonRequest.Mark(1)
	case XmlLabel:
		m.PutsBackend.XmlRequest.Mark(1)
	case DefinesTTLLabel:
		m.PutsBackend.DefinesTTL.Mark(1)
	case InvFormatLabel:
		m.PutsBackend.InvalidRequest.Mark(1)
	}
	if sizeInBytes > 0 {
		m.PutsBackend.RequestLength.Update(int64(sizeInBytes))
	}
}

func (m *InfluxMetrics) RecordGetBackendRequest(status string, duration *time.Time) {
	if status != "" {
		switch status {
		case ErrorLabel:
			m.GetsBackend.Errors.Mark(1)
		case BadRequestLabel:
			m.GetsBackend.BadRequest.Mark(1)
		case AddLabel:
			m.GetsBackend.Request.Mark(1)
		}
	} else if duration != nil {
		m.GetsBackend.Duration.UpdateSince(*duration)
	}
}

func (m *InfluxMetrics) RecordConnectionMetrics(label string) {
	switch label {
	case AddLabel:
		m.Connections.ActiveConnections.Inc(1)
	case SubstractLabel:
		m.Connections.ActiveConnections.Dec(1)
	case CloseLabel:
		m.Connections.ConnectionCloseErrors.Mark(1)
	case AcceptLabel:
		m.Connections.ConnectionAcceptErrors.Mark(1)
	}
}

func (m *InfluxMetrics) RecordExtraTTLSeconds(value float64) {
	m.ExtraTTL.ExtraTTLSeconds.Update(int64(value))
}

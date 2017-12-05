package metrics

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/vrischmann/go-metrics-influxdb"
	"log"
	"os"
	"time"
)

type MetricsEntry struct {
	Duration   metrics.Timer
	Errors     metrics.Meter
	BadRequest metrics.Meter
	Request    metrics.Meter
}

type MetricsEntryByFormat struct {
	Duration       metrics.Timer
	Errors         metrics.Meter
	BadRequest     metrics.Meter
	JsonRequest    metrics.Meter
	XmlRequest     metrics.Meter
	InvalidRequest metrics.Meter
}

func NewMetricsEntry(name string, r metrics.Registry) *MetricsEntry {
	return &MetricsEntry{
		Duration:   metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
		BadRequest: metrics.GetOrRegisterMeter(fmt.Sprintf("%s.bad_request_count", name), r),
		Request:    metrics.GetOrRegisterMeter(fmt.Sprintf("%s.request_count", name), r),
	}
}

func NewMetricsEntryByType(name string, r metrics.Registry) *MetricsEntryByFormat {
	return &MetricsEntryByFormat{
		Duration:       metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:         metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
		BadRequest:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.bad_request_count", name), r),
		JsonRequest:    metrics.GetOrRegisterMeter(fmt.Sprintf("%s.json_request_count", name), r),
		XmlRequest:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.xml_request_count", name), r),
		InvalidRequest: metrics.GetOrRegisterMeter(fmt.Sprintf("%s.unknown_request_count", name), r),
	}
}

type Metrics struct {
	Registry    metrics.Registry
	Puts        *MetricsEntry
	Gets        *MetricsEntry
	PutsBackend *MetricsEntryByFormat
	GetsBackend *MetricsEntry
}

// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m *Metrics) Export() {
	metricsTarget := viper.GetString("metrics.target")
	if metricsTarget == "none" {
		return
	}
	if metricsTarget == "stderr" {
		metrics.Log(m.Registry, time.Second*10, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
		return
	}
	// Preserve old behavior by defaulting here.
	influxdb.InfluxDB(
		m.Registry,                          // metrics registry
		time.Second*10,                      // interval
		viper.GetString("metrics.host"),     // the InfluxDB url
		viper.GetString("metrics.database"), // your InfluxDB database
		viper.GetString("metrics.username"), // your InfluxDB user
		viper.GetString("metrics.password"), // your InfluxDB password
	)
}

func CreateMetrics() *Metrics {
	flushTime := time.Second * 10
	r := metrics.NewPrefixedRegistry("prebidcache.")
	m := &Metrics{
		Registry:    r,
		Puts:        NewMetricsEntry("puts.current_url", r),
		Gets:        NewMetricsEntry("gets.current_url", r),
		PutsBackend: NewMetricsEntryByType("puts.backend", r),
		GetsBackend: NewMetricsEntry("gets.backend", r),
	}

	metrics.RegisterDebugGCStats(m.Registry)
	metrics.RegisterRuntimeMemStats(m.Registry)

	go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	go metrics.CaptureDebugGCStats(m.Registry, flushTime)

	return m
}

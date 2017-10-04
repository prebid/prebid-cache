package metrics

import (
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/vrischmann/go-metrics-influxdb"
	"time"
)

type MetricsEntry struct {
	Request    metrics.Meter
	Duration   metrics.Timer
	Errors     metrics.Meter
	BadRequest metrics.Meter
}

func newMetricsEntry(name string, r metrics.Registry) *MetricsEntry {
	me := &MetricsEntry{
		Request:    metrics.GetOrRegisterMeter(fmt.Sprintf("%s.request_count", name), r),
		Duration:   metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:     metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
		BadRequest: metrics.GetOrRegisterMeter(fmt.Sprintf("%s.bad_request_count", name), r),
	}

	return me
}

type Metrics struct {
	Registry    metrics.Registry
	Puts        *MetricsEntry
	Gets        *MetricsEntry
	PutsBackend *MetricsEntry
	GetsBackend *MetricsEntry
}

// Export begins sending metrics to the configured database.
// This method blocks indefinitely, so it should probably be run in a goroutine.
func (m *Metrics) Export() {
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
		Puts:        newMetricsEntry("puts.current_url", r),
		Gets:        newMetricsEntry("gets.current_url", r),
		PutsBackend: newMetricsEntry("puts.backend", r),
		GetsBackend: newMetricsEntry("gets.backend", r),
	}

	metrics.RegisterDebugGCStats(m.Registry)
	metrics.RegisterRuntimeMemStats(m.Registry)

	go metrics.CaptureRuntimeMemStats(m.Registry, flushTime)
	go metrics.CaptureDebugGCStats(m.Registry, flushTime)

	return m
}

package metrics

import (
	"github.com/prebid/prebid-cache/config"
	//"github.com/prometheus/client_golang/prometheus"
	//"github.com/rcrowley/go-metrics"
	"time"
)

/* Object to access metric engines     */
type Metrics struct {
	MetricEngines []CacheMetrics
}

/* Methods so the metrics object executes the methods of the `CacheMetrics` interface    */
func (m Metrics) RecPutRequest(status string, duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordPutRequest(status, duration)
	}
}
func (m Metrics) RecGetRequest(status string, duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordGetRequest(status, duration)
	}
}
func (m Metrics) RecPutBackendRequest(status string, duration *time.Time, sizeInBytes float64) {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendRequest(status, duration, sizeInBytes)
	}
}
func (m Metrics) RecGetBackendRequest(status string, duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordGetBackendRequest(status, duration)
	}
}
func (m Metrics) RecConnectionMetrics(status string) {
	for _, me := range m.MetricEngines {
		me.RecordConnectionMetrics(status)
	}
}
func (m Metrics) RecExtraTTLSeconds(value float64) {
	for _, me := range m.MetricEngines {
		me.RecordExtraTTLSeconds(value)
	}
}

func (m Metrics) Export(cfg config.Configuration) {
	for _, me := range m.MetricEngines {
		me.Export(cfg.Metrics)
	}
}

/* Interface definition                */
type CacheMetrics interface {
	RecordPutRequest(status string, duration *time.Time)
	RecordGetRequest(status string, duration *time.Time)
	RecordPutBackendRequest(status string, duration *time.Time, sizeInBytes float64)
	RecordGetBackendRequest(status string, duration *time.Time)
	RecordConnectionMetrics(label string)
	RecordExtraTTLSeconds(aVar float64)
	Export(cfg config.Metrics)
}

func CreateMetrics(cfg config.Configuration) *Metrics {
	engineList := make([]CacheMetrics, 0, 2)

	if cfg.Metrics.Influx.Host != "" {
		//returnEngines.Influx = CreateInfluxMetrics()
		engineList = append(engineList, CreateInfluxMetrics())
	}
	if cfg.Metrics.Prometheus.Port != 0 {
		//returnEngines.Prometheus = CreatePrometheusMetrics(cfg.Metrics.Prometheus)
		engineList = append(engineList, CreatePrometheusMetrics(cfg.Metrics.Prometheus))
	}
	return &Metrics{MetricEngines: engineList}
}

//func (cacheMetrics CacheMetricsEngines) Add(metricName string, start *time.Time, value string) {
//	if cacheMetrics.Influx != nil {
//		cacheMetrics.Influx.Increment(metricName, start, value)
//	}
//	if cacheMetrics.Prometheus != nil {
//		cacheMetrics.Prometheus.Increment(metricName, start, value)
//	}
//}
//
//func (cacheMetrics CacheMetricsEngines) Substract(metricName string) {
//	if cacheMetrics.Influx != nil {
//		cacheMetrics.Influx.Decrement(metricName)
//	}
//	if cacheMetrics.Prometheus != nil {
//		cacheMetrics.Prometheus.Decrement(metricName)
//	}
//}

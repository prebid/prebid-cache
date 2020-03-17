package metrics

import (
	"github.com/prebid/prebid-cache/config"
	influx "github.com/prebid/prebid-cache/metrics/influx"
	prometheus "github.com/prebid/prebid-cache/metrics/prometheus"
	"time"
)

// Metrics provides access to metric engines.
type Metrics struct {
	MetricEngines []CacheMetrics
}

// Methods so the metrics object executes the methods of the `CacheMetrics` interface
func (m Metrics) RecordPutError() {
	for _, me := range m.MetricEngines {
		me.RecordPutError()
	}
}

func (m Metrics) RecordPutBadRequest() {
	for _, me := range m.MetricEngines {
		me.RecordPutBadRequest()
	}
}

func (m Metrics) RecordPutTotal() {
	for _, me := range m.MetricEngines {
		me.RecordPutTotal()
	}
}

func (m Metrics) RecordPutDuration(duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordPutDuration(duration)
	}
}

func (m Metrics) RecordGetError() {
	for _, me := range m.MetricEngines {
		me.RecordGetError()
	}
}

func (m Metrics) RecordGetBadRequest() {
	for _, me := range m.MetricEngines {
		me.RecordGetBadRequest()
	}
}

func (m Metrics) RecordGetTotal() {
	for _, me := range m.MetricEngines {
		me.RecordGetTotal()
	}
}

func (m Metrics) RecordGetDuration(duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordGetDuration(duration)
	}
}

func (m Metrics) RecordPutBackendXml() {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendXml()
	}
}

func (m Metrics) RecordPutBackendJson() {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendJson()
	}
}

func (m Metrics) RecordPutBackendInvalid() {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendInvalid()
	}
}

func (m Metrics) RecordPutBackendDefTTL() {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendDefTTL()
	}
}

func (m Metrics) RecordPutBackendDuration(duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendDuration(duration)
	}
}

func (m Metrics) RecordPutBackendError() {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendError()
	}
}

func (m Metrics) RecordPutBackendSize(sizeInBytes float64) {
	for _, me := range m.MetricEngines {
		me.RecordPutBackendSize(sizeInBytes)
	}
}

func (m Metrics) RecordGetBackendDuration(duration *time.Time) {
	for _, me := range m.MetricEngines {
		me.RecordGetBackendDuration(duration)
	}
}

func (m Metrics) RecordGetBackendTotal() {
	for _, me := range m.MetricEngines {
		me.RecordGetBackendTotal()
	}
}

func (m Metrics) RecordGetBackendError() {
	for _, me := range m.MetricEngines {
		me.RecordGetBackendError()
	}
}

func (m Metrics) RecordConnectionOpen() {
	for _, me := range m.MetricEngines {
		me.RecordConnectionOpen()
	}
}

func (m Metrics) RecordConnectionClosed() {
	for _, me := range m.MetricEngines {
		me.RecordConnectionClosed()
	}
}

func (m Metrics) RecordCloseConnectionErrors() {
	for _, me := range m.MetricEngines {
		me.RecordCloseConnectionErrors()
	}
}

func (m Metrics) RecordAcceptConnectionErrors() {
	for _, me := range m.MetricEngines {
		me.RecordAcceptConnectionErrors()
	}
}

func (m Metrics) RecordExtraTTLSeconds(value float64) {
	for _, me := range m.MetricEngines {
		me.RecordExtraTTLSeconds(value)
	}
}

func (m Metrics) Export(cfg config.Configuration) {
	for _, me := range m.MetricEngines {
		me.Export(cfg.Metrics)
	}
}

// CacheMetrics Interface
type CacheMetrics interface {
	Export(cfg config.Metrics)
	RecordPutError()
	RecordPutBadRequest()
	RecordPutTotal()
	RecordPutDuration(duration *time.Time)
	RecordGetError()
	RecordGetBadRequest()
	RecordGetTotal()
	RecordGetDuration(duration *time.Time)
	RecordPutBackendXml()
	RecordPutBackendJson()
	RecordPutBackendInvalid()
	RecordPutBackendDefTTL()
	RecordPutBackendDuration(duration *time.Time)
	RecordPutBackendError()
	RecordPutBackendSize(sizeInBytes float64)
	RecordGetBackendTotal()
	RecordGetBackendDuration(duration *time.Time)
	RecordGetBackendError()
	RecordConnectionOpen()
	RecordConnectionClosed()
	RecordCloseConnectionErrors()
	RecordAcceptConnectionErrors()
	RecordExtraTTLSeconds(value float64)
}

func CreateMetrics(cfg config.Configuration) *Metrics {
	engineList := make([]CacheMetrics, 0, 2)

	if cfg.Metrics.Influx.Enabled {
		engineList = append(engineList, influx.CreateInfluxMetrics())
	}
	if cfg.Metrics.Prometheus.Enabled {
		engineList = append(engineList, prometheus.CreatePrometheusMetrics(cfg.Metrics.Prometheus))
	}
	return &Metrics{MetricEngines: engineList}
}

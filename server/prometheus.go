package server

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/prebid/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
)

func newPrometheusServer(cfg *config.Configuration, promRegistry *prometheus.Registry) *http.Server {
	if promRegistry == nil {
		log.Errorf("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
	}
	return &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Metrics.Prometheus.Port),
		Handler: promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{
			ErrorLog:            loggerForPrometheus{},
			MaxRequestsInFlight: 5,
			Timeout:             cfg.Metrics.Prometheus.Timeout(),
		}),
	}
}

type loggerForPrometheus struct{}

func (loggerForPrometheus) Println(v ...interface{}) {
	log.Warning(v...)
}

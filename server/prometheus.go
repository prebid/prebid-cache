package server

import (
	"fmt"
	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
)

func newPrometheusServer(cfg *config.Configuration, promRegistry *prometheus.Registry) *http.Server {
	if promRegistry == nil {
		logger.Error("Prometheus metrics configured, but a Prometheus metrics engine was not found. Cannot set up a Prometheus listener.")
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
	if len(v) == 0 {
		return
	}

	if len(v) == 1 {
		logger.Warn(fmt.Sprintf("%v", v[0]))
	} else {
		logger.Warn(fmt.Sprintf("%v", v[0]), v[1:]...)
	}
}

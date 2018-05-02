package main

import (
	_ "net/http/pprof"
	"os"

	log "github.com/Sirupsen/logrus"

	backendConfig "github.com/prebid/prebid-cache/backends/config"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints/routing"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/server"
)

func main() {
	cfg := config.NewConfig()
	initLogging(cfg)
	appMetrics := metrics.CreateMetrics()
	backend := backendConfig.NewBackend(cfg, appMetrics)
	handler := routing.NewHandler(cfg, backend, appMetrics)
	go appMetrics.Export(cfg.Metrics)
	server.Listen(cfg, handler, appMetrics.Connections)
}

func initLogging(cfg config.Configuration) {
	level, err := log.ParseLevel(string(cfg.Log.Level))
	if err != nil {
		log.Fatalf("Invalid logrus level: %v", err)
	}
	log.SetOutput(os.Stdout)
	log.SetLevel(level)
	log.Info("Log level set to: ", log.GetLevel())
}

package main

import (
	_ "net/http/pprof"
	"os"

	log "github.com/sirupsen/logrus"

	backendConfig "github.com/prebid/prebid-cache/backends/config"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints/routing"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/server"
)

func main() {
	log.SetOutput(os.Stdout)
	cfg := config.NewConfig("config")
	setLogLevel(cfg.Log.Level)
	cfg.ValidateAndLog()

	appMetrics := metrics.CreateMetrics(cfg)
	backend := backendConfig.NewBackend(cfg, appMetrics)
	publicHandler := routing.NewPublicHandler(cfg, backend, appMetrics)
	adminHandler := routing.NewAdminHandler(cfg, backend, appMetrics)
	go appMetrics.Export(cfg)
	server.Listen(cfg, publicHandler, adminHandler, appMetrics)
}

func setLogLevel(logLevel config.LogLevel) {
	level, err := log.ParseLevel(string(logLevel))
	if err != nil {
		log.Fatalf("Invalid logrus level: %v", err)
	}
	log.SetLevel(level)
}

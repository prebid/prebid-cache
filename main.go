package main

import (
	_ "net/http/pprof"

	backendConfig "github.com/PubMatic-OpenWrap/prebid-cache/backends/config"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/PubMatic-OpenWrap/prebid-cache/endpoints/routing"
	"github.com/PubMatic-OpenWrap/prebid-cache/metrics"
	"github.com/PubMatic-OpenWrap/prebid-cache/server"
)

const configFileName = "config"

func main() {

	//log.SetOutput(os.Stdout)
	cfg := config.NewConfig(configFileName)
	//setLogLevel(cfg.Log.Level)
	cfg.ValidateAndLog()

	appMetrics := metrics.CreateMetrics(cfg)
	backend := backendConfig.NewBackend(cfg, appMetrics)
	publicHandler := routing.NewPublicHandler(cfg, backend, appMetrics)
	adminHandler := routing.NewAdminHandler(cfg, backend, appMetrics)
	go appMetrics.Export(cfg)
	server.Listen(cfg, publicHandler, adminHandler, appMetrics)
}

/*func setLogLevel(logLevel config.LogLevel) {
	level, err := log.ParseLevel(string(logLevel))
	if err != nil {
		log.Fatalf("Invalid logrus level: %v", err)
	}
	log.SetLevel(level)
}*/

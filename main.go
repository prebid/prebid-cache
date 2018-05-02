package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/didip/tollbooth"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"

	"github.com/didip/tollbooth/limiter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints"
	endpointDecorators "github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/server"
)

func initRateLimter(next http.Handler, cfg config.RateLimiting) http.Handler {
	// Sip rate limiter when disabled
	if !cfg.Enabled {
		return next
	}

	limit := tollbooth.NewLimiter(cfg.MaxRequestsPerSecond, time.Second, &limiter.ExpirableOptions{
		DefaultExpirationTTL: 1 * time.Hour,
	})
	limit.SetIPLookups([]string{"X-Forwarded-For", "X-Real-IP"})
	limit.SetMessage(`{ "error": "rate limit" }`)
	limit.SetMessageContentType("application/json")

	return tollbooth.LimitHandler(limit, next)
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

func main() {
	cfg := config.NewConfig()
	initLogging(cfg)
	appMetrics := metrics.CreateMetrics()

	backend := backends.NewBackend(cfg.Backend)
	if cfg.RequestLimits.MaxSize > 0 {
		backend = backendDecorators.EnforceSizeLimit(backend, cfg.RequestLimits.MaxSize)
	}
	backend = backendDecorators.LogMetrics(backend, appMetrics)
	if viper.GetString("compression.type") == "snappy" {
		backend = compression.SnappyCompress(backend)
	}

	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.

	router.POST("/cache", endpointDecorators.MonitorHttp(endpoints.NewPutHandler(backend, cfg.RequestLimits.MaxNumValues), appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(endpoints.NewGetHandler(backend), appMetrics.Gets))

	go appMetrics.Export(cfg.Metrics)

	server.Listen(cfg, router, appMetrics.Connections)
}

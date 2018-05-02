package main

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/spf13/viper"

	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints"
	endpointDecorators "github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/server"
)

func main() {
	cfg := config.NewConfig()
	initLogging(cfg)
	appMetrics := metrics.CreateMetrics()
	backend := newBackend(cfg, appMetrics)
	handler := newHandler(cfg, backend, appMetrics)
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

func newBackend(cfg config.Configuration, appMetrics *metrics.Metrics) backends.Backend {
	backend := backends.NewBackend(cfg.Backend)
	if cfg.RequestLimits.MaxSize > 0 {
		backend = backendDecorators.EnforceSizeLimit(backend, cfg.RequestLimits.MaxSize)
	}
	backend = backendDecorators.LogMetrics(backend, appMetrics)
	if viper.GetString("compression.type") == "snappy" {
		backend = compression.SnappyCompress(backend)
	}
	return backend
}

func newHandler(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics) http.Handler {
	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.
	router.POST("/cache", endpointDecorators.MonitorHttp(endpoints.NewPutHandler(dataStore, cfg.RequestLimits.MaxNumValues), appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(endpoints.NewGetHandler(dataStore), appMetrics.Gets))

	handler := handleCors(router)
	handler = handleRateLimiting(handler, cfg.RateLimiting)
	return handler
}

func handleCors(handler http.Handler) http.Handler {
	coresCfg := cors.New(cors.Options{AllowCredentials: true})
	return coresCfg.Handler(handler)
}

func handleRateLimiting(next http.Handler, cfg config.RateLimiting) http.Handler {
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

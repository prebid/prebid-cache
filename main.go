package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/didip/tollbooth"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"

	"os/signal"
	"syscall"

	"github.com/didip/tollbooth/limiter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints"
	endpointDecorators "github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
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

func initLogging(cfg *config.Configuration) {
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
	if cfg.Compression.Type == config.CompressionSnappy {
		backend = compression.SnappyCompress(backend)
	}
	backend = backendDecorators.LogMetrics(backend, appMetrics)

	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.

	router.POST("/cache", endpointDecorators.MonitorHttp(endpoints.NewPutHandler(backend, cfg.RequestLimits), appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(endpoints.NewGetHandler(backend), appMetrics.Gets))

	go appMetrics.Export(cfg.Metrics)

	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	adminURI := fmt.Sprintf(":%d", cfg.AdminPort)
	fmt.Println("Admin running on: ", adminURI)
	adminServer := &http.Server{Addr: adminURI, Handler: nil}
	go (func() {
		err := adminServer.ListenAndServe()
		log.Errorf("Admin server failure: %v", err)
		stopSignals <- syscall.SIGTERM
	})()

	coresCfg := cors.New(cors.Options{AllowCredentials: true})
	corsRouter := coresCfg.Handler(router)

	limitHandler := initRateLimter(corsRouter, cfg.RateLimiting)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      limitHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go (func() {
		log.Info("Starting server on port: ", server.Addr)
		err := server.ListenAndServe()
		log.Errorf("Main server failure: %v", err)
		stopSignals <- syscall.SIGTERM
	})()

	<-stopSignals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Failed to shut down server: %v", err)
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		log.Errorf("Failed to shut down admin server: %v", err)
	}
}

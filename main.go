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
	"github.com/spf13/viper"

	"os/signal"
	"syscall"

	"github.com/didip/tollbooth/limiter"
	"github.com/prebid/prebid-cache/backends"
	backendDecorators "github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/endpoints"
	endpointDecorators "github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
)

func initRateLimter(next http.Handler) http.Handler {
	viper.SetDefault("rate_limiter.enabled", true)
	viper.SetDefault("rate_limiter.num_requests", 100)

	// Sip rate limiter when disabled
	if viper.GetBool("rate_limiter.enabled") != true {
		return next
	}

	viper.SetDefault("request_limits.max_size_bytes", 10*1024)
	viper.SetDefault("request_limits.max_num_values", 10)

	limit := tollbooth.NewLimiter(viper.GetInt64("rate_limiter.num_requests"), time.Second, &limiter.ExpirableOptions{
		DefaultExpirationTTL: 1 * time.Hour,
	})
	limit.SetIPLookups([]string{"X-Forwarded-For", "X-Real-IP"})
	limit.SetMessage(`{ "error": "rate limit" }`)
	limit.SetMessageContentType("application/json")

	return tollbooth.LimitHandler(limit, next)
}

func main() {
	viper.SetConfigName("config")              // name of config file (without extension)
	viper.AddConfigPath("/etc/prebid-cache/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.prebid-cache") // call multiple times to add many search paths
	viper.AddConfigPath(".")                   // optionally look for config in the working directory
	err := viper.ReadInConfig()                // Find and read the config file
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	level, err := log.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(level)
	log.Info("Setting log level to: ", log.GetLevel())

	port := viper.GetInt("port")

	appMetrics := metrics.CreateMetrics()

	backend := backends.NewBackend(viper.GetString("backend.type"))
	if maxSize := viper.GetInt("request_limits.max_size_bytes"); maxSize > 0 {
		backend = backendDecorators.EnforceSizeLimit(backend, maxSize)
	}
	if viper.GetString("compression.type") == "snappy" {
		backend = compression.SnappyCompress(backend)
	}
	backend = backendDecorators.LogMetrics(backend, appMetrics)

	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.

	router.POST("/cache", endpointDecorators.MonitorHttp(endpoints.NewPutHandler(backend, viper.GetInt("request_limits.max_num_values")), appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(endpoints.NewGetHandler(backend), appMetrics.Gets))
	go appMetrics.Export()

	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	adminURI := fmt.Sprintf(":%s", viper.GetString("admin_port"))
	fmt.Println("Admin running on: ", adminURI)
	adminServer := &http.Server{Addr: adminURI, Handler: nil}
	go (func() {
		err := adminServer.ListenAndServe()
		log.Errorf("Admin server failure: %v", err)
		stopSignals <- syscall.SIGTERM
	})()

	coresCfg := cors.New(cors.Options{AllowCredentials: true})
	corsRouter := coresCfg.Handler(router)

	limitHandler := initRateLimter(corsRouter)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
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

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
	"github.com/prebid/prebid-cache/endpoints"
	endpointDecorators "github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/server"
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

	appMetrics := metrics.CreateMetrics()

	backend := backends.NewBackend(viper.GetString("backend.type"))
	if maxSize := viper.GetInt("request_limits.max_size_bytes"); maxSize > 0 {
		backend = backendDecorators.EnforceSizeLimit(backend, maxSize)
	}
	backend = backendDecorators.LogMetrics(backend, appMetrics)
	if viper.GetString("compression.type") == "snappy" {
		backend = compression.SnappyCompress(backend)
	}

	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.

	router.POST("/cache", endpointDecorators.MonitorHttp(endpoints.NewPutHandler(backend, viper.GetInt("request_limits.max_num_values")), appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(endpoints.NewGetHandler(backend), appMetrics.Gets))
	go appMetrics.Export()

	server.Listen(viper.GetInt("port"), viper.GetInt("admin_port"), router, appMetrics.Connections)
}

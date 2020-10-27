package routing

import (
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/endpoints"
	"github.com/prebid/prebid-cache/endpoints/decorators"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/rs/cors"
)

func NewAdminHandler(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics) http.Handler {
	router := httprouter.New()
	addReadRoutes(cfg, dataStore, appMetrics, router)
	addWriteRoutes(cfg, dataStore, appMetrics, router)
	return router
}

func NewPublicHandler(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics) http.Handler {
	router := httprouter.New()
	addReadRoutes(cfg, dataStore, appMetrics, router)
	if cfg.Routes.AllowPublicWrite {
		addWriteRoutes(cfg, dataStore, appMetrics, router)
	}

	handler := handleCors(router)
	handler = handleRateLimiting(handler, cfg.RateLimiting)
	return handler
}

func addReadRoutes(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics, router *httprouter.Router) {
	router.GET("/", endpoints.NewIndexHandler(cfg.IndexResponse)) //Default route handler
	router.GET("/status", endpoints.Status)                       // Determines whether the server is ready for more traffic.
	router.GET("/cache", decorators.MonitorHttp(endpoints.NewGetHandler(dataStore, cfg.RequestLimits.AllowSettingKeys), appMetrics, decorators.GetMethod))
}

func addWriteRoutes(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics, router *httprouter.Router) {
	router.POST("/cache", decorators.MonitorHttp(endpoints.NewPutHandler(dataStore, cfg.RequestLimits.MaxNumValues, cfg.RequestLimits.AllowSettingKeys), appMetrics, decorators.PostMethod))
}

func handleCors(handler http.Handler) http.Handler {
	coresCfg := cors.New(cors.Options{AllowCredentials: true, AllowOriginFunc: func(origin string) bool {
		return true
	}})
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

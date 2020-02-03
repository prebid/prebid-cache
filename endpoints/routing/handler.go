package routing

import (
	"net/http"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-cache/backends"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/PubMatic-OpenWrap/prebid-cache/endpoints"
	"github.com/PubMatic-OpenWrap/prebid-cache/endpoints/decorators"
	"github.com/PubMatic-OpenWrap/prebid-cache/metrics"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/julienschmidt/httprouter"
)

func NewHandler(cfg config.Configuration, dataStore backends.Backend, appMetrics *metrics.Metrics) http.Handler {
	router := httprouter.New()
	router.GET("/status", endpoints.Status) // Determines whether the server is ready for more traffic.
	router.POST("/cache", decorators.MonitorHttp(endpoints.NewPutHandler(dataStore, cfg.RequestLimits.MaxNumValues), appMetrics.Puts))
	router.GET("/cache", decorators.MonitorHttp(endpoints.NewGetHandler(dataStore), appMetrics.Gets))
	router.GET("/healthcheck", endpoints.HealthCheck) // Determines whether the server is up and running.

	handler := handleCors(router)
	handler = handleRateLimiting(handler, cfg.RateLimiting)
	return handler
}

func handleCors(handler http.Handler) http.Handler {
	return corsHandler(handler)
}

func corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if len(origin) == 0 {
			origin = "*"
		} else {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)

		h.ServeHTTP(w, r)
	})
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

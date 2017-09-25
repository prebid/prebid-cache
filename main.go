package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/didip/tollbooth"
	"github.com/julienschmidt/httprouter"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	influxdb "github.com/vrischmann/go-metrics-influxdb"
	"strings"

	"github.com/didip/tollbooth/limiter"
	"sync"
	"os/signal"
	"syscall"
)

// When we insert a value into the cache, we'll prefix it with one of these.
// When we fetch a value from the cache, we'll trim these back off and use
// the info to determine the MIME type of our response.
const (
	XML_PREFIX  = "xml"
	JSON_PREFIX = "json"
)

// This status code signals that we're having trouble reaching a dependent service (currently Azure).
// This service sits behind an nginx load balancer which considers the normal 500 and 504 errors to
// be a sign of bad service health. If te service responds with these, it will stop forwarding traffic
// in case the service is dying.
//
// However... we're running behind Kubernetes. The Horizontal Pod Autoscaler should take care of
// "an overwhelmed service" by allocating more machines. If nginx scales back the traffic, the HPA
// scales *down* those machines... and creates a vicious cycle.
//
// Kurt Adam says he's working on a solution for this... but until it's ready, we'll use this
// non-standard 5xx response to dodge nginx if Azure times out.
const httpDependencyTimeout = 597

var (
	MaxValueLength = 1024 * 10
	MaxNumValues   = 10
)

type GetResponse struct {
	Value interface{} `json:"value"`
}

type PutAnyObject struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type PutAnyRequest struct {
	Puts []PutAnyObject `json:"puts"`
}

type PutObject struct {
	Value string `json:"value"`
}

type PutRequest struct {
	Puts []PutObject `json:"puts"`
}

type PutResponseObject struct {
	UUID string `json:"uuid"`
}

type PutResponse struct {
	Responses []PutResponseObject `json:"responses"`
}

type MetricsEntry struct {
	Request  metrics.Meter
	Duration metrics.Timer
	Errors   metrics.Meter
}

func newMetricsEntry(name string, r metrics.Registry) *MetricsEntry {
	me := &MetricsEntry{
		Request:  metrics.GetOrRegisterMeter(fmt.Sprintf("%s.request_count", name), r),
		Duration: metrics.GetOrRegisterTimer(fmt.Sprintf("%s.request_duration", name), r),
		Errors:   metrics.GetOrRegisterMeter(fmt.Sprintf("%s.error_count", name), r),
	}

	return me
}

type Metrics struct {
	registry        metrics.Registry
	badRequestCount metrics.Meter
	putsLegacy      *MetricsEntry
	getsLegacy      *MetricsEntry
	putsCurrentURL  *MetricsEntry
	getsCurrentURL  *MetricsEntry
	putsBackend     *MetricsEntry
	getsBackend     *MetricsEntry
}

func createMetrics() *Metrics {

	flushTime := time.Second * 10
	r := metrics.NewPrefixedRegistry("prebidcache.")
	m := &Metrics{
		registry:        r,
		badRequestCount: metrics.GetOrRegisterMeter("bad_request_count", r),
		putsLegacy:      newMetricsEntry("puts.legacy_url", r),
		getsLegacy:      newMetricsEntry("gets.legacy_url", r),
		putsCurrentURL:  newMetricsEntry("puts.current_url", r),
		getsCurrentURL:  newMetricsEntry("gets.current_url", r),
		putsBackend:     newMetricsEntry("puts.backend", r),
		getsBackend:     newMetricsEntry("gets.backend", r),
	}

	metrics.RegisterDebugGCStats(m.registry)
	metrics.RegisterRuntimeMemStats(m.registry)

	go metrics.CaptureRuntimeMemStats(m.registry, flushTime)
	go metrics.CaptureDebugGCStats(m.registry, flushTime)

	return m
}

// AppHandlers stores the interfaces which our endpoint handlers depend on.
// This exists for dependency injection, to make the app testable
type AppHandlers struct {
	Backend Backend
	Metrics *Metrics

	putRequestPool    sync.Pool // Stores PutRequest instances
	putAnyRequestPool sync.Pool // Stores PutAnyRequest instances
	putResponsePool   sync.Pool // Stores PutResponse instances with MaxNumValues slots
}

// PutCacheHandler serves "POST /cache" requests.
func (deps *AppHandlers) PutCacheHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	//metricsCallback := deps.Metrics.StartPostCache()
	deps.Metrics.putsCurrentURL.Request.Mark(1)
	deps.Metrics.putsCurrentURL.Duration.Time(func() {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			deps.sendError(w, "Failed to read the request body.", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		put := deps.putAnyRequestPool.Get().(PutAnyRequest)
		defer deps.putAnyRequestPool.Put(put)

		err = json.Unmarshal(body, &put)
		if err != nil {
			deps.sendError(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
			return
		}

		if len(put.Puts) > MaxNumValues {
			deps.sendError(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
			return
		}

		resps := deps.putResponsePool.Get().(PutResponse)
		resps.Responses = make([]PutResponseObject, len(put.Puts))
		defer deps.putResponsePool.Put(resps)

		for i, p := range put.Puts {
			if len(p.Value) > MaxValueLength {
				deps.sendError(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
				return
			}

			if len(p.Value) == 0 {
				deps.sendError(w, "Missing value.", http.StatusBadRequest)
				return
			}

			var toCache string
			if p.Type == XML_PREFIX {
				if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
					deps.sendError(w, fmt.Sprintf("XML messages must have a String value. Found %v", p.Value), http.StatusBadRequest)
					return
				}

				// Be careful about the the cross-script escaping issues here. JSON requires quotation marks to be escaped,
				// for example... so we'll need to un-escape it before we consider it to be XML content.
				var interpreted string
				json.Unmarshal(p.Value, &interpreted)
				toCache = p.Type + interpreted
			} else if p.Type == JSON_PREFIX {
				toCache = p.Type + string(p.Value)
			} else {
				deps.sendError(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
				return
			}

			log.Debugf("Storing value: %s", toCache)
			resps.Responses[i].UUID = uuid.NewV4().String()
			err = deps.TimeBackendPut(resps.Responses[i].UUID, toCache)
			if err != nil {
				log.Error("POST /cache Error while writing to the backend:", err)
				switch err {
				case context.DeadlineExceeded:
					deps.sendError(w, "Timeout writing value to the backend", httpDependencyTimeout)
				default:
					deps.sendError(w, err.Error(), http.StatusInternalServerError)
				}
				deps.Metrics.putsCurrentURL.Errors.Mark(1)
				return
			}
		}

		bytes, err := json.Marshal(&resps)
		if err != nil {
			deps.sendError(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
			deps.Metrics.putsCurrentURL.Errors.Mark(1)
			return
		}

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	})
}

// GetCacheHandler serves "GET /cache?uuid={id}" endpoints.
func (deps *AppHandlers) GetCacheHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	deps.Metrics.getsCurrentURL.Request.Mark(1)
	deps.Metrics.getsCurrentURL.Duration.Time(func() {

		id := parseUUID(r)
		if id == "" {
			deps.sendError(w, "Missing required parameter uuid", http.StatusBadRequest)
			return
		}
		value, err := deps.TimeBackendGet(id)
		if err != nil {
			deps.sendError(w, "No content stored for uuid="+id, http.StatusNotFound)
			return
		}

		if strings.HasPrefix(value, XML_PREFIX) {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(value)[len(XML_PREFIX):])
		} else if strings.HasPrefix(value, JSON_PREFIX) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(value)[len(JSON_PREFIX):])
		} else {
			deps.Metrics.getsCurrentURL.Errors.Mark(1)
			deps.sendError(w, "Cache data was corrupted. Cannot determine type.", http.StatusInternalServerError)
		}
	})
}

// PutHandler is deprecated. It can be removed as soon as prebid-server is updated to stop using it.
func (deps *AppHandlers) PutHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	deps.Metrics.putsLegacy.Request.Mark(1)
	deps.Metrics.putsLegacy.Duration.Time(func() {
		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			deps.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		put := deps.putRequestPool.Get().(PutRequest)
		defer deps.putRequestPool.Put(put)

		err = json.Unmarshal(body, &put)
		if err != nil {
			deps.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(put.Puts) > MaxNumValues {
			deps.sendError(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
			return
		}

		resps := deps.putResponsePool.Get().(PutResponse)
		defer deps.putResponsePool.Put(resps)
		resps.Responses = make([]PutResponseObject, len(put.Puts))

		for i, p := range put.Puts {
			if len(p.Value) > MaxValueLength {
				deps.sendError(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
				return
			}

			if len(p.Value) == 0 {
				deps.sendError(w, "Missing value.", http.StatusBadRequest)
				return
			}

			log.Debugf("Value: %s", p.Value)
			resps.Responses[i].UUID = uuid.NewV4().String()
			err = deps.TimeBackendPut(resps.Responses[i].UUID, p.Value)
			if err != nil {
				log.Error("POST /put Error while writing to the backend:", err)
				switch err {
				case context.DeadlineExceeded:
					deps.sendError(w, "Timeout writing value to the backend", httpDependencyTimeout)
				default:
					deps.sendError(w, err.Error(), http.StatusInternalServerError)
				}
				deps.Metrics.putsLegacy.Errors.Mark(1)
				return
			}
		}

		bytes, err := json.Marshal(&resps)
		if err != nil {
			deps.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Write(bytes)
	})
}

// GetHandler is deprecated. It can be removed as soon as prebid-server is updated to stop using it.
func (deps *AppHandlers) GetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	// Automate all of this with customer middleware
	deps.Metrics.getsLegacy.Request.Mark(1)
	deps.Metrics.getsLegacy.Duration.Time(func() {

		/* Handles POST */
		w.Header().Set("Content-Type", "application/json")

		id := parseUUID(r)
		var value, err = deps.TimeBackendGet(id)

		if err != nil {
			w.Write([]byte("{ \"error\": \"not found\" }"))
		} else {
			fmt.Fprintf(w, "%s", value)
		}
	})
}

func (deps *AppHandlers) TimeBackendGet(uuid string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	deps.Metrics.getsBackend.Request.Mark(1)
	ts := time.Now()
	value, err := deps.Backend.Get(ctx, uuid)
	deps.Metrics.getsBackend.Duration.Update(time.Since(ts))

	if err != nil {
		deps.Metrics.getsBackend.Errors.Mark(1)
	}

	return value, err
}

func (deps *AppHandlers) TimeBackendPut(key string, value string) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	deps.Metrics.putsBackend.Request.Mark(1)
	deps.Metrics.putsBackend.Duration.Time(func() {
		err = deps.Backend.Put(ctx, key, value)
	})

	if err != nil {
		deps.Metrics.getsBackend.Errors.Mark(1)
	}

	return err
}

func (deps *AppHandlers) sendError(w http.ResponseWriter, err string, status int) {
	if status == http.StatusBadRequest {
		deps.Metrics.badRequestCount.Mark(1)
	}

	w.WriteHeader(status)
	w.Write([]byte(err))
}

func parseUUID(r *http.Request) string {
	return r.URL.Query().Get("uuid")
}

func initRateLimter(next http.Handler) http.Handler {
	viper.SetDefault("rate_limiter.enabled", true)
	viper.SetDefault("rate_limiter.num_requests", 100)

	// Sip rate limiter when disabled
	if viper.GetBool("rate_limiter.enabled") != true {
		return next
	}

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

	var appHandlers = AppHandlers{
		Backend: NewBackend(viper.GetString("backend.type")),
		Metrics: createMetrics(),

		putAnyRequestPool: sync.Pool{
			New: func() interface{} {
				return PutAnyRequest{}
			},
		},

		putRequestPool: sync.Pool{
			New: func() interface{} {
				return PutRequest{}
			},
		},

		putResponsePool: sync.Pool{
			New: func() interface{} {
				return PutResponse{}
			},
		},
	}

	go influxdb.InfluxDB(
		appHandlers.Metrics.registry,        // metrics registry
		time.Second*10,                      // interval
		viper.GetString("metrics.host"),     // the InfluxDB url
		viper.GetString("metrics.database"), // your InfluxDB database
		viper.GetString("metrics.username"), // your InfluxDB user
		viper.GetString("metrics.password"), // your InfluxDB password
	)

	router := httprouter.New()
	router.POST("/put", appHandlers.PutHandler)
	router.GET("/get", appHandlers.GetHandler)

	router.POST("/cache", appHandlers.PutCacheHandler)
	router.GET("/cache", appHandlers.GetCacheHandler)

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

	handler := &LoggingMiddleware{handler: corsRouter}
	limitHandler := initRateLimter(handler)

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

	<- stopSignals

	ctx, cancel := context.WithTimeout(context.Background(), 10 * time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Failed to shut down server: %v", err)
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		log.Errorf("Failed to shut down admin server: %v", err)
	}
}

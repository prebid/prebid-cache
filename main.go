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
	"github.com/rs/cors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"strings"

	"errors"
	"github.com/Prebid-org/prebid-cache/backends"
	backendDecorators "github.com/Prebid-org/prebid-cache/backends/decorators"
	endpointDecorators "github.com/Prebid-org/prebid-cache/endpoints/decorators"
	"github.com/Prebid-org/prebid-cache/metrics"
	"github.com/didip/tollbooth/limiter"
	"os/signal"
	"sync"
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

// AppHandlers stores the interfaces which our endpoint handlers depend on.
// This exists for dependency injection, to make the app testable
type AppHandlers struct {
	Backend backends.Backend

	putRequestPool    sync.Pool // Stores PutRequest instances
	putAnyRequestPool sync.Pool // Stores PutAnyRequest instances
	putResponsePool   sync.Pool // Stores PutResponse instances with MaxNumValues slots
}

// PutHandler serves "POST /cache" requests.
func (deps *AppHandlers) PutHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read the request body.", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	put := deps.putAnyRequestPool.Get().(PutAnyRequest)
	defer deps.putAnyRequestPool.Put(put)

	err = json.Unmarshal(body, &put)
	if err != nil {
		http.Error(w, "Request body "+string(body)+" is not valid JSON.", http.StatusBadRequest)
		return
	}

	if len(put.Puts) > MaxNumValues {
		http.Error(w, fmt.Sprintf("More keys than allowed: %d", MaxNumValues), http.StatusBadRequest)
		return
	}

	resps := deps.putResponsePool.Get().(PutResponse)
	resps.Responses = make([]PutResponseObject, len(put.Puts))
	defer deps.putResponsePool.Put(resps)

	for i, p := range put.Puts {
		if len(p.Value) > MaxValueLength {
			http.Error(w, fmt.Sprintf("Value is larger than allowed size: %d", MaxValueLength), http.StatusBadRequest)
			return
		}

		if len(p.Value) == 0 {
			http.Error(w, "Missing value.", http.StatusBadRequest)
			return
		}

		var toCache string
		if p.Type == XML_PREFIX {
			if p.Value[0] != byte('"') || p.Value[len(p.Value)-1] != byte('"') {
				http.Error(w, fmt.Sprintf("XML messages must have a String value. Found %v", p.Value), http.StatusBadRequest)
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
			http.Error(w, fmt.Sprintf("Type must be one of [\"json\", \"xml\"]. Found %v", p.Type), http.StatusBadRequest)
			return
		}

		log.Debugf("Storing value: %s", toCache)
		resps.Responses[i].UUID = uuid.NewV4().String()
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		err = deps.Backend.Put(ctx, resps.Responses[i].UUID, toCache)

		if err != nil {
			log.Error("POST /cache Error while writing to the backend:", err)
			switch err {
			case context.DeadlineExceeded:
				http.Error(w, "Timeout writing value to the backend", httpDependencyTimeout)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	bytes, err := json.Marshal(&resps)
	if err != nil {
		http.Error(w, "Failed to serialize UUIDs into JSON.", http.StatusInternalServerError)
		return
	}

	/* Handles POST */
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

// GetHandler serves "GET /cache?uuid={id}" endpoints.
func (deps *AppHandlers) GetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := parseUUID(r)
	if err != nil {
		if id == "" {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	value, err := deps.Backend.Get(ctx, id)

	if err != nil {
		http.Error(w, "No content stored for uuid="+id, http.StatusNotFound)
		return
	}

	if strings.HasPrefix(value, XML_PREFIX) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(value)[len(XML_PREFIX):])
	} else if strings.HasPrefix(value, JSON_PREFIX) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value)[len(JSON_PREFIX):])
	} else {
		http.Error(w, "Cache data was corrupted. Cannot determine type.", http.StatusInternalServerError)
	}
}

func status(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// We might want more logic here eventually... but for now, we're ok to serve more traffic as
	// long as the server responds.
	w.WriteHeader(http.StatusNoContent)
}

func parseUUID(r *http.Request) (string, error) {
	id := r.URL.Query().Get("uuid")
	var err error = nil
	if id == "" {
		err = errors.New("Missing required parameter uuid")
	} else if len(id) != 36 {
		// UUIDs are 36 characters long... so this quick check lets us filter out most invalid
		// ones before even checking the backend.
		err = fmt.Errorf("No content stored for uuid=%s", id)
	}
	return id, err
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

	appMetrics := metrics.CreateMetrics()
	var appHandlers = AppHandlers{
		Backend: backendDecorators.LogMetrics(backends.NewBackend(viper.GetString("backend.type")), appMetrics),

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

	router := httprouter.New()
	router.GET("/status", status) // Determines whether the server is ready for more traffic.

	router.POST("/cache", endpointDecorators.MonitorHttp(appHandlers.PutHandler, appMetrics.Puts))
	router.GET("/cache", endpointDecorators.MonitorHttp(appHandlers.GetHandler, appMetrics.Gets))
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

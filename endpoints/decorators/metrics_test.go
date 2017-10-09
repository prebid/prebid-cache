package decorators

import (
	"github.com/julienschmidt/httprouter"
	pbcmetrics "github.com/Prebid-org/prebid-cache/metrics"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/Prebid-org/prebid-cache/metrics/metricstest"
)

func TestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	entry := doRequest(handler)

	metricstest.AssertSuccessMetricsExist(t, entry)
}

func TestBadRequestMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(400)
	}
	entry := doRequest(handler)

	if entry.Request.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
	if entry.Duration.Count() != 0 {
		t.Errorf("The request duration should not have been counted.")
	}
	if entry.BadRequest.Count() != 1 {
		t.Errorf("A Bad request should have been counted.")
	}
	if entry.Errors.Count() != 0 {
		t.Errorf("No Errors should have been counted.")
	}
}

func TestErrorMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(500)
	}
	entry := doRequest(handler)
	metricstest.AssertErrorMetricsExist(t, entry)
}

func TestNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	entry := doRequest(handler)
	metricstest.AssertSuccessMetricsExist(t, entry)
}

func TestWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	entry := doRequest(handler)
	metricstest.AssertSuccessMetricsExist(t, entry)
}

func doRequest(handler func(http.ResponseWriter, *http.Request, httprouter.Params)) *pbcmetrics.MetricsEntry {
	reg := metrics.NewRegistry()
	entry := pbcmetrics.NewMetricsEntry("foo", reg)
	monitoredHandler := MonitorHttp(handler, entry)
	monitoredHandler(httptest.NewRecorder(), nil, nil)
	return entry
}

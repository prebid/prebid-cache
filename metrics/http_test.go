package metrics

import (
	"github.com/julienschmidt/httprouter"
	"github.com/rcrowley/go-metrics"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	entry := doRequest(handler)

	assertSuccessMetricsExist(t, entry)
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
	assertErrorMetricsExist(t, entry)
}

func TestNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	entry := doRequest(handler)
	assertSuccessMetricsExist(t, entry)
}

func TestWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	entry := doRequest(handler)
	assertSuccessMetricsExist(t, entry)
}

func doRequest(handler func(http.ResponseWriter, *http.Request, httprouter.Params)) *MetricsEntry {
	reg := metrics.NewRegistry()
	entry := newMetricsEntry("foo", reg)
	monitoredHandler := MonitorHttp(handler, entry)
	monitoredHandler(httptest.NewRecorder(), nil, nil)
	return entry
}

func assertSuccessMetricsExist(t *testing.T, entry *MetricsEntry) {
	t.Helper()
	if entry.Request.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
	if entry.Duration.Count() != 1 {
		t.Errorf("The request duration should have been counted.")
	}
	if entry.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if entry.Errors.Count() != 0 {
		t.Errorf("No Errors should have been counted.")
	}
}

func assertErrorMetricsExist(t *testing.T, entry *MetricsEntry) {
	t.Helper()
	if entry.Request.Count() != 1 {
		t.Errorf("The request should have been counted.")
	}
	if entry.Duration.Count() != 0 {
		t.Errorf("The request duration should not have been counted.")
	}
	if entry.BadRequest.Count() != 0 {
		t.Errorf("No Bad requests should have been counted.")
	}
	if entry.Errors.Count() != 1 {
		t.Errorf("An Error should have been counted.")
	}
}

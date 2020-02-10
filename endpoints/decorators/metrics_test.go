package decorators

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRequestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	doRequest(handler, "gets")

	assert.Equalf(t, int64(1), metricstest.HT2["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutRequestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	doRequest(handler, "puts")

	assert.Equalf(t, int64(1), metricstest.HT2["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestBadGetRequestMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(400)
	}
	doRequest(handler, "gets")

	assert.Equalf(t, int64(1), metricstest.HT2["gets.current_url.request.total"], "Unsuccessful get request has not been accounted for in the total request count")
	assert.Equalf(t, int64(1), metricstest.HT2["gets.current_url.request.bad_request"], "Unsuccessful get request has not been accounted for in the bad request count")
	assert.Equal(t, metricstest.HT1["gets.current_url.duration"], 0.00, "Unsuccessful get request duration should have been logged")
}

func TestBadPutRequestMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(400)
	}
	doRequest(handler, "puts")

	assert.Equalf(t, int64(1), metricstest.HT2["puts.current_url.request.total"], "Unsuccessful put request has not been accounted for in the total request count")
	assert.Equalf(t, int64(1), metricstest.HT2["puts.current_url.request.bad_request"], "Unsuccessful put request has not been accounted for in the bad request count")
	assert.Equal(t, metricstest.HT1["puts.current_url.duration"], 0.00, "Unsuccessful put request duration should have been logged")
}

func TestGetRequestErrorMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(500)
	}
	doRequest(handler, "gets")

	assert.Equal(t, int64(1), metricstest.HT2["gets.current_url.request.error"], "Failed get request should have been accounted under the error label")
	assert.Equal(t, int64(1), metricstest.HT2["gets.current_url.request.total"], "Failed get request should have been accounted in the request totals")
}

func TestPutRequestErrorMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(500)
	}
	doRequest(handler, "puts")

	assert.Equal(t, int64(1), metricstest.HT2["puts.current_url.request.error"], "Failed put request should have been accounted under the error label")
	assert.Equal(t, int64(1), metricstest.HT2["puts.current_url.request.total"], "Failed put request should have been accounted in the request totals")
}

func TestGetReqNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	doRequest(handler, "gets")

	assert.Equalf(t, int64(1), metricstest.HT2["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutReqNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	doRequest(handler, "puts")

	assert.Equalf(t, int64(1), metricstest.HT2["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestGetReqWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	doRequest(handler, "gets")

	assert.Equalf(t, int64(1), metricstest.HT2["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutReqWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	doRequest(handler, "puts")

	assert.Equalf(t, int64(1), metricstest.HT2["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.HT1["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func doRequest(handler func(http.ResponseWriter, *http.Request, httprouter.Params), method string) {
	m := metricstest.CreateMockMetrics()
	monitoredHandler := MonitorHttp(handler, m, method)
	monitoredHandler(httptest.NewRecorder(), nil, nil)
}

package decorators

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/stretchr/testify/assert"
)

func TestGetRequestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	doRequest(handler, GetMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutRequestSuccessMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	}
	doRequest(handler, PostMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestBadGetRequestMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(400)
	}
	doRequest(handler, GetMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["gets.current_url.request.total"], "Unsuccessful get request has not been accounted for in the total request count")
	assert.Equalf(t, int64(1), metricstest.MockCounters["gets.current_url.request.bad_request"], "Unsuccessful get request has not been accounted for in the bad request count")
	assert.Equal(t, metricstest.MockHistograms["gets.current_url.duration"], 0.00, "Unsuccessful get request duration should have been logged")
}

func TestBadPutRequestMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(400)
	}
	doRequest(handler, PostMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["puts.current_url.request.total"], "Unsuccessful put request has not been accounted for in the total request count")
	assert.Equalf(t, int64(1), metricstest.MockCounters["puts.current_url.request.bad_request"], "Unsuccessful put request has not been accounted for in the bad request count")
	assert.Equal(t, metricstest.MockHistograms["puts.current_url.duration"], 0.00, "Unsuccessful put request duration should have been logged")
}

func TestCustomKeyPutRequestMetrics(t *testing.T) {
	metrics := metricstest.CreateMockMetrics()

	type testExpectedValues struct {
		totalRequests     int64
		badRequests       int64
		customKeyRequests int64
	}
	testCases := []struct {
		desc                  string
		inHandler             func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params)
		expectedCounterValues testExpectedValues
	}{
		{
			desc: "A put request that comes with its own custom key and Put endpoint throws no error",
			inHandler: func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
				w.WriteHeader(CacheUpdateCode)
				w.WriteHeader(200)
			},
			expectedCounterValues: testExpectedValues{
				totalRequests:     int64(1),
				badRequests:       int64(0),
				customKeyRequests: int64(1),
			},
		},
		{
			desc: "A put request that comes with its own custom key and Put endpoint throws error",
			inHandler: func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
				w.WriteHeader(CacheUpdateCode)
				w.WriteHeader(400)
			},
			expectedCounterValues: testExpectedValues{
				totalRequests:     int64(2),
				badRequests:       int64(1),
				customKeyRequests: int64(2),
			},
		},
	}

	for _, tc := range testCases {
		// Run test
		monitoredHandler := MonitorHttp(tc.inHandler, metrics, PostMethod)
		monitoredHandler(httptest.NewRecorder(), nil, nil)

		// Assert
		assert.Equalf(t, tc.expectedCounterValues.totalRequests, metricstest.MockCounters["puts.current_url.request.total"], "Put request has not been accounted for in the 'total request' count")
		assert.Equalf(t, tc.expectedCounterValues.badRequests, metricstest.MockCounters["puts.current_url.request.bad_request"], "Put request has not been accounted for in the 'bad request' count")
		assert.Equalf(t, tc.expectedCounterValues.customKeyRequests, metricstest.MockCounters["puts.current_url.request.custom_key"], "Put request has not been accounted for in the 'put requests with custom key' count")
	}

}

func TestGetRequestErrorMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(500)
	}
	doRequest(handler, GetMethod)

	assert.Equal(t, int64(1), metricstest.MockCounters["gets.current_url.request.error"], "Failed get request should have been accounted under the error label")
	assert.Equal(t, int64(1), metricstest.MockCounters["gets.current_url.request.total"], "Failed get request should have been accounted in the request totals")
}

func TestPutRequestErrorMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.WriteHeader(500)
	}
	doRequest(handler, PostMethod)

	assert.Equal(t, int64(1), metricstest.MockCounters["puts.current_url.request.error"], "Failed put request should have been accounted under the error label")
	assert.Equal(t, int64(1), metricstest.MockCounters["puts.current_url.request.total"], "Failed put request should have been accounted in the request totals")
}

func TestGetReqNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	doRequest(handler, GetMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutReqNoExplicitHeaderMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {}
	doRequest(handler, PostMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func TestGetReqWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	doRequest(handler, GetMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["gets.current_url.request.total"], "Successful get request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["gets.current_url.duration"], 0.00, "Successful get request duration should be greater than zero")
}

func TestPutReqWriteBytesMetrics(t *testing.T) {
	var handler = func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Write([]byte("Success"))
	}
	doRequest(handler, PostMethod)

	assert.Equalf(t, int64(1), metricstest.MockCounters["puts.current_url.request.total"], "Successful put request has not been accounted for in the total request count")
	assert.Greater(t, metricstest.MockHistograms["puts.current_url.duration"], 0.00, "Successful put request duration should be greater than zero")
}

func doRequest(handler func(http.ResponseWriter, *http.Request, httprouter.Params), method int) {
	m := metricstest.CreateMockMetrics()
	monitoredHandler := MonitorHttp(handler, m, method)
	monitoredHandler(httptest.NewRecorder(), nil, nil)
}

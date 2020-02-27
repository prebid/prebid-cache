package decorators

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/metrics"
	"net/http"
	"time"
)

type writerWithStatus struct {
	delegate   http.ResponseWriter
	statusCode int
}

func (w *writerWithStatus) WriteHeader(statusCode int) {
	// Capture only the first call, because that's the one the client got.
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.delegate.WriteHeader(statusCode)
}

func (w *writerWithStatus) Write(bytes []byte) (int, error) {
	return w.delegate.Write(bytes)
}

func (w *writerWithStatus) Header() http.Header {
	return w.delegate.Header()
}

func MonitorHttp(handler httprouter.Handle, me *metrics.Metrics, method string) httprouter.Handle {
	return httprouter.Handle(func(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
		metricsLogSuccess(me, method)
		wrapper := writerWithStatus{
			delegate: resp,
		}

		start := time.Now()
		handler(&wrapper, req, params)
		respCode := wrapper.statusCode
		// If the calling function never calls WriterHeader explicitly, Go auto-fills it with a 200
		if respCode == 0 || respCode >= 200 && respCode < 300 {
			metricsLogDuration(me, method, &start)
		} else if respCode >= 400 && respCode < 500 {
			metricsLogBadRequest(me, method)
		} else {
			metricsLogError(me, method)
		}
	})
}

func metricsLogDuration(me *metrics.Metrics, method string, duration *time.Time) {
	if method == "POST" {
		me.RecordPutDuration(duration)
	} else if method == "GET" {
		me.RecordGetDuration(duration)
	}
}
func metricsLogSuccess(me *metrics.Metrics, method string) {
	if method == "POST" {
		me.RecordPutTotal()
	} else if method == "GET" {
		me.RecordGetTotal()
	}
}
func metricsLogBadRequest(me *metrics.Metrics, method string) {
	if method == "POST" {
		me.RecordPutBadRequest()
	} else if method == "GET" {
		me.RecordGetBadRequest()
	}
}
func metricsLogError(me *metrics.Metrics, method string) {
	if method == "POST" {
		me.RecordPutError()
	} else if method == "GET" {
		me.RecordGetError()
	}
}

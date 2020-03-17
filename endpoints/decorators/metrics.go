package decorators

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/metrics"
	"net/http"
	"time"
)

const (
	PostMethod = 1
	GetMethod  = 2
)

type metricsFunctions struct {
	LogSuccess    func()
	LogDuration   func(duration *time.Time)
	LogBadRequest func()
	LogError      func()
}

func assignMetricsFunctions(m *metrics.Metrics, method int) *metricsFunctions {
	metrics := &metricsFunctions{}
	switch method {
	case PostMethod:
		metrics.LogSuccess = m.RecordPutTotal
		metrics.LogDuration = m.RecordPutDuration
		metrics.LogBadRequest = m.RecordPutBadRequest
		metrics.LogError = m.RecordPutError
	case GetMethod:
		metrics.LogSuccess = m.RecordGetTotal
		metrics.LogDuration = m.RecordGetDuration
		metrics.LogBadRequest = m.RecordGetBadRequest
		metrics.LogError = m.RecordGetError
	}
	return metrics
}

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

func MonitorHttp(handler httprouter.Handle, m *metrics.Metrics, method int) httprouter.Handle {
	return httprouter.Handle(func(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
		mf := assignMetricsFunctions(m, method)
		mf.LogSuccess()
		wrapper := writerWithStatus{
			delegate: resp,
		}

		start := time.Now()
		handler(&wrapper, req, params)
		respCode := wrapper.statusCode
		// If the calling function never calls WriterHeader explicitly, Go auto-fills it with a 200
		if respCode == 0 || respCode >= 200 && respCode < 300 {
			mf.LogDuration(&start)
		} else if respCode >= 400 && respCode < 500 {
			mf.LogBadRequest()
		} else {
			mf.LogError()
		}
	})
}

package decorators

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-cache/metrics"
)

const (
	PostMethod = 1
	GetMethod  = 2
)

const CustomUuidKey string = "IsKeyProvided"

type metricsFunctions struct {
	RecordTotal       func()
	RecordDuration    func(duration time.Duration)
	RecordBadRequest  func()
	RecordError       func()
	RecordKeyProvided func()
}

func assignMetricsFunctions(m *metrics.Metrics, method int) *metricsFunctions {
	metrics := &metricsFunctions{}
	switch method {
	case PostMethod:
		metrics.RecordTotal = m.RecordPutTotal
		metrics.RecordDuration = m.RecordPutDuration
		metrics.RecordBadRequest = m.RecordPutBadRequest
		metrics.RecordError = m.RecordPutError
		metrics.RecordKeyProvided = m.RecordPutKeyProvided
	case GetMethod:
		metrics.RecordTotal = m.RecordGetTotal
		metrics.RecordDuration = m.RecordGetDuration
		metrics.RecordBadRequest = m.RecordGetBadRequest
		metrics.RecordError = m.RecordGetError
		metrics.RecordKeyProvided = func() {} // Doesn't get called from Get endpoint
	}
	return metrics
}

// writerWithStatus implements the http.ResponseWriter interface in order to store
// extra information needed for metrics purposes.
type writerWithStatus struct {
	delegate       http.ResponseWriter
	statusCode     int
	wasKeyProvided bool
}

// WriteHeader is writerWithStatus's implementation of the http.ResponseWriter interface
// WriteHeader(statusCode int) method and stores the backend client's statusCode
// into the writerWithStatus's statusCode field so we can log metrics
// accoding to its value in the decorator implemented in MonitorHttp()
func (w *writerWithStatus) WriteHeader(statusCode int) {
	// Capture only the first call, because that's the one the client got.
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.delegate.WriteHeader(statusCode)
}

// Write is writerWithStatus's implementation of the http.ResponseWriter interface
// Write(bytes []byte) (int, error) method and includes logic to set the wasKeyProvided
// field to true if the "IsKeyProvided" keyword is passed as parameter. If not, bytes
// is assumed to be the Put response and is passed as parameter to http.ResponseWriter's
// Write(bytes []byte) (int, error) instead.
func (w *writerWithStatus) Write(bytes []byte) (int, error) {
	if string(bytes) == CustomUuidKey {
		w.wasKeyProvided = true
		return 0, nil
	}

	return w.delegate.Write(bytes)
}

func (w *writerWithStatus) Header() http.Header {
	return w.delegate.Header()
}

func MonitorHttp(handler httprouter.Handle, m *metrics.Metrics, method int) httprouter.Handle {
	return httprouter.Handle(func(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
		mf := assignMetricsFunctions(m, method)
		mf.RecordTotal()
		wrapper := writerWithStatus{
			delegate: resp,
		}

		start := time.Now()
		handler(&wrapper, req, params)

		if wrapper.wasKeyProvided {
			mf.RecordKeyProvided()
		}

		respCode := wrapper.statusCode
		// If the calling function never calls WriterHeader explicitly, Go auto-fills it with a 200
		if respCode == 0 || respCode >= 200 && respCode < 300 {
			mf.RecordDuration(time.Since(start))
		} else if respCode >= 400 && respCode < 500 {
			mf.RecordBadRequest()
		} else {
			mf.RecordError()
		}
	})
}

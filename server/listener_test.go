package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func assertMetrics(t *testing.T, expectedMetrics metricsRecorded, actualMetrics metricstest.MockMetrics) {
	t.Helper()

	//if expectedMetrics.RecordPutTotal > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutTotal")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutTotal")
	//}
	//if expectedMetrics.RecordPutKeyProvided > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutKeyProvided")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutKeyProvided")
	//}
	//if expectedMetrics.RecordPutBadRequest > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBadRequest")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBadRequest")
	//}
	//if expectedMetrics.RecordPutError > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutError")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutError")
	//}
	//if expectedMetrics.RecordPutDuration > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutDuration")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutDuration")
	//}
	//if expectedMetrics.RecordPutBackendXml > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendXml")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendXml")
	//}
	//if expectedMetrics.RecordPutBackendJson > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendJson")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendJson")
	//}
	//if expectedMetrics.RecordPutBackendError > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendError")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendError")
	//}
	//if expectedMetrics.RecordPutBackendInvalid > 0 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendInvalid")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendInvalid")
	//}
	//if expectedMetrics.RecordPutBackendSize > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendSize")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendSize")
	//}
	//if expectedMetrics.RecordPutBackendTTLSeconds > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendTTLSeconds")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendTTLSeconds")
	//}
	//if expectedMetrics.RecordPutBackendDuration > 0.00 {
	//	actualMetrics.AssertCalled(t, "RecordPutBackendDuration")
	//} else {
	//	actualMetrics.AssertNotCalled(t, "RecordPutBackendDuration")
	//}
	// ---
	if expectedMetrics.RecordAcceptConnectionErrors > 0 {
		actualMetrics.AssertCalled(t, "RecordAcceptConnectionErrors")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordAcceptConnectionErrors")
	}
	if expectedMetrics.RecordCloseConnectionErrors > 0 {
		actualMetrics.AssertCalled(t, "RecordCloseConnectionErrors")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordCloseConnectionErrors")
	}
	if expectedMetrics.RecordConnectionClosed > 0 {
		actualMetrics.AssertCalled(t, "RecordConnectionClosed")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordConnectionClosed")
	}
	if expectedMetrics.RecordConnectionOpen > 0 {
		actualMetrics.AssertCalled(t, "RecordConnectionOpen")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordConnectionOpen")
	}
	if expectedMetrics.RecordGetBackendDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordGetBackendDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendDuration")
	}
	if expectedMetrics.RecordGetBackendError > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBackendError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendError")
	}
	if expectedMetrics.RecordGetBackendTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBackendTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBackendTotal")
	}
	if expectedMetrics.RecordGetBadRequest > 0 {
		actualMetrics.AssertCalled(t, "RecordGetBadRequest")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetBadRequest")
	}
	if expectedMetrics.RecordGetDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordGetDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetDuration")
	}
	if expectedMetrics.RecordGetError > 0 {
		actualMetrics.AssertCalled(t, "RecordGetError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetError")
	}
	if expectedMetrics.RecordGetTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordGetTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordGetTotal")
	}
	if expectedMetrics.RecordKeyNotFoundError > 0 {
		actualMetrics.AssertCalled(t, "RecordKeyNotFoundError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordKeyNotFoundError")
	}
	if expectedMetrics.RecordMissingKeyError > 0 {
		actualMetrics.AssertCalled(t, "RecordMissingKeyError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordMissingKeyError")
	}
	if expectedMetrics.RecordPutBackendDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendDuration")
	}
	if expectedMetrics.RecordPutBackendError > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendError")
	}
	if expectedMetrics.RecordPutBackendInvalid > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendInvalid")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendInvalid")
	}
	if expectedMetrics.RecordPutBackendJson > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendJson")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendJson")
	}
	if expectedMetrics.RecordPutBackendSize > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendSize")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendSize")
	}
	if expectedMetrics.RecordPutBackendTTLSeconds > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutBackendTTLSeconds")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendTTLSeconds")
	}
	if expectedMetrics.RecordPutBackendXml > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBackendXml")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBackendXml")
	}
	if expectedMetrics.RecordPutBadRequest > 0 {
		actualMetrics.AssertCalled(t, "RecordPutBadRequest")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutBadRequest")
	}
	if expectedMetrics.RecordPutDuration > 0.00 {
		actualMetrics.AssertCalled(t, "RecordPutDuration")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutDuration")
	}
	if expectedMetrics.RecordPutError > 0 {
		actualMetrics.AssertCalled(t, "RecordPutError")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutError")
	}
	if expectedMetrics.RecordPutKeyProvided > 0 {
		actualMetrics.AssertCalled(t, "RecordPutKeyProvided")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutKeyProvided")
	}
	if expectedMetrics.RecordPutTotal > 0 {
		actualMetrics.AssertCalled(t, "RecordPutTotal")
	} else {
		actualMetrics.AssertNotCalled(t, "RecordPutTotal")
	}
}

func createMockMetrics() metricstest.MockMetrics {
	mockMetrics := metricstest.MockMetrics{}
	mockMetrics.On("RecordAcceptConnectionErrors")
	mockMetrics.On("RecordCloseConnectionErrors")
	mockMetrics.On("RecordConnectionClosed")
	mockMetrics.On("RecordConnectionOpen")
	mockMetrics.On("RecordGetBackendDuration", mock.Anything)
	mockMetrics.On("RecordGetBackendError")
	mockMetrics.On("RecordGetBackendTotal")
	mockMetrics.On("RecordGetBadRequest")
	mockMetrics.On("RecordGetDuration", mock.Anything)
	mockMetrics.On("RecordGetError")
	mockMetrics.On("RecordGetTotal")
	mockMetrics.On("RecordKeyNotFoundError")
	mockMetrics.On("RecordMissingKeyError")
	mockMetrics.On("RecordPutBackendDuration", mock.Anything)
	mockMetrics.On("RecordPutBackendError")
	mockMetrics.On("RecordPutBackendInvalid")
	mockMetrics.On("RecordPutBackendJson")
	mockMetrics.On("RecordPutBackendSize", mock.Anything)
	mockMetrics.On("RecordPutBackendTTLSeconds", mock.Anything)
	mockMetrics.On("RecordPutBackendXml")
	mockMetrics.On("RecordPutBadRequest")
	mockMetrics.On("RecordPutDuration", mock.Anything)
	mockMetrics.On("RecordPutError")
	mockMetrics.On("RecordPutKeyProvided")
	mockMetrics.On("RecordPutTotal")
	return mockMetrics
}

type metricsRecorded struct {
	// Connection metrics
	RecordAcceptConnectionErrors int64 `json:"acceptConnectionErrors"`
	RecordCloseConnectionErrors  int64 `json:"closeConnectionErrors"`
	RecordConnectionClosed       int64 `json:"connectionClosed"`
	RecordConnectionOpen         int64 `json:"connectionOpen"`

	// Get metrics
	RecordGetBackendDuration float64 `json:"recordGetBackendDuration"`
	RecordGetBackendError    int64   `json:"recordGetBackendError"`
	RecordGetBackendTotal    int64   `json:"recordGetBackendTotal"`
	RecordGetBadRequest      int64   `json:"recordGetBadrequest"`
	RecordGetDuration        float64 `json:"recordGetDuration"`
	RecordGetError           int64   `json:"recordGetError"`
	RecordGetTotal           int64   `json:"recordGetTotal"`

	// Put metrics
	RecordKeyNotFoundError     int64   `json:"keyNotFoundError"`
	RecordMissingKeyError      int64   `json:"missingKeyError"`
	RecordPutBackendDuration   float64 `json:"putBackendDuration"`
	RecordPutBackendError      int64   `json:"putBackendError"`
	RecordPutBackendInvalid    int64   `json:"putBackendInvalid"`
	RecordPutBackendJson       int64   `json:"totalJsonRequests"`
	RecordPutBackendSize       float64 `json:"putBackendSize"`
	RecordPutBackendTTLSeconds float64 `json:"putBackendTTLSeconds"`
	RecordPutBackendXml        int64   `json:"totalXmlRequests"`
	RecordPutBadRequest        int64   `json:"putBadRequest"`
	RecordPutDuration          float64 `json:"putDuration"`
	RecordPutError             int64   `json:"putError"`
	RecordPutKeyProvided       int64   `json:"putKeyProvided"`
	RecordPutTotal             int64   `json:"putTotal"`
}

func TestConnections(t *testing.T) {
	testCases := []struct {
		desc                    string
		allowAccept, allowClose bool
		expectedConnectionError error
		expectedMetrics         metricsRecorded
	}{
		{
			desc:                    "net.Listener will fail when attempting to open a connection. Expect error and RecordAcceptConnectionErrors to be called",
			allowAccept:             false,
			allowClose:              false,
			expectedConnectionError: errors.New("Failed to open connection"),
			expectedMetrics: metricsRecorded{
				RecordAcceptConnectionErrors: 1,
			},
		},
		{
			desc:                    "net.Listener will fail when attempting to open a connection. Expect error and RecordAcceptConnectionErrors to be called",
			allowAccept:             false,
			allowClose:              true,
			expectedConnectionError: errors.New("Failed to open connection"),
			expectedMetrics: metricsRecorded{
				RecordAcceptConnectionErrors: 1,
			},
		},
		{
			desc:                    "net.Listener will open and close connections successfully. Expect ConnectionOpen and ConnectionClosed metrics to be logged",
			allowAccept:             true,
			allowClose:              true,
			expectedConnectionError: nil,
			expectedMetrics: metricsRecorded{
				RecordConnectionOpen:   1,
				RecordConnectionClosed: 1,
			},
		},
		{
			desc:                    "net.Listener will open a connection but will fail when trying to close it. Expect ConnectionOpen and a CloseConnectionErrors to be accounted for in the metrics",
			allowAccept:             true,
			allowClose:              false,
			expectedConnectionError: errors.New("Failed to close connection."),
			expectedMetrics: metricsRecorded{
				RecordCloseConnectionErrors: 1,
				RecordConnectionOpen:        1,
			},
		},
	}

	for _, tc := range testCases {
		mockMetrics := createMockMetrics()
		m := &metrics.Metrics{
			MetricEngines: []metrics.CacheMetrics{
				&mockMetrics,
			},
		}

		var listener net.Listener = &mockListener{
			listenSuccess: tc.allowAccept,
			closeSuccess:  tc.allowClose,
		}

		listener = &monitorableListener{listener, m}
		conn, err := listener.Accept()
		if tc.allowAccept {
			err = conn.Close()
		}
		assert.Equal(t, tc.expectedConnectionError, err, tc.desc)
		assertMetrics(t, tc.expectedMetrics, mockMetrics)
	}
}

func assertCount(t *testing.T, context string, actual int64, expected int) {
	t.Helper()
	if actual != int64(expected) {
		t.Errorf("%s: expected %d, got %d", context, expected, actual)
	}
}

type mockListener struct {
	listenSuccess bool
	closeSuccess  bool
}

func (l *mockListener) Accept() (net.Conn, error) {
	if l.listenSuccess {
		return &mockConnection{l.closeSuccess}, nil
	} else {
		return nil, errors.New("Failed to open connection")
	}
}

func (l *mockListener) Close() error {
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return &mockAddr{}
}

type mockConnection struct {
	closeSuccess bool
}

func (c *mockConnection) Read(b []byte) (n int, err error) {
	return len(b), nil
}

func (c *mockConnection) Write(b []byte) (n int, err error) {
	return
}

func (c *mockConnection) Close() error {
	if c.closeSuccess {
		return nil
	} else {
		return errors.New("Failed to close connection.")
	}
}

func (c *mockConnection) LocalAddr() net.Addr {
	return &mockAddr{}
}

func (c *mockConnection) RemoteAddr() net.Addr {
	return &mockAddr{}
}

func (c *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

type mockAddr struct{}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return "192.0.2.1:25"
}

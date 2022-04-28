package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/metrics"
	"github.com/prebid/prebid-cache/metrics/metricstest"
	"github.com/stretchr/testify/assert"
)

func TestConnections(t *testing.T) {
	testCases := []struct {
		desc                    string
		allowAccept, allowClose bool
		expectedConnectionError error
		expectedMetrics         metricstest.MetricsRecorded
	}{
		{
			desc:                    "net.Listener will fail when attempting to open a connection. Expect error and RecordAcceptConnectionErrors to be called",
			allowAccept:             false,
			allowClose:              false,
			expectedConnectionError: errors.New("Failed to open connection"),
			expectedMetrics: metricstest.MetricsRecorded{
				RecordAcceptConnectionErrors: 1,
			},
		},
		{
			desc:                    "net.Listener will fail when attempting to open a connection. Expect error and RecordAcceptConnectionErrors to be called",
			allowAccept:             false,
			allowClose:              true,
			expectedConnectionError: errors.New("Failed to open connection"),
			expectedMetrics: metricstest.MetricsRecorded{
				RecordAcceptConnectionErrors: 1,
			},
		},
		{
			desc:                    "net.Listener will open and close connections successfully. Expect ConnectionOpen and ConnectionClosed metrics to be logged",
			allowAccept:             true,
			allowClose:              true,
			expectedConnectionError: nil,
			expectedMetrics: metricstest.MetricsRecorded{
				RecordConnectionOpen:   1,
				RecordConnectionClosed: 1,
			},
		},
		{
			desc:                    "net.Listener will open a connection but will fail when trying to close it. Expect ConnectionOpen and a CloseConnectionErrors to be accounted for in the metrics",
			allowAccept:             true,
			allowClose:              false,
			expectedConnectionError: errors.New("Failed to close connection."),
			expectedMetrics: metricstest.MetricsRecorded{
				RecordCloseConnectionErrors: 1,
				RecordConnectionOpen:        1,
			},
		},
	}

	for _, tc := range testCases {
		mockMetrics := metricstest.CreateMockMetrics()
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
		metricstest.AssertMetrics(t, tc.expectedMetrics, mockMetrics)
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

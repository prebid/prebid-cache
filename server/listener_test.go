package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prebid/prebid-cache/metrics"
	"github.com/stretchr/testify/assert"
)

func TestNormalConnectionMetrics(t *testing.T) {
	doTest(t, true, true)
}

func TestAcceptErrorMetrics(t *testing.T) {
	doTest(t, false, false)
}

func TestCloseErrorMetrics(t *testing.T) {
	doTest(t, true, false)
}

func doTest(t *testing.T, allowAccept bool, allowClose bool) {
	m := metrics.CreateMockMetrics()

	var listener net.Listener = &mockListener{
		listenSuccess: allowAccept,
		closeSuccess:  allowClose,
	}

	listener = &monitorableListener{listener, m}
	conn, err := listener.Accept()
	if !allowAccept {
		if err == nil {
			t.Error("The listener.Accept() error should propagate from the underlying listener.")
		}
		assert.Equal(t, metrics.HT1["connections.connections_opened"], 0.00, "Should not log any connections")
		assert.Equal(t, int64(1), metrics.HT2["connections.connection_error.accept"], "Metrics engine should not log an accept connection error")
		assert.Equal(t, int64(0), metrics.HT2["connections.connection_error.close"], "Metrics engine should have logged a close connection error")
		return
	}
	assert.Equal(t, int64(0), metrics.HT2["connections.connection_error.accept"], "Metrics engine should not log an accept connection error")
	assert.Equal(t, metrics.HT1["connections.connections_opened"], 1.00, "Should not log any connections")

	err = conn.Close()
	if allowClose {
		assert.Equal(t, metrics.HT1["connections.connections_opened"], 0.00, "Should not log any connections")
		assert.Equal(t, int64(0), metrics.HT2["connections.connection_error.accept"], "Metrics engine should not log an accept connection error")
		assert.Equal(t, int64(0), metrics.HT2["connections.connection_error.close"], "Metrics engine should have logged a close connection error")
	} else {
		if err == nil {
			t.Error("The connection.Close() error should propagate from the underlying listener.")
		}
		assert.Equal(t, metrics.HT1["connections.connections_opened"], 1.00, "Should not log any connections")
		assert.Equal(t, int64(0), metrics.HT2["connections.connection_error.accept"], "Metrics engine should not log an accept connection error")
		assert.Equal(t, int64(1), metrics.HT2["connections.connection_error.close"], "Metrics engine should have logged a close connection error")
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

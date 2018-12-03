package server

import (
	"errors"
	"net"
	"testing"
	"time"

	pbcmetrics "github.com/PubMatic-OpenWrap/prebid-cache/metrics"
	metrics "github.com/rcrowley/go-metrics"
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
	connMetrics := pbcmetrics.NewConnectionMetrics(metrics.NewRegistry())

	var listener net.Listener = &mockListener{
		listenSuccess: allowAccept,
		closeSuccess:  allowClose,
	}

	listener = &monitorableListener{listener, connMetrics}
	conn, err := listener.Accept()
	if !allowAccept {
		if err == nil {
			t.Error("The listener.Accept() error should propagate from the underlying listener.")
		}
		assertCount(t, "When Accept() fails, connection count", connMetrics.ActiveConnections.Count(), 0)
		assertCount(t, "When Accept() fails, Accept() errors", connMetrics.ConnectionAcceptErrors.Count(), 1)
		assertCount(t, "When Accept() fails, Close() errors", connMetrics.ConnectionCloseErrors.Count(), 0)
		return
	}
	assertCount(t, "When Accept() succeeds, active connections", connMetrics.ActiveConnections.Count(), 1)
	assertCount(t, "When Accept() succeeds, Accept() errors", connMetrics.ConnectionAcceptErrors.Count(), 0)

	err = conn.Close()
	if allowClose {
		assertCount(t, "When Accept() and Close() succeed, connection count", connMetrics.ActiveConnections.Count(), 0)
		assertCount(t, "When Accept() and Close() succeed, Accept() errors", connMetrics.ConnectionAcceptErrors.Count(), 0)
		assertCount(t, "When Accept() and Close() succeed, Close() errors", connMetrics.ConnectionCloseErrors.Count(), 0)
	} else {
		if err == nil {
			t.Error("The connection.Close() error should propagate from the underlying listener.")
		}
		assertCount(t, "When Accept() succeeds sand Close() fails, connection count", connMetrics.ActiveConnections.Count(), 1)
		assertCount(t, "When Accept() succeeds sand Close() fails, Accept() errors", connMetrics.ConnectionAcceptErrors.Count(), 0)
		assertCount(t, "When Accept() succeeds sand Close() fails, Close() errors", connMetrics.ConnectionCloseErrors.Count(), 1)
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

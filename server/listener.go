package server

import (
	"net"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/metrics"
)

// monitorableListener tracks any opened connections in the metrics.
type monitorableListener struct {
	net.Listener
	metrics *metrics.Metrics
}

// monitorableConnection tracks any closed connections in the metrics.
type monitorableConnection struct {
	net.Conn
	metrics *metrics.Metrics
}

func (l *monitorableConnection) Close() error {
	err := l.Conn.Close()
	if err == nil {
		l.metrics.RecordConnectionClosed()
	} else {
		logger.Error("Error closing connection: %v", err)
		l.metrics.RecordCloseConnectionErrors()
	}
	return err
}

func (ln *monitorableListener) Accept() (c net.Conn, err error) {
	tc, err := ln.Listener.Accept()
	if err != nil {
		logger.Error("Error accepting connection: %v", err)
		ln.metrics.RecordAcceptConnectionErrors()
		return tc, err
	}
	ln.metrics.RecordConnectionOpen()
	return &monitorableConnection{
		tc,
		ln.metrics,
	}, nil
}

// tcpKeepAliveListener is copy/pasted from the implementation here: https://golang.org/pkg/net/http/#Server.ListenAndServe
// Since it's not public, the best we can do is copy/paste it here.
//
// We should revisit this after Go 1.11. See also:
// - https://github.com/golang/go/issues/23378
// - https://github.com/golang/go/issues/23459
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"git.pubmatic.com/PubMatic/go-common.git/logger"

	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
)

// Listen serves requests and blocks forever, until OS signals shut down the process.
func Listen(cfg config.Configuration, handler http.Handler, metrics *metrics.ConnectionMetrics) {
	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	stopAdmin := make(chan os.Signal)
	stopMain := make(chan os.Signal)
	done := make(chan struct{})

	// Rig up each server so that it listens on a channel for signals. These use different channels for each server
	// because a shared channel would only alert one consumer (whichever one happens to read it first).
	//
	// After a server has finished shutting down, it should send a signal in through the "done" channel.
	mainServer := newMainServer(cfg, handler)
	adminServer := newAdminServer(cfg)
	go shutdownAfterSignals(mainServer, stopMain, done)
	go shutdownAfterSignals(adminServer, stopAdmin, done)

	// Attach the servers to the sockets
	mainListener, err := newListener(mainServer.Addr, metrics)
	if err != nil {
		logger.Error("Error listening for TCP connections on %s: %v", mainServer.Addr, err)
		return
	}
	adminListener, err := newListener(adminServer.Addr, nil)
	if err != nil {
		logger.Error("Error listening for TCP connections on %s: %v", adminServer.Addr, err)
		return
	}
	go runServer(mainServer, "Main", mainListener)
	go runServer(adminServer, "Admin", adminListener)

	// Then block the thread. When the OS sends a shutdown signal, alert each of the servers.
	// Once they're finished shutting down (the "done" channel gets pinged for each server),
	// this funciton can return.
	wait(stopSignals, done, stopMain, stopAdmin)
	return
}

func newAdminServer(cfg config.Configuration) *http.Server {
	return &http.Server{
		Addr: ":" + strconv.Itoa(cfg.AdminPort),
	}
}

func newMainServer(cfg config.Configuration, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
}

func runServer(server *http.Server, name string, listener net.Listener) {
	logger.Info("%s server starting on: %s", name, server.Addr)
	err := server.Serve(listener)
	logger.Error("%s server quit with error: %v", name, err)
}

func newListener(address string, metrics *metrics.ConnectionMetrics) (net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("Error listening for TCP connections on %s: %v", address, err)
	}

	// This cast is in Go's core libs as Server.ListenAndServe(), so it _should_ be safe, but just in case it changes in a future version...
	if casted, ok := ln.(*net.TCPListener); ok {
		ln = &tcpKeepAliveListener{casted}
	} else {
		logger.Warn("net.Listen(\"tcp\", \"addr\") didn't return a TCPListener as it did in Go 1.9. Things will probably work fine... but this should be investigated.")
	}

	if metrics != nil {
		ln = &monitorableListener{ln, metrics}
	}

	return ln, nil
}

func wait(inbound <-chan os.Signal, done <-chan struct{}, outbound ...chan<- os.Signal) {
	sig := <-inbound

	for i := 0; i < len(outbound); i++ {
		go sendSignal(outbound[i], sig)
	}

	for i := 0; i < len(outbound); i++ {
		<-done
	}
}

func shutdownAfterSignals(server *http.Server, stopper <-chan os.Signal, done chan<- struct{}) {
	sig := <-stopper

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var s struct{}
	logger.Info("Stopping %s because of signal: %s", server.Addr, sig.String())
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown %s: %v", server.Addr, err)
	}
	done <- s
}

func sendSignal(to chan<- os.Signal, sig os.Signal) {
	to <- sig
}

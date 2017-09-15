package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// TODO
// 0. CORES
// 1. Rate limiting
// 2. Authorization
// 3. Backlisting?

type LoggingMiddleware struct {
	handler http.Handler
}

func (m *LoggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Debug("Request: ", r.URL.Path)
	m.handler.ServeHTTP(w, r)
}

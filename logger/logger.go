package logger

import (
	"fmt"

	"github.com/golang/glog"
)

func InfoWithRequestID(requestID string, format string, v ...interface{}) {
	log := "RqID: " + requestID + " " + fmt.Sprintf(format, v...)
	glog.InfoDepth(1, log)
}

func DebugWithRequestID(requestID string, format string, v ...interface{}) {
	log := "RqID: " + requestID + " " + fmt.Sprintf(format, v...)
	glog.InfoDepth(1, log)
}

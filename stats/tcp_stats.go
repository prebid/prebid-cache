package stats

import (
	"fmt"
	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"git.pubmatic.com/PubMatic/go-common.git/tcpstats"
	"github.com/PubMatic-OpenWrap/prebid-cache/constant"
)

type statsTCP struct {
	statsClient *tcpstats.Client
}

type statLogger struct{}

func (l statLogger) Error(format string, args ...interface{}) {
	logger.Error(format, args...)
}

func (l statLogger) Info(format string, args ...interface{}) {
	logger.Info(format, args...)
}

func initTCPStatsClient(statIP, statPort, server, dc string,
	pubInterval, pubThreshold, retries, dialTimeout, keepAliveDur, maxIdleConn, maxIdleConnPerHost int) (iStats, error) {

	cgf := tcpstats.Config{
		Host:                statIP,
		Port:                statPort,
		Server:              server,
		DC:                  dc,
		PublishingInterval:  pubInterval,
		PublishingThreshold: pubThreshold,
		Retries:             retries,
		DialTimeout:         dialTimeout,
		KeepAliveDuration:   keepAliveDur,
		MaxIdleConns:        maxIdleConn,
		MaxIdleConnsPerHost: maxIdleConnPerHost,
	}

	sc, err := tcpstats.NewClient(cgf, statLogger{})
	if err != nil {
		logger.Error("Failed to connect to stats server via TCP")
		return nil, err
	}

	return statsTCP{statsClient: sc}, nil
}

func (st statsTCP) LogCacheFailedGetStats(errorString string) {
	st.statsClient.PublishStat(fmt.Sprintf(constant.StatsKeyCacheFailedGet, errorString), 1)
}

func (st statsTCP) LogCacheMissStats() {
	st.statsClient.PublishStat(constant.StatsKeyCacheMiss, 1)
}

func (st statsTCP) LogCacheFailedPutStats(errorString string) {
	st.statsClient.PublishStat(fmt.Sprintf(constant.StatsKeyCacheFailedPut, errorString), 1)
}

func (st statsTCP) LogCacheRequestedGetStats() {
	st.statsClient.PublishStat(constant.StatsKeyCacheRequestedGet, 1)
}

func (st statsTCP) LogCacheRequestedPutStats() {
	st.statsClient.PublishStat(constant.StatsKeyCacheRequestedPut, 1)
}

func (st statsTCP) LogAerospikeErrorStats() {
	st.statsClient.PublishStat(constant.StatsKeyAerospikeCreationError, 1)
}

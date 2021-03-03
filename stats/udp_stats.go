package stats

import (
	"errors"
	"fmt"
	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"git.pubmatic.com/PubMatic/go-common.git/stats"
	"github.com/PubMatic-OpenWrap/prebid-cache/constant"
)

type statsUDP struct {
	statsClient *stats.S
}

func initUDPStatsClient(statIP, statPort, statServer, dc string) (iStats, error) {
	statURL := statIP + ":" + statPort
	sc := stats.NewStats(statURL, statServer, dc)
	if sc == nil {
		logger.Error("Failed to connect to stats server via UDP")
		return nil, errors.New("failed to connect to stats server via UDP")
	}

	return statsUDP{statsClient: sc}, nil
}

func (su statsUDP) LogCacheFailedGetStats(errorString string) {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyCacheFailedGet, errorString), constant.StatsKeyCacheFailedGetCutoff, 1)
}

func (su statsUDP) LogCacheMissStats() {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyCacheMiss), constant.StatsKeyCacheMissCutOff, 1)
}

func (su statsUDP) LogCacheFailedPutStats(errorString string) {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyCacheFailedPut, errorString), constant.StatsKeyCacheFailedPutCutoff, 1)
}

func (su statsUDP) LogCacheRequestedGetStats() {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyCacheRequestedGet), constant.StatsKeyCacheRequestedGetCutoff, 1)
}

func (su statsUDP) LogCacheRequestedPutStats() {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyCacheRequestedPut), constant.StatsKeyCacheRequestedPutCutoff, 1)
}

func (su statsUDP) LogAerospikeErrorStats() {
	_ = su.statsClient.Increment(fmt.Sprintf(constant.StatsKeyAerospikeCreationError), constant.StatsKeyAerospikeCreationErrorCutoff, 1)
}

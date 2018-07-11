package stats

import (
	"fmt"

	"github.com/prebid/prebid-cache/constant"

	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"git.pubmatic.com/PubMatic/go-common.git/stats"
)

var S *stats.S

func InitStat(statIP, statPort, statServer, dc string) {
	statURL := statIP + ":" + statPort
	S = stats.NewStats(statURL, statServer, dc)
	if S == nil {
		logger.Error("Falied to Connect Stat Server ")
	}
}

func LogCacheFailedGetStats() {
	S.Increment(fmt.Sprintf(constant.StatsKeyCacheFailedGet),
		constant.StatsKeyCacheFailedGetCutoff, 1)
}

func LogCacheFailedPutStats() {
	S.Increment(fmt.Sprintf(constant.StatsKeyCacheFailedPut),
		constant.StatsKeyCacheFailedPutCutoff, 1)
}

func LogCacheRequestedGetStats() {
	S.Increment(fmt.Sprintf(constant.StatsKeyCacheRequestedGet),
		constant.StatsKeyCacheRequestedGetCutoff, 1)
}

func LogCacheRequestedPutStats() {
	S.Increment(fmt.Sprintf(constant.StatsKeyCacheRequestedPut),
		constant.StatsKeyCacheRequestedPutCutoff, 1)
}

func LogAerospikeErrorStats() {
	S.Increment(fmt.Sprintf(constant.StatsKeyAerospikeCreationError),
		constant.StatsKeyAerospikeCreationErrorCutoff, 1)
}

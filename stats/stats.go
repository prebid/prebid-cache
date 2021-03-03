package stats

var (
	sc iStats
)

func InitStat(host, udpPort, server, dc string,

	tcpPort string,
	pubInterval int,
	pubThreshold int,
	retries int,
	dialTimeout int,
	keepAliveDuration int,
	maxIdleCons int,
	maxIdleConsPerHost int,

	useTCP bool) {

	var err error
	if useTCP {
		sc, err = initTCPStatsClient(host, tcpPort, server, dc, pubInterval, pubThreshold, retries, dialTimeout, keepAliveDuration, maxIdleCons, maxIdleConsPerHost)
	} else {
		sc, err = initUDPStatsClient(host, udpPort, server, dc)
	}

	if err != nil {
		sc = noStats{}
	}
}

type iStats interface {
	LogCacheFailedGetStats(errorString string)
	LogCacheMissStats()
	LogCacheFailedPutStats(errorString string)
	LogCacheRequestedGetStats()
	LogCacheRequestedPutStats()
	LogAerospikeErrorStats()
}

func LogCacheFailedGetStats(errorString string) {
	sc.LogCacheFailedGetStats(errorString)
}

func LogCacheMissStats() {
	sc.LogCacheMissStats()
}

func LogCacheFailedPutStats(errorString string) {
	sc.LogCacheFailedPutStats(errorString)
}

func LogCacheRequestedGetStats() {
	sc.LogCacheRequestedGetStats()
}

func LogCacheRequestedPutStats() {
	sc.LogCacheRequestedPutStats()
}

func LogAerospikeErrorStats() {
	sc.LogAerospikeErrorStats()
}

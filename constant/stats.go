package constant

const (
	// SERVER LEVEL STATS KEYS

	//StatsKeyCacheFailedGet : stats Key for Failed Cache get operations
	StatsKeyCacheFailedGet       = "hb:cachegetfail"
	StatsKeyCacheFailedGetCutoff = 1

	//StatsKeyCacheFailedPut : stats Key for Failed Cache put operations
	StatsKeyCacheFailedPut       = "hb:cacheputfail"
	StatsKeyCacheFailedPutCutoff = 1

	//StatsKeyCacheRequestedGet : stats Key for Requested Cache get operations
	StatsKeyCacheRequestedGet       = "hb:cachereqget"
	StatsKeyCacheRequestedGetCutoff = 10

	//StatsKeyCacheRequestedPut : stats Key for Requested Cache put operations
	StatsKeyCacheRequestedPut       = "hb:cachereqput"
	StatsKeyCacheRequestedPutCutoff = 10

	//StatsKeyAerospikeCreationError : stats Key for Aerospike creation error
	StatsKeyAerospikeCreationError       = "hb:cacheaerospikeerr"
	StatsKeyAerospikeCreationErrorCutoff = 1
)

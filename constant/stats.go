package constant

const (
	// SERVER LEVEL STATS KEYS
	/*
		001 - Failed Cache get operations
		002 - Failed Cache misses
		003 - Failed Cache put operations
		004 - Requested Cache get operations
		005 - Requested Cache put operations
		006 - Aerospike creation error
	*/

	//StatsKeyCacheFailedGet : stats Key for Failed Cache get operations
	StatsKeyCacheFailedGet       = "hb:001:%s"
	StatsKeyCacheFailedGetCutoff = 10

	//StatsKeyCacheMiss : stats Key for Failed Cache misses
	StatsKeyCacheMiss       = "hb:002"
	StatsKeyCacheMissCutOff = 10

	//StatsKeyCacheFailedPut : stats Key for Failed Cache put operations
	StatsKeyCacheFailedPut       = "hb:003:%s"
	StatsKeyCacheFailedPutCutoff = 10

	//StatsKeyCacheRequestedGet : stats Key for Requested Cache get operations
	StatsKeyCacheRequestedGet       = "hb:004"
	StatsKeyCacheRequestedGetCutoff = 10

	//StatsKeyCacheRequestedPut : stats Key for Requested Cache put operations
	StatsKeyCacheRequestedPut       = "hb:005"
	StatsKeyCacheRequestedPutCutoff = 10

	//StatsKeyAerospikeCreationError : stats Key for Aerospike creation error
	StatsKeyAerospikeCreationError       = "hb:006"
	StatsKeyAerospikeCreationErrorCutoff = 10
)

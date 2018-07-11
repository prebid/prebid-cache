package stats

import (
	"testing"
)

func TestStatsInit(t *testing.T) {
	InitStat("127.0.0.1", "8888", "TestHost", "TestDC")
}

func TestStatsLogCacheFailedGetStats(t *testing.T) {
	LogCacheFailedGetStats()
}

func TestStatsLogCacheFailedPutStats(t *testing.T) {
	LogCacheFailedPutStats()
}

func TestStatsLogCacheRequestedGetStats(t *testing.T) {
	LogCacheRequestedGetStats()
}

func TestStatsLogCacheRequestedPutStats(t *testing.T) {
	LogCacheRequestedPutStats()
}

func TestStatsLogAerospikeErrorStats(t *testing.T) {
	LogAerospikeErrorStats()
}

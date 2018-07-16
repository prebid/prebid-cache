package stats

import (
	"testing"
)

func TestStatsInit(t *testing.T) {
	InitStat("127.0.0.1", "8888", "TestHost", "TestDC")
}

func TestStatsLogCacheFailedGetStats(t *testing.T) {
	LogCacheFailedGetStats("Error string")
}

func TestStatsLogCacheFailedPutStats(t *testing.T) {
	LogCacheFailedPutStats("Error string")
}

func TestStatsLogCacheRequestedGetStats(t *testing.T) {
	LogCacheRequestedGetStats()
}

func TestStatsLogCacheMissStats(t *testing.T) {
	LogCacheMissStats()
}

func TestStatsLogCacheRequestedPutStats(t *testing.T) {
	LogCacheRequestedPutStats()
}

func TestStatsLogAerospikeErrorStats(t *testing.T) {
	LogAerospikeErrorStats()
}

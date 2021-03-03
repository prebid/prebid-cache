package stats

type noStats struct{}

func (ns noStats) LogCacheFailedGetStats(string) {}
func (ns noStats) LogCacheMissStats() {}
func (ns noStats) LogCacheFailedPutStats(string) {}
func (ns noStats) LogCacheRequestedGetStats() {}
func (ns noStats) LogCacheRequestedPutStats() {}
func (ns noStats) LogAerospikeErrorStats() {}

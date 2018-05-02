package config

import (
	"log"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
	"github.com/spf13/viper"
)

func NewBackend(cfg config.Configuration, appMetrics *metrics.Metrics) backends.Backend {
	backend := newBaseBackend(cfg.Backend)
	if cfg.RequestLimits.MaxSize > 0 {
		backend = decorators.EnforceSizeLimit(backend, cfg.RequestLimits.MaxSize)
	}
	backend = decorators.LogMetrics(backend, appMetrics)
	if viper.GetString("compression.type") == "snappy" {
		backend = compression.SnappyCompress(backend)
	}
	return backend
}

func newBaseBackend(cfg config.Backend) backends.Backend {
	switch cfg.Type {
	case config.BackendCassandra:
		return backends.NewCassandraBackend(cfg.Cassandra)
	case config.BackendMemory:
		return backends.NewMemoryBackend()
	case config.BackendMemcache:
		return backends.NewMemcacheBackend(cfg.Memcache)
	case config.BackendAzure:
		return backends.NewAzureBackend(cfg.Azure.Account, cfg.Azure.Key)
	case config.BackendAerospike:
		return backends.NewAerospikeBackend(cfg.Aerospike)
	default:
		log.Fatalf("Unknown backend type: %s", cfg.Type)
	}

	panic("Error creating backend. This shouldn't happen.")
}

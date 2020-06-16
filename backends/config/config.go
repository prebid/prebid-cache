package config

import (
	log "github.com/sirupsen/logrus"

	"github.com/prebid/prebid-cache/backends"
	"github.com/prebid/prebid-cache/backends/decorators"
	"github.com/prebid/prebid-cache/compression"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/metrics"
)

func NewBackend(cfg config.Configuration, appMetrics *metrics.Metrics) backends.Backend {
	backend := newBaseBackend(cfg.Backend, appMetrics)
	backend = decorators.LimitTTLs(backend, cfg.RequestLimits.MaxTTLSeconds)
	if cfg.RequestLimits.MaxSize > 0 {
		backend = decorators.EnforceSizeLimit(backend, cfg.RequestLimits.MaxSize)
	}
	// Metrics must be taken _before_ compression because it relies on the
	// "json" or "xml" prefix on the payload. Compression might munge this.
	// We should re-work this strategy at some point.
	backend = applyCompression(cfg.Compression, backend)
	backend = decorators.LogMetrics(backend, appMetrics)
	return backend
}

func applyCompression(cfg config.Compression, backend backends.Backend) backends.Backend {
	switch cfg.Type {
	case config.CompressionNone:
		return backend
	case config.CompressionSnappy:
		return compression.SnappyCompress(backend)
	default:
		log.Fatalf("Unknown compression type: %s", cfg.Type)
	}

	panic("Error applying compression. This shouldn't happen.")
}

func newBaseBackend(cfg config.Backend, appMetrics *metrics.Metrics) backends.Backend {
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
		return backends.NewAerospikeBackend(cfg.Aerospike, appMetrics)
	case config.BackendRedis:
		return backends.NewRedisBackend(cfg.Redis)
	default:
		log.Fatalf("Unknown backend type: %s", cfg.Type)
	}

	panic("Error creating backend. This shouldn't happen.")
}

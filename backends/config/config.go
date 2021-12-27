package config

import (
	"git.pubmatic.com/PubMatic/go-common.git/logger"
	"github.com/PubMatic-OpenWrap/prebid-cache/backends"
	"github.com/PubMatic-OpenWrap/prebid-cache/backends/decorators"
	"github.com/PubMatic-OpenWrap/prebid-cache/compression"
	"github.com/PubMatic-OpenWrap/prebid-cache/config"
	"github.com/PubMatic-OpenWrap/prebid-cache/metrics"
)

func NewBackend(cfg config.Configuration, appMetrics *metrics.Metrics) backends.Backend {
	backend := newBaseBackend(cfg.Backend, appMetrics)
	backend = applyCompression(cfg.Compression, backend)
	if cfg.RequestLimits.MaxSize > 0 {
		backend = decorators.EnforceSizeLimit(backend, cfg.RequestLimits.MaxSize)
	}
	// Metrics must be taken _before_ compression because it relies on the
	// "json" or "xml" prefix on the payload. Compression might munge this.
	// We should re-work this strategy at some point.
	backend = decorators.LogMetrics(backend, appMetrics)
	backend = decorators.LimitTTLs(backend, getMaxTTLSeconds(cfg))
	return backend
}

func applyCompression(cfg config.Compression, backend backends.Backend) backends.Backend {
	switch cfg.Type {
	case config.CompressionNone:
		return backend
	case config.CompressionSnappy:
		return compression.SnappyCompress(backend)
	default:
		logger.Fatal("Unknown compression type: %s", cfg.Type)
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
	case config.BackendAerospike:
		return backends.NewAerospikeBackend(cfg.Aerospike, appMetrics)
	case config.BackendRedis:
		return backends.NewRedisBackend(cfg.Redis)
	default:
		logger.Fatal("Unknown backend type: %s", cfg.Type)
	}

	panic("Error creating backend. This shouldn't happen.")
}

// getMaxTTLSeconds was added for backards compatibility. This function will select either
// config.backend.aerospike.default_ttl_seconds or backend.redis.expiration over
// config.request_limits.max_ttl_seconds if they are not zero and hold a smaller TTL value
// than config.request_limits.max_ttl_seconds does. In other words: smaller, backend-level-defined,
// non-zero TTL values take precedence.
//
// Notice that both config.backend.aerospike.default_ttl_seconds and backend.redis.expiration
// are getting deprecated in favor of config.request_limits.max_ttl_seconds
func getMaxTTLSeconds(cfg config.Configuration) int {
	maxTTLSeconds := cfg.RequestLimits.MaxTTLSeconds

	switch cfg.Backend.Type {
	case config.BackendCassandra:
		// If config.request_limits.max_ttl_seconds was defined to be less than 2400 seconds, go
		// with 2400 as it has been the TTL limit hardcoded in the Cassandra backend so far.
		if maxTTLSeconds > 2400 {
			maxTTLSeconds = 2400
		}
	case config.BackendAerospike:
		// If both config.request_limits.max_ttl_seconds and config.backend.aerospike.default_ttl_seconds
		// were defined, the smallest value takes preference
		if cfg.Backend.Aerospike.DefaultTTL > 0 && maxTTLSeconds > cfg.Backend.Aerospike.DefaultTTL {
			maxTTLSeconds = cfg.Backend.Aerospike.DefaultTTL
		}
	case config.BackendRedis:
		// If both config.request_limits.max_ttl_seconds and backend.redis.expiration
		// were defined, the smallest value takes preference
		if cfg.Backend.Redis.Expiration > 0 && maxTTLSeconds > cfg.Backend.Redis.Expiration*60 {
			maxTTLSeconds = cfg.Backend.Redis.Expiration * 60
		}
	}
	return maxTTLSeconds
}

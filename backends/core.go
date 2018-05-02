package backends

import (
	"context"

	log "github.com/Sirupsen/logrus"
	"github.com/prebid/prebid-cache/config"
)

// Backend interface for storing data
type Backend interface {
	Put(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
}

func NewBackend(cfg config.Backend) Backend {
	switch cfg.Type {
	case config.BackendCassandra:
		return NewCassandraBackend(cfg.Cassandra)
	case config.BackendMemory:
		return NewMemoryBackend()
	case config.BackendMemcache:
		return NewMemcacheBackend(cfg.Memcache)
	case config.BackendAzure:
		return NewAzureBackend(cfg.Azure.Account, cfg.Azure.Key)
	case config.BackendAerospike:
		return NewAerospikeBackend(cfg.Aerospike)
	default:
		log.Fatalf("Unknown backend type: %s", cfg.Type)
	}

	panic("Error creating backend. This shouldn't happen.")
}

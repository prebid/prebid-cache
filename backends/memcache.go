package backends

import (
	"context"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
)

type MemcacheDataStore interface {
	Get(key string) (*memcache.Item, error)
	Put(key string, value string, ttlSeconds int) error
}

// Memcache Object use to implement MemcacheDataStore interface
type Memcache struct {
	client *memcache.Client
}

func (mc *Memcache) Get(key string) (*memcache.Item, error) {
	return mc.client.Get(key)
}

func (mc *Memcache) Put(key string, value string, ttlSeconds int) error {
	return mc.client.Set(&memcache.Item{
		Expiration: int32(ttlSeconds),
		Key:        key,
		Value:      []byte(value),
	})
}

//------------------------------------------------------------------------------

// MemcacheBackend implements the Backend interface
type MemcacheBackend struct {
	client MemcacheDataStore
}

// NewMemcacheBackend create a new memcache backend
func NewMemcacheBackend(cfg config.Memcache) *MemcacheBackend {
	return &MemcacheBackend{
		client: &Memcache{
			memcache.New(cfg.Hosts...),
		},
	}
}

func (mc *MemcacheBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := mc.client.Get(key)

	if err != nil {
		if err == memcache.ErrCacheMiss {
			err = utils.KeyNotFoundError{}
		}
		return "", err
	}

	return string(res.Value), nil
}

// Put calls Set(item *Item), that writes the given item, unconditionally as
// opposed to Add, that writes the given item only if no value already exists or
// Replace, that writes only if the server already holds data for this key
func (mc *MemcacheBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	return mc.client.Put(key, value, ttlSeconds)
}

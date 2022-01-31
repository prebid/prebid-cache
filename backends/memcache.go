package backends

import (
	"context"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prebid/prebid-cache/config"
	"github.com/prebid/prebid-cache/utils"
	log "github.com/sirupsen/logrus"
)

// MemcacheDataStore is an interface that helps us communicate with an instance of the
// memcached cache server. Its implementation is intended to use the
// "github.com/bradfitz/gomemcache/memcache" client
type MemcacheDataStore interface {
	Get(key string) (*memcache.Item, error)
	Put(key string, value string, ttlSeconds int) error
}

// Memcache Object use to implement MemcacheDataStore interface
type Memcache struct {
	client *memcache.Client
}

// Get uses the github.com/bradfitz/gomemcache/memcache library to retrieve
// the value stored under 'key', if any
func (mc *Memcache) Get(key string) (*memcache.Item, error) {
	return mc.client.Get(key)
}

// Put uses the github.com/bradfitz/gomemcache/memcache library to store
// 'value' under 'key'. Because Prebid Cache doesn't implement 'upsert',
// Put calls Add(item *Item), that writes the given item only if no value
// already exists for its key as opposed to Set(item *Item), or Replace(item *Item)
func (mc *Memcache) Put(key string, value string, ttlSeconds int) error {
	return mc.client.Add(&memcache.Item{
		Expiration: int32(ttlSeconds),
		Key:        key,
		Value:      []byte(value),
	})
}

// MemcacheBackend implements the Backend interface
type MemcacheBackend struct {
	memcache MemcacheDataStore
}

// NewMemcacheBackend creates a new memcache backend and expects a valid
// 'cfg config.Memcache' argument
func NewMemcacheBackend(cfg config.Memcache) *MemcacheBackend {
	var mc *memcache.Client
	if cfg.ConfigHost != "" {
		var err error
		mc, err = memcache.NewDiscoveryClient(cfg.ConfigHost, time.Duration(cfg.PollIntervalSeconds)*time.Second)
		if err != nil {
			log.Fatalf("%v", err)
			panic("Memcache failure. This shouldn't happen.")
		}
	} else {
		mc = memcache.New(cfg.Hosts...)
	}

	return &MemcacheBackend{
		memcache: &Memcache{mc},
	}
}

// Get makes the MemcacheDataStore client to retrieve the value that has been previously
// stored under 'key'. If unseuccessful, returns an empty value and a KeyNotFoundError
// or other, memcache-related error
func (mc *MemcacheBackend) Get(ctx context.Context, key string) (string, error) {
	res, err := mc.memcache.Get(key)

	if err != nil {
		if err == memcache.ErrCacheMiss {
			err = utils.NewPBCError(utils.KEY_NOT_FOUND)
		}
		return "", err
	}

	return string(res.Value), nil
}

// Put makes the MemcacheDataStore client to store `value` only if `key` doesn't exist
// in the storage already. If it does, no operation is performed and Put returns RecordExistsError
func (mc *MemcacheBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	err := mc.memcache.Put(key, value, ttlSeconds)
	if err != nil && err == memcache.ErrNotStored {
		return utils.NewPBCError(utils.RECORD_EXISTS)
	}
	return err
}

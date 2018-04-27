package backends

import (
	"context"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/prebid/prebid-cache/config"
)

// MemcacheConfig is used to configure the cluster
type MemcacheConfig struct {
	hosts []string
}

// Memcache Object use to implement backend interface
type Memcache struct {
	client *memcache.Client
}

// NewMemcacheBackend create a new memcache backend
func NewMemcacheBackend(cfg config.Memcache) *Memcache {
	c := &Memcache{}
	mc := memcache.New(cfg.Hosts...)
	c.client = mc
	return c
}

func (mc *Memcache) Get(ctx context.Context, key string) (string, error) {
	res, err := mc.client.Get(key)

	if err != nil {
		return "", err
	}

	return string(res.Value), nil
}

func (mc *Memcache) Put(ctx context.Context, key string, value string) error {
	err := mc.client.Set(&memcache.Item{Key: key, Value: []byte(value)})

	if err != nil {
		return err
	}

	return nil
}

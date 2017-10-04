package backends

import (
	"context"
	"github.com/bradfitz/gomemcache/memcache"
)

// MemcacheConfig is used to configure the cluster
type MemcacheConfig struct {
	hosts string
}

// Memcache Object use to implement backend interface
type Memcache struct {
	client *memcache.Client
}

// NewMemcacheBackend create a new memcache backend
func NewMemcacheBackend(config *MemcacheConfig) (*Memcache, error) {
	c := &Memcache{}
	mc := memcache.New(config.hosts)
	c.client = mc

	return c, nil
}

func (mc *Memcache) Get(ctx context.Context, key string) (string, error) {
	res, err := mc.client.Get(key)

	if err != nil {
		return "", err
	}

	return string(res.Value), nil
}

func (mc *Memcache) Put(ctx context.Context, key string, value string) error {
	mc.client.Set(&memcache.Item{Key: key, Value: []byte(value)})
	return nil
}

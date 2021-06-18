package backends

import (
	"context"
	"sync"

	"github.com/prebid/prebid-cache/utils"
)

type MemoryBackend struct {
	db map[string]string
	mu sync.Mutex
}

func (b *MemoryBackend) Get(ctx context.Context, key string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	v, ok := b.db[key]
	if !ok {
		return "", utils.KeyNotFoundError{}
	}

	return v, nil
}

func (b *MemoryBackend) Put(ctx context.Context, key string, value string, ttlSeconds int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.db[key] = value
	return nil
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		db: make(map[string]string),
	}
}

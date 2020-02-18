package backends

import (
	"context"
	"fmt"
	"sync"
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
		return "", fmt.Errorf("Not found")
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

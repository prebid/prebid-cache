package backends

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
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
	log.Infof("metrics TTL seconds logged: %d", ttlSeconds)
	return nil
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		db: make(map[string]string),
	}
}

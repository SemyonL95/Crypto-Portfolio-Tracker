package cache

import (
	"context"
	"sync"
)

type Cache[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewCache[K comparable, V any](size int) *Cache[K, V] {
	return &Cache[K, V]{
		m: make(map[K]V, size),
	}
}

func (c *Cache[K, V]) Get(_ context.Context, k K) (V, bool) {
	c.mu.RLock()
	v, ok := c.m[k]
	c.mu.RUnlock()
	return v, ok
}

func (c *Cache[K, V]) Set(_ context.Context, k K, v V) {
	c.mu.Lock()
	c.m[k] = v
	c.mu.Unlock()
}

func (c *Cache[K, V]) GetBatch(_ context.Context, keys []K) map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make(map[K]V, len(keys))

	for _, k := range keys {
		if v, ok := c.m[k]; ok {
			res[k] = v
		}
	}

	return res
}

func (c *Cache[K, V]) SetBatch(_ context.Context, items map[K]V) {
	c.mu.Lock()
	for k, v := range items {
		c.m[k] = v
	}
	c.mu.Unlock()
}

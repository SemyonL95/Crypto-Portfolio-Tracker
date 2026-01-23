package price

import (
	"context"
	"fmt"
	"sync"
	"testtask/internal/domain/price"
	"time"
)

type cacheItem struct {
	price     *price.Price
	ExpiresAt time.Time
}
type Cache struct {
	mu    *sync.RWMutex
	items map[string]*cacheItem
	ttl   uint8
}

// NewCache creates a new price cache
func NewCache(ttlSeconds uint8) *Cache {
	return &Cache{
		mu:    &sync.RWMutex{},
		items: make(map[string]*cacheItem),
		ttl:   ttlSeconds,
	}
}

func (c *Cache) GetPrice(_ context.Context, key string, currency string) (*price.Price, bool) {
	c.mu.RLock()
	v, ok := c.items[c.cacheKey(key, currency)]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if v.ExpiresAt.After(time.Now()) {
		delete(c.items, key)

		return nil, false
	}

	return v.price, true
}

func (c *Cache) SetPrice(_ context.Context, key string, price *price.Price, currency string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.items[c.cacheKey(key, currency)] = &cacheItem{
		price:     price,
		ExpiresAt: now.Add(time.Duration(c.ttl) * time.Second),
	}

	return true
}

// GetPrices retrieves multiple prices from cache (batch fetch)
func (c *Cache) GetPrices(_ context.Context, keys []string, currency string) (map[string]*price.Price, error) {
	if len(keys) == 0 {
		return make(map[string]*price.Price), nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make(map[string]*price.Price, len(keys))
	now := time.Now()

	for _, key := range keys {
		cacheKey := c.cacheKey(key, currency)
		v, ok := c.items[cacheKey]
		if !ok {
			continue
		}

		// Check if expired
		if v.ExpiresAt.Before(now) {
			// Item expired, will be cleaned up on next write
			continue
		}

		results[key] = v.price
	}

	return results, nil
}

// SetPrices sets multiple prices in cache (batch set)
func (c *Cache) SetPrices(_ context.Context, prices map[*price.Token]*price.Price, currency string) error {
	if len(prices) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, price := range prices {
		c.items[c.cacheKey(key.ID, currency)] = &cacheItem{
			price:     price,
			ExpiresAt: now.Add(time.Duration(c.ttl) * time.Second),
		}
	}

	return nil
}

// cacheKey generates a cache key from a token and currency
func (c *Cache) cacheKey(tokenID string, currency string) string {
	key := tokenID
	return fmt.Sprintf("%s:%s", key, currency)
}

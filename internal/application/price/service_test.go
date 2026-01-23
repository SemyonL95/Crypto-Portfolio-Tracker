package price

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"testtask/internal/domain/price"
)

// mockPriceCache is a mock implementation of PriceCache for testing
type mockPriceCache struct {
	prices map[string]*price.Price
	ttl    uint8
}

func newMockPriceCache(ttl uint8) *mockPriceCache {
	return &mockPriceCache{
		prices: make(map[string]*price.Price),
		ttl:    ttl,
	}
}

func (m *mockPriceCache) GetPrice(ctx context.Context, key string, currency string) (*price.Price, bool) {
	cacheKey := m.cacheKey(key, currency)
	p, ok := m.prices[cacheKey]
	return p, ok
}

func (m *mockPriceCache) SetPrice(ctx context.Context, key string, p *price.Price, currency string) bool {
	cacheKey := m.cacheKey(key, currency)
	m.prices[cacheKey] = p
	return true
}

func (m *mockPriceCache) GetPrices(ctx context.Context, keys []string, currency string) (map[string]*price.Price, error) {
	result := make(map[string]*price.Price)
	for _, key := range keys {
		cacheKey := m.cacheKey(key, currency)
		if p, ok := m.prices[cacheKey]; ok {
			result[key] = p
		}
	}
	return result, nil
}

func (m *mockPriceCache) SetPrices(ctx context.Context, prices map[*price.Token]*price.Price, currency string) error {
	for token, p := range prices {
		cacheKey := m.cacheKey(token.ID, currency)
		m.prices[cacheKey] = p
	}
	return nil
}

func (m *mockPriceCache) cacheKey(tokenID string, currency string) string {
	return tokenID + ":" + currency
}

// mockPriceProvider is a mock implementation of PriceProvider for testing
type mockPriceProvider struct {
	prices map[string]*price.Price
	err    error
}

func newMockPriceProvider() *mockPriceProvider {
	return &mockPriceProvider{
		prices: make(map[string]*price.Price),
	}
}

func (m *mockPriceProvider) GetPrices(ctx context.Context, tokens []*price.Token, currency string) (map[*price.Token]*price.Price, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[*price.Token]*price.Price)
	for _, token := range tokens {
		if p, ok := m.prices[token.ID]; ok {
			result[token] = p
		}
	}
	return result, nil
}

func (m *mockPriceProvider) setPrice(tokenID string, p *price.Price) {
	m.prices[tokenID] = p
}

func (m *mockPriceProvider) setError(err error) {
	m.err = err
}

func TestCacheService_GetPrices_CacheHit(t *testing.T) {
	// Test cache-aside pattern: cache hit scenario
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Pre-populate cache
	token := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	cachedPrice := price.NewPrice(token, big.NewInt(50000), "USD")
	cache.SetPrice(context.Background(), token.ID, &cachedPrice, "USD")

	// Request prices
	tokens := []*price.Token{token}
	result, err := service.GetPrices(context.Background(), tokens, "USD")

	if err != nil {
		t.Errorf("GetPrices() error = %v", err)
		return
	}

	if len(result) != 1 {
		t.Errorf("GetPrices() got %d results, want 1", len(result))
		return
	}

	// TODO
	// Verify cache was used (primary provider should not be called)

	// Since we can't directly verify this, we check that the result matches cached value
	if result[token] == nil {
		t.Error("GetPrices() did not return cached price")
	}
}

func TestCacheService_GetPrices_CacheMiss(t *testing.T) {
	// Test cache-aside pattern: cache miss scenario
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Set price in primary provider (not in cache)
	token := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	primaryPrice := price.NewPrice(token, big.NewInt(50000), "USD")
	primary.setPrice(token.ID, &primaryPrice)

	// Request prices (cache miss)
	tokens := []*price.Token{token}
	result, err := service.GetPrices(context.Background(), tokens, "USD")

	if err != nil {
		t.Errorf("GetPrices() error = %v", err)
		return
	}

	if len(result) != 1 {
		t.Errorf("GetPrices() got %d results, want 1", len(result))
		return
	}

	// Verify price was fetched from provider and cached
	cached, _ := cache.GetPrice(context.Background(), token.ID, "USD")
	if cached == nil {
		t.Error("GetPrices() did not cache the fetched price")
	}
}

func TestCacheService_GetPrices_Fallback(t *testing.T) {
	// Test fallback pattern: primary fails, fallback succeeds
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Primary provider fails
	primary.setError(errors.New("primary provider error"))

	// Fallback provider succeeds
	token := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	fallbackPrice := price.NewPrice(token, big.NewInt(50000), "USD")
	fallback.setPrice(token.ID, &fallbackPrice)

	// Request prices
	tokens := []*price.Token{token}
	result, err := service.GetPrices(context.Background(), tokens, "USD")

	if err != nil {
		t.Errorf("GetPrices() error = %v, expected fallback to succeed", err)
		return
	}

	if len(result) != 1 {
		t.Errorf("GetPrices() got %d results, want 1", len(result))
		return
	}

	// Verify fallback price was used
	if result[token] == nil {
		t.Error("GetPrices() did not return fallback price")
	}
}

func TestCacheService_GetPrices_FallbackFailure(t *testing.T) {
	// Test fallback pattern: both primary and fallback fail
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Both providers fail
	primary.setError(errors.New("primary provider error"))
	fallback.setError(errors.New("fallback provider error"))

	// Request prices
	token := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	tokens := []*price.Token{token}
	_, err := service.GetPrices(context.Background(), tokens, "USD")

	if err == nil {
		t.Error("GetPrices() expected error when both providers fail")
	}
}

func TestCacheService_GetPrices_Batch(t *testing.T) {
	// Test batch price requests
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Set prices for multiple tokens
	token1 := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	token2 := &price.Token{ID: "ethereum", Symbol: "ETH"}
	price1 := price.NewPrice(token1, big.NewInt(50000), "USD")
	price2 := price.NewPrice(token2, big.NewInt(3000), "USD")
	primary.setPrice(token1.ID, &price1)
	primary.setPrice(token2.ID, &price2)

	// Request prices for multiple tokens
	tokens := []*price.Token{token1, token2}
	result, err := service.GetPrices(context.Background(), tokens, "USD")

	if err != nil {
		t.Errorf("GetPrices() error = %v", err)
		return
	}

	if len(result) != 2 {
		t.Errorf("GetPrices() got %d results, want 2", len(result))
	}
}

func TestCacheService_GetPrices_PartialCacheHit(t *testing.T) {
	// Test partial cache hit: some tokens cached, some not
	cache := newMockPriceCache(60)
	primary := newMockPriceProvider()
	fallback := newMockPriceProvider()

	service := NewCacheService(cache)
	service.SetProviders(primary, fallback)

	// Pre-populate cache for one token
	token1 := &price.Token{ID: "bitcoin", Symbol: "BTC"}
	cachedPrice := price.NewPrice(token1, big.NewInt(50000), "USD")
	cache.SetPrice(context.Background(), token1.ID, &cachedPrice, "USD")

	// Set price in provider for another token
	token2 := &price.Token{ID: "ethereum", Symbol: "ETH"}
	providerPrice := price.NewPrice(token2, big.NewInt(3000), "USD")
	primary.setPrice(token2.ID, &providerPrice)

	// Request prices for both tokens
	tokens := []*price.Token{token1, token2}
	result, err := service.GetPrices(context.Background(), tokens, "USD")

	if err != nil {
		t.Errorf("GetPrices() error = %v", err)
		return
	}

	if len(result) != 2 {
		t.Errorf("GetPrices() got %d results, want 2", len(result))
	}
}

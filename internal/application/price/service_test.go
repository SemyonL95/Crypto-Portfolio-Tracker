package price

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	domainprice "testtask/internal/domain/price"
	"testtask/internal/domain/token"
	"time"
)

// mockCache implements domain.Cache[string, domainprice.Price]
type mockCache struct {
	mu         sync.RWMutex
	items      map[string]domainprice.Price
	getCalls   int
	setCalls   int
	setupCalls int // Track calls made during setup
}

func newMockCache() *mockCache {
	return &mockCache{
		items: make(map[string]domainprice.Price),
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (domainprice.Price, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.getCalls++
	v, ok := m.items[key]
	return v, ok
}

func (m *mockCache) Set(ctx context.Context, key string, value domainprice.Price) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	m.items[key] = value
}

func (m *mockCache) resetCallCounters() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setupCalls = m.setCalls
	m.setCalls = 0
	m.getCalls = 0
}

func (m *mockCache) getSetCallsAfterSetup() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.setCalls
}

func (m *mockCache) GetBatch(ctx context.Context, keys []string) map[string]domainprice.Price {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.getCalls++
	result := make(map[string]domainprice.Price)
	for _, key := range keys {
		if v, ok := m.items[key]; ok {
			result[key] = v
		}
	}
	return result
}

func (m *mockCache) SetBatch(ctx context.Context, items map[string]domainprice.Price) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setCalls++
	for k, v := range items {
		m.items[k] = v
	}
}

// mockProvider implements domainprice.Provider
type mockProvider struct {
	prices    map[string]*domainprice.Price
	err       error
	callCount int
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		prices: make(map[string]*domainprice.Price),
	}
}

func (m *mockProvider) GetPrices(
	ctx context.Context,
	tokens []*token.Token,
	currency string,
) (map[*token.Token]*domainprice.Price, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[*token.Token]*domainprice.Price)
	for _, t := range tokens {
		if p, ok := m.prices[t.ID]; ok {
			// Create a copy with the correct token pointer
			priceCopy := *p
			priceCopy.Token = t
			result[t] = &priceCopy
		}
	}
	return result, nil
}

func (m *mockProvider) setPrice(tokenID string, p *domainprice.Price) {
	m.prices[tokenID] = p
}

func (m *mockProvider) setError(err error) {
	m.err = err
}

// mockRateLimiter implements domain.RateLimiterService
type mockRateLimiter struct {
	mu          sync.Mutex
	allowCalls  int
	shouldAllow bool
	err         error
}

func newMockRateLimiter() *mockRateLimiter {
	return &mockRateLimiter{
		shouldAllow: true,
	}
}

func (m *mockRateLimiter) Allow(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowCalls++
	if m.err != nil {
		return m.err
	}
	if !m.shouldAllow {
		return errors.New("rate limit exceeded")
	}
	return nil
}

func (m *mockRateLimiter) setShouldAllow(allow bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldAllow = allow
}

func (m *mockRateLimiter) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *mockRateLimiter) getAllowCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.allowCalls
}

func TestService_GetPrices(t *testing.T) {
	tests := []struct {
		name            string
		setupCache      func(*mockCache)
		setupPrimary    func(*mockProvider)
		setupFallback   func(*mockProvider)
		tokens          []*token.Token
		currency        string
		wantErr         bool
		wantResultCount int
		wantCacheHit    bool
		wantAPIHit      bool
		wantMockHit     bool
		validateResult  func(*testing.T, map[*token.Token]*domainprice.Price, *mockCache, *mockProvider, *mockProvider)
	}{
		{
			name: "cache hit - valid cached price",
			setupCache: func(c *mockCache) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				cachedPrice := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				cachedPrice.LastUpdated = time.Now() // Fresh cache
				c.Set(context.Background(), "bitcoin:USD", cachedPrice)
			},
			setupPrimary: func(p *mockProvider) {
				// Should not be called
			},
			setupFallback: func(p *mockProvider) {
				// Should not be called
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 1,
			wantCacheHit:    true,
			wantAPIHit:      false,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount > 0 {
					t.Error("Primary provider should not be called on cache hit")
				}
				if fallback.callCount > 0 {
					t.Error("Fallback provider should not be called on cache hit")
				}
				if cache.getSetCallsAfterSetup() > 0 {
					t.Error("Cache should not be written to on cache hit")
				}
			},
		},
		{
			name: "API hit - cache miss, primary provider succeeds",
			setupCache: func(c *mockCache) {
				// Empty cache
			},
			setupPrimary: func(p *mockProvider) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				price := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				p.setPrice("bitcoin", &price)
			},
			setupFallback: func(p *mockProvider) {
				// Should not be called
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 1,
			wantCacheHit:    false,
			wantAPIHit:      true,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount != 1 {
					t.Errorf("Primary provider should be called once, got %d", primary.callCount)
				}
				if fallback.callCount > 0 {
					t.Error("Fallback provider should not be called when primary succeeds")
				}
				if cache.setCalls == 0 {
					t.Error("Cache should be written to after API hit")
				}
				// Verify price was cached
				cached, ok := cache.Get(context.Background(), "bitcoin:USD")
				if !ok {
					t.Error("Price should be cached after API hit")
				}
				if cached.Value.Cmp(big.NewInt(5000000000000)) != 0 {
					t.Errorf("Cached price value mismatch, got %s", cached.Value.String())
				}
			},
		},
		{
			name: "mock hit - cache miss, primary fails, fallback succeeds",
			setupCache: func(c *mockCache) {
				// Empty cache
			},
			setupPrimary: func(p *mockProvider) {
				p.setError(errors.New("primary provider error"))
			},
			setupFallback: func(p *mockProvider) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				price := domainprice.NewPrice(btcToken, big.NewInt(1000000000), "USD")
				p.setPrice("bitcoin", &price)
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 1,
			wantCacheHit:    false,
			wantAPIHit:      false,
			wantMockHit:     true,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount != 1 {
					t.Errorf("Primary provider should be called once, got %d", primary.callCount)
				}
				if fallback.callCount != 1 {
					t.Errorf("Fallback provider should be called once, got %d", fallback.callCount)
				}
				if cache.setCalls == 0 {
					t.Error("Cache should be written to after fallback hit")
				}
				// Verify fallback price was cached
				cached, ok := cache.Get(context.Background(), "bitcoin:USD")
				if !ok {
					t.Error("Price should be cached after fallback hit")
				}
				if cached.Value.Cmp(big.NewInt(1000000000)) != 0 {
					t.Errorf("Cached price value mismatch, got %s, want 1000000000", cached.Value.String())
				}
			},
		},
		{
			name: "expired cache - should refetch from API",
			setupCache: func(c *mockCache) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				cachedPrice := domainprice.NewPrice(btcToken, big.NewInt(4000000000000), "USD")
				cachedPrice.LastUpdated = time.Now().Add(-2 * time.Minute) // Expired
				c.Set(context.Background(), "bitcoin:USD", cachedPrice)
			},
			setupPrimary: func(p *mockProvider) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				price := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				p.setPrice("bitcoin", &price)
			},
			setupFallback: func(p *mockProvider) {
				// Should not be called
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 1,
			wantCacheHit:    false,
			wantAPIHit:      true,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount != 1 {
					t.Errorf("Primary provider should be called once for expired cache, got %d", primary.callCount)
				}
				// Verify new price was cached (overwriting expired one)
				cached, ok := cache.Get(context.Background(), "bitcoin:USD")
				if !ok {
					t.Error("New price should be cached after refetch")
				}
				if cached.Value.Cmp(big.NewInt(5000000000000)) != 0 {
					t.Errorf("Cached price should be updated, got %s, want 5000000000000", cached.Value.String())
				}
			},
		},
		{
			name: "batch request - partial cache hit",
			setupCache: func(c *mockCache) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				cachedPrice := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				cachedPrice.LastUpdated = time.Now()
				c.Set(context.Background(), "bitcoin:USD", cachedPrice)
			},
			setupPrimary: func(p *mockProvider) {
				ethToken := &token.Token{ID: "ethereum", Symbol: "ETH"}
				price := domainprice.NewPrice(ethToken, big.NewInt(300000000000), "USD")
				p.setPrice("ethereum", &price)
			},
			setupFallback: func(p *mockProvider) {
				// Should not be called
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
				{ID: "ethereum", Symbol: "ETH"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 2,
			wantCacheHit:    true,
			wantAPIHit:      true,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount != 1 {
					t.Errorf("Primary provider should be called once for cache miss, got %d", primary.callCount)
				}
				if len(results) != 2 {
					t.Errorf("Should return 2 prices, got %d", len(results))
				}
			},
		},
		{
			name: "both providers fail",
			setupCache: func(c *mockCache) {
				// Empty cache
			},
			setupPrimary: func(p *mockProvider) {
				p.setError(errors.New("primary provider error"))
			},
			setupFallback: func(p *mockProvider) {
				p.setError(errors.New("fallback provider error"))
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         true,
			wantResultCount: 0,
			wantCacheHit:    false,
			wantAPIHit:      false,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount != 1 {
					t.Errorf("Primary provider should be called once, got %d", primary.callCount)
				}
				if fallback.callCount != 1 {
					t.Errorf("Fallback provider should be called once, got %d", fallback.callCount)
				}
			},
		},
		{
			name: "empty tokens list",
			setupCache: func(c *mockCache) {
				// Empty cache
			},
			setupPrimary: func(p *mockProvider) {
				// Should not be called
			},
			setupFallback: func(p *mockProvider) {
				// Should not be called
			},
			tokens:          []*token.Token{},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 0,
			wantCacheHit:    false,
			wantAPIHit:      false,
			wantMockHit:     false,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, cache *mockCache, primary *mockProvider, fallback *mockProvider) {
				if primary.callCount > 0 {
					t.Error("Primary provider should not be called for empty tokens")
				}
				if fallback.callCount > 0 {
					t.Error("Fallback provider should not be called for empty tokens")
				}
				if len(results) != 0 {
					t.Errorf("Should return empty results, got %d", len(results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cache := newMockCache()
			primary := newMockProvider()
			fallback := newMockProvider()

			tt.setupCache(cache)
			tt.setupPrimary(primary)
			tt.setupFallback(fallback)

			cache.resetCallCounters()

			service := NewService(cache, primary, fallback, nil)

			results, err := service.GetPrices(context.Background(), tt.tokens, tt.currency)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // Skip further checks if error expected
			}

			if len(results) != tt.wantResultCount {
				t.Errorf("GetPrices() got %d results, want %d", len(results), tt.wantResultCount)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, results, cache, primary, fallback)
			}
		})
	}
}

func TestRateLimitedService_GetPrices(t *testing.T) {
	tests := []struct {
		name             string
		maxBatchSize     int
		setupRateLimiter func(*mockRateLimiter)
		setupProvider    func(*mockProvider)
		tokens           []*token.Token
		currency         string
		wantErr          bool
		wantResultCount  int
		wantBatches      int
		validateResult   func(*testing.T, map[*token.Token]*domainprice.Price, *mockRateLimiter, *mockProvider)
	}{
		{
			name:         "single token - rate limit allows",
			maxBatchSize: 10,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(true) // Allow returns nil
			},
			setupProvider: func(p *mockProvider) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				btcPrice := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				p.setPrice("bitcoin", &btcPrice)
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 1, // Provider should be called when allowed
			wantBatches:     1,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if rl.getAllowCalls() != 1 {
					t.Errorf("Rate limiter should be called once, got %d", rl.getAllowCalls())
				}
				if p.callCount != 1 {
					t.Errorf("Provider should be called when rate limit allows, got %d calls", p.callCount)
				}
				if len(results) != 1 {
					t.Errorf("Should return 1 price, got %d", len(results))
				}
			},
		},
		{
			name:         "rate limit exceeded - provider not called",
			maxBatchSize: 5,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(false) // Allow returns error
			},
			setupProvider: func(p *mockProvider) {
				// Provider should not be called when rate limit is exceeded
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
				{ID: "ethereum", Symbol: "ETH"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 0, // No results because provider is not called
			wantBatches:     1,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if rl.getAllowCalls() != 1 {
					t.Errorf("Rate limiter should be called once, got %d", rl.getAllowCalls())
				}
				if p.callCount != 0 {
					t.Errorf("Provider should NOT be called when rate limit exceeded, got %d calls", p.callCount)
				}
				if len(results) != 0 {
					t.Errorf("Should return 0 prices when rate limited, got %d", len(results))
				}
			},
		},
		{
			name:         "multiple batches - rate limit exceeded for all batches",
			maxBatchSize: 2,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(false) // Allow returns error, so provider should NOT be called
			},
			setupProvider: func(p *mockProvider) {
				// Provider should not be called when rate limit is exceeded
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
				{ID: "ethereum", Symbol: "ETH"},
				{ID: "solana", Symbol: "SOL"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 0, // No results because provider is not called
			wantBatches:     2,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if len(results) != 0 {
					t.Errorf("Should return 0 prices when rate limited, got %d", len(results))
				}
				if rl.getAllowCalls() != 2 {
					t.Errorf("Rate limiter should be called twice (2 batches), got %d", rl.getAllowCalls())
				}
				// Provider should NOT be called when rate limit is exceeded
				if p.callCount != 0 {
					t.Errorf("Provider should NOT be called when rate limit exceeded, got %d calls", p.callCount)
				}
			},
		},
		{
			name:         "provider error - should return error",
			maxBatchSize: 5,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(true) // Rate limit allows, so provider will be called
			},
			setupProvider: func(p *mockProvider) {
				p.setError(errors.New("provider error"))
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
			},
			currency:        "USD",
			wantErr:         true, // Should return error when provider fails
			wantResultCount: 0,
			wantBatches:     1,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if p.callCount != 1 {
					t.Errorf("Provider should be called once, got %d", p.callCount)
				}
				if len(results) != 0 {
					t.Errorf("Should return empty results on provider error, got %d", len(results))
				}
			},
		},
		{
			name:         "empty tokens list",
			maxBatchSize: 5,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(true)
			},
			setupProvider: func(p *mockProvider) {
				// Should not be called
			},
			tokens:          []*token.Token{},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 0,
			wantBatches:     0,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if rl.getAllowCalls() != 0 {
					t.Errorf("Rate limiter should not be called for empty tokens, got %d", rl.getAllowCalls())
				}
				if p.callCount != 0 {
					t.Errorf("Provider should not be called for empty tokens, got %d", p.callCount)
				}
				if len(results) != 0 {
					t.Errorf("Should return empty results, got %d", len(results))
				}
			},
		},
		{
			name:         "exact batch size boundary - rate limit allows",
			maxBatchSize: 3,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(true) // Rate limit allows, so provider will be called
			},
			setupProvider: func(p *mockProvider) {
				btcToken := &token.Token{ID: "bitcoin", Symbol: "BTC"}
				ethToken := &token.Token{ID: "ethereum", Symbol: "ETH"}
				solToken := &token.Token{ID: "solana", Symbol: "SOL"}
				btcPrice := domainprice.NewPrice(btcToken, big.NewInt(5000000000000), "USD")
				ethPrice := domainprice.NewPrice(ethToken, big.NewInt(300000000000), "USD")
				solPrice := domainprice.NewPrice(solToken, big.NewInt(10000000000), "USD")
				p.setPrice("bitcoin", &btcPrice)
				p.setPrice("ethereum", &ethPrice)
				p.setPrice("solana", &solPrice)
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
				{ID: "ethereum", Symbol: "ETH"},
				{ID: "solana", Symbol: "SOL"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 3,
			wantBatches:     1,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if len(results) != 3 {
					t.Errorf("Should return 3 prices, got %d", len(results))
				}
				if rl.getAllowCalls() != 1 {
					t.Errorf("Rate limiter should be called once (single batch), got %d", rl.getAllowCalls())
				}
				if p.callCount != 1 {
					t.Errorf("Provider should be called once, got %d", p.callCount)
				}
			},
		},
		{
			name:         "large batch - multiple batches with rate limit allows",
			maxBatchSize: 2,
			setupRateLimiter: func(rl *mockRateLimiter) {
				rl.setShouldAllow(true) // Rate limit allows, so provider will be called
			},
			setupProvider: func(p *mockProvider) {
				tokens := []string{"bitcoin", "ethereum", "solana", "cardano", "polkadot"}
				for _, tokenID := range tokens {
					token := &token.Token{ID: tokenID, Symbol: tokenID[:3]}
					price := domainprice.NewPrice(token, big.NewInt(1000000000), "USD")
					p.setPrice(tokenID, &price)
				}
			},
			tokens: []*token.Token{
				{ID: "bitcoin", Symbol: "BTC"},
				{ID: "ethereum", Symbol: "ETH"},
				{ID: "solana", Symbol: "SOL"},
				{ID: "cardano", Symbol: "ADA"},
				{ID: "polkadot", Symbol: "DOT"},
			},
			currency:        "USD",
			wantErr:         false,
			wantResultCount: 5,
			wantBatches:     3,
			validateResult: func(t *testing.T, results map[*token.Token]*domainprice.Price, rl *mockRateLimiter, p *mockProvider) {
				if len(results) != 5 {
					t.Errorf("Should return 5 prices, got %d", len(results))
				}
				if rl.getAllowCalls() != 3 {
					t.Errorf("Rate limiter should be called 3 times (3 batches), got %d", rl.getAllowCalls())
				}
				if p.callCount != 3 {
					t.Errorf("Provider should be called 3 times (3 batches), got %d", p.callCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			rateLimiter := newMockRateLimiter()
			provider := newMockProvider()

			tt.setupRateLimiter(rateLimiter)
			tt.setupProvider(provider)

			service := &RateLimitedService{
				maxBatchSize: tt.maxBatchSize,
				rateLimiter:  rateLimiter,
				provider:     provider,
			}

			results, err := service.GetPrices(context.Background(), tt.tokens, tt.currency)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return // Skip further checks if error expected
			}

			if len(results) != tt.wantResultCount {
				t.Errorf("GetPrices() got %d results, want %d", len(results), tt.wantResultCount)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, results, rateLimiter, provider)
			}
		})
	}
}

package price

import (
	"context"
	"fmt"
	"testtask/internal/application/ratelimiter"
	"testtask/internal/domain"
	domainPrice "testtask/internal/domain/price"
	domainToken "testtask/internal/domain/token"
	"time"
)

const cacheTTL = 1 * time.Minute

type Service struct {
	cacheTTL         time.Duration
	cache            domain.Cache[string, domainPrice.Price]
	primaryProvider  domainPrice.Provider
	fallbackProvider domainPrice.Provider
	rateLimiter      *ratelimiter.RateLimiter
}

type RateLimitedService struct {
	maxBatchSize int
	rateLimiter  domain.RateLimiterService
	provider     domainPrice.Provider
}

func (r *RateLimitedService) GetPrices(ctx context.Context, tokens []*domainToken.Token, currency string) (map[*domainToken.Token]*domainPrice.Price, error) {
	results := make(map[*domainToken.Token]*domainPrice.Price)

	for i := 0; i < len(tokens); i += r.maxBatchSize {
		end := i + r.maxBatchSize
		if end > len(tokens) {
			end = len(tokens)
		}

		batchTokens := tokens[i:end]

		batchResults := make(map[*domainToken.Token]*domainPrice.Price)
		// If rate limiter allows (returns nil), call the provider
		if err := r.rateLimiter.Allow(ctx); err == nil {
			res, err := r.provider.GetPrices(ctx, batchTokens, currency)
			if err != nil {
				// TODO add logging
				return nil, err
			}
			batchResults = res
		}
		// If rate limit exceeded (error returned), skip this batch

		for token, price := range batchResults {
			results[token] = price
		}
	}

	return results, nil
}

func NewService(
	cache domain.Cache[string, domainPrice.Price],
	primaryProvider domainPrice.Provider,
	fallbackProvider domainPrice.Provider,
	rateLimiter *ratelimiter.RateLimiter,
) *Service {
	return &Service{
		cacheTTL:         cacheTTL,
		cache:            cache,
		primaryProvider:  primaryProvider,
		fallbackProvider: fallbackProvider,
		rateLimiter:      rateLimiter,
	}
}

// GetPrices retrieves prices for multiple tokens with cache-aside pattern
// 1. First tries to get from cache
// 2. If cache misses, goes to primary provider (CoinGecko API)
// 3. If primary fails, goes to fallback provider (mock)
// 4. Caches the results with 1 minute TTL
func (s *Service) GetPrices(
	ctx context.Context,
	tokens []*domainToken.Token,
	currency string,
) (map[*domainToken.Token]*domainPrice.Price, error) {
	if len(tokens) == 0 {
		return make(map[*domainToken.Token]*domainPrice.Price), nil
	}

	results := make(map[*domainToken.Token]*domainPrice.Price)
	var missedTokens []*domainToken.Token

	cacheKeys := make([]string, 0, len(tokens))
	tokenToKey := make(map[string]*domainToken.Token)
	for _, t := range tokens {
		key := s.cacheKey(t.ID, currency)
		cacheKeys = append(cacheKeys, key)
		tokenToKey[key] = t
	}

	cachedPrices := s.cache.GetBatch(ctx, cacheKeys)
	now := time.Now()

	for key, cachedPrice := range cachedPrices {
		t := tokenToKey[key]
		if t == nil {
			continue
		}

		if now.Sub(cachedPrice.LastUpdated) < s.cacheTTL {
			priceCopy := cachedPrice
			results[t] = &priceCopy
		} else {
			missedTokens = append(missedTokens, t)
		}
	}

	for _, t := range tokens {
		if _, found := results[t]; !found {
			missedTokens = append(missedTokens, t)
		}
	}

	if len(missedTokens) > 0 {
		var fetched map[*domainToken.Token]*domainPrice.Price
		var err error

		// If rate limiter is provided, check rate limit before calling primary provider
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Allow(ctx); err == nil {
				// Rate limit allows, try primary provider
				fetched, err = s.primaryProvider.GetPrices(ctx, missedTokens, currency)
			} else {
				// Rate limit exceeded, skip primary and go to fallback
				err = fmt.Errorf("rate limit exceeded: %w", err)
			}
		} else {
			// No rate limiter, call primary provider directly
			fetched, err = s.primaryProvider.GetPrices(ctx, missedTokens, currency)
		}

		// If primary failed or was rate limited, try fallback
		if err != nil {
			fetched, err = s.fallbackProvider.GetPrices(ctx, missedTokens, currency)
			if err != nil {
				return nil, fmt.Errorf("both primary and fallback providers failed: %w", err)
			}
		}

		for t, p := range fetched {
			results[t] = p
		}

		s.cacheFetchedPrices(ctx, fetched, currency)
	}

	return results, nil
}

func (s *Service) cacheKey(tokenID, currency string) string {
	return fmt.Sprintf("%s:%s", tokenID, currency)
}

func (s *Service) cacheFetchedPrices(
	ctx context.Context,
	prices map[*domainToken.Token]*domainPrice.Price,
	currency string,
) {
	cacheItems := make(map[string]domainPrice.Price, len(prices))
	for t, p := range prices {
		key := s.cacheKey(t.ID, currency)
		cacheItems[key] = *p
	}
	s.cache.SetBatch(ctx, cacheItems)
}

package price

import (
	"context"
	"fmt"
	loggeradapter "testtask/internal/adapters/logger"
	"testtask/internal/application/ratelimiter"
	"testtask/internal/domain"
	domainPrice "testtask/internal/domain/price"
	domainToken "testtask/internal/domain/token"
	"time"

	"go.uber.org/zap"
)

const cacheTTL = 1 * time.Minute

type Service struct {
	cacheTTL         time.Duration
	cache            domain.Cache[string, domainPrice.Price]
	primaryProvider  domainPrice.Provider
	fallbackProvider domainPrice.Provider
	rateLimiter      *ratelimiter.RateLimiter
	logger           *loggeradapter.Logger
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
	logger *loggeradapter.Logger,
) *Service {
	if logger == nil {
		logger = loggeradapter.NewNopLogger()
	}
	return &Service{
		cacheTTL:         cacheTTL,
		cache:            cache,
		primaryProvider:  primaryProvider,
		fallbackProvider: fallbackProvider,
		rateLimiter:      rateLimiter,
		logger:           logger,
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
		s.logger.Debug("GetPrices called with empty tokens list")
		return make(map[*domainToken.Token]*domainPrice.Price), nil
	}

	s.logger.Info("Getting prices", zap.Int("token_count", len(tokens)), zap.String("currency", currency))

	results := make(map[*domainToken.Token]*domainPrice.Price)
	var missedTokens []*domainToken.Token

	cacheKeys := make([]string, 0, len(tokens))
	tokenToKey := make(map[string]*domainToken.Token)
	for _, t := range tokens {
		key := s.cacheKey(t.Address, currency)
		cacheKeys = append(cacheKeys, key)
		tokenToKey[key] = t
	}

	cachedPrices := s.cache.GetBatch(ctx, cacheKeys)
	now := time.Now()
	cacheHits := 0
	cacheMisses := 0

	for key, cachedPrice := range cachedPrices {
		t := tokenToKey[key]
		if t == nil {
			continue
		}

		if now.Sub(cachedPrice.LastUpdated) < s.cacheTTL {
			priceCopy := cachedPrice
			results[t] = &priceCopy
			cacheHits++
		} else {
			missedTokens = append(missedTokens, t)
			cacheMisses++
		}
	}

	for _, t := range tokens {
		if _, found := results[t]; !found {
			missedTokens = append(missedTokens, t)
			cacheMisses++
		}
	}

	s.logger.Info("Cache lookup completed", zap.Int("cache_hits", cacheHits), zap.Int("cache_misses", cacheMisses), zap.Int("total_tokens", len(tokens)))

	if len(missedTokens) > 0 {
		s.logger.Info("Fetching prices from provider", zap.Int("missed_token_count", len(missedTokens)), zap.String("currency", currency))
		var fetched map[*domainToken.Token]*domainPrice.Price
		var err error
		usedFallback := false

		// If rate limiter is provided, check rate limit before calling primary provider
		rateLimitErr := s.rateLimiter.Allow(ctx)
		if rateLimitErr == nil {
			// Rate limit allows, try primary provider
			s.logger.Debug("Rate limit allows, calling primary provider", zap.Int("token_count", len(missedTokens)))
			fetched, err = s.primaryProvider.GetPrices(ctx, missedTokens, currency)
			if err != nil {
				s.logger.Warn("Primary provider failed", zap.Error(err))
			} else {
				s.logger.Info("Successfully fetched prices from primary provider", zap.Int("price_count", len(fetched)))
			}
		} else {
			// Rate limit exceeded, skip primary and go to fallback
			s.logger.Warn("Rate limit exceeded, using fallback provider", zap.Error(rateLimitErr))
			err = fmt.Errorf("rate limit exceeded: %w", rateLimitErr)
		}

		// If primary failed or was rate limited, try fallback
		if err != nil {
			s.logger.Info("Falling back to fallback provider", zap.Int("token_count", len(missedTokens)))
			usedFallback = true
			fetched, err = s.fallbackProvider.GetPrices(ctx, missedTokens, currency)
			if err != nil {
				s.logger.Error("Both primary and fallback providers failed", zap.Error(err))
				return nil, fmt.Errorf("both primary and fallback providers failed: %w", err)
			}
			s.logger.Info("Successfully fetched prices from fallback provider", zap.Int("price_count", len(fetched)))
		}

		for t, p := range fetched {
			results[t] = p
		}

		s.cacheFetchedPrices(ctx, fetched, currency)
		if usedFallback {
			s.logger.Debug("Cached prices from fallback provider", zap.Int("price_count", len(fetched)))
		} else {
			s.logger.Debug("Cached prices from primary provider", zap.Int("price_count", len(fetched)))
		}
	}

	s.logger.Info("Successfully retrieved all prices", zap.Int("total_prices", len(results)), zap.String("currency", currency))
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
		key := s.cacheKey(t.Address, currency)
		cacheItems[key] = *p
	}
	s.cache.SetBatch(ctx, cacheItems)
}

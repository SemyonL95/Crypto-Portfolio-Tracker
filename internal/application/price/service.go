package price

import (
	"context"
	"testtask/internal/domain/price"
)

// CacheService wraps a PriceProvider with cache-aside pattern and TTL
// This is an adapter that adds caching behavior to any PriceProvider
type CacheService struct {
	cache            price.PriceCache
	primaryProvider  price.PriceProvider
	fallbackProvider price.PriceProvider
}

// NewCacheService creates a new cached price provider
func NewCacheService(cache price.PriceCache) *CacheService {
	return &CacheService{
		cache: cache,
	}
}

// SetProviders sets the primary and fallback price providers
func (s *CacheService) SetProviders(primary, fallback price.PriceProvider) {
	s.primaryProvider = primary
	s.fallbackProvider = fallback
}

// GetPrices retrieves prices for multiple tokens with cache-aside pattern
// Implements PriceProvider interface from domain
func (s *CacheService) GetPrices(
	ctx context.Context,
	tokens []*price.Token,
	currency string,
) (map[*price.Token]*price.Price, error) {
	results := make(map[*price.Token]*price.Price)
	var missedTokens []*price.Token

	// Check cache for all tokens
	var tokenIDs []string
	for _, token := range tokens {
		tokenIDs = append(tokenIDs, token.ID)
	}

	prices, err := s.cache.GetPrices(ctx, tokenIDs, currency)
	if err == nil {
		for _, token := range tokens {
			if _, ok := prices[token.ID]; !ok {
				missedTokens = append(missedTokens, token)
			} else {
				results[token] = prices[token.ID]
			}
		}
	} else {
		// TODO log error
		missedTokens = tokens
	}

	// Fetch missed tokens from provider
	if len(missedTokens) > 0 {
		fetched, err := s.primaryProvider.GetPrices(ctx, missedTokens, currency)
		if err != nil {
			//TODO log error

			// we assume that here will no errors
			fetched, err = s.fallbackProvider.GetPrices(ctx, missedTokens, currency)
			if err != nil {
				return nil, err
			}
		}

		for token, price := range fetched {
			results[token] = price
		}

		errCache := s.cache.SetPrices(ctx, results, currency)
		if errCache != nil {
			// TODO log err
		}
	}

	return results, nil
}

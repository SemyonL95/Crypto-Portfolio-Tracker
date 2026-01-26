package domain

import "context"

type RateLimiterService interface {
	Allow(ctx context.Context) error
}

type Cache[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, bool)
	Set(ctx context.Context, key K, value V)
	GetBatch(ctx context.Context, keys []K) map[K]V
	SetBatch(ctx context.Context, items map[K]V)
}

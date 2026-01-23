package domain

import "context"

type RateLimiterService interface {
	Allow(ctx context.Context) error
}

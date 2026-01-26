package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimiter is a generic rate limiter that limits calls within a time window
// It uses a token bucket algorithm with a sliding window approach
type RateLimiter struct {
	mu              sync.Mutex
	maxCalls        int
	windowDuration  time.Duration
	callTimestamps  []time.Time
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

// NewRateLimiter creates a new rate limiter with the specified max calls and window duration
func NewRateLimiter(maxCalls int, windowDuration time.Duration) *RateLimiter {
	if maxCalls <= 0 {
		maxCalls = 1 // Minimum 1 call
	}
	if windowDuration <= 0 {
		windowDuration = time.Minute // Default to 1 minute
	}

	// Set cleanup interval to 10% of window duration, but at least 1 second
	cleanupInterval := windowDuration / 10
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	return &RateLimiter{
		maxCalls:        maxCalls,
		windowDuration:  windowDuration,
		callTimestamps:  make([]time.Time, 0, maxCalls),
		cleanupInterval: cleanupInterval,
		lastCleanup:     time.Now(),
	}
}

func (rl *RateLimiter) Allow(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if now.Sub(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup(now)
		rl.lastCleanup = now
	}

	cutoff := now.Add(-rl.windowDuration)
	validTimestamps := rl.callTimestamps[:0]
	for _, ts := range rl.callTimestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	rl.callTimestamps = validTimestamps

	if len(rl.callTimestamps) >= rl.maxCalls {
		return ErrRateLimitExceeded
	}

	rl.callTimestamps = append(rl.callTimestamps, now)
	return nil
}

// cleanup removes old timestamps outside the window
func (rl *RateLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-rl.windowDuration)
	validTimestamps := rl.callTimestamps[:0]
	for _, ts := range rl.callTimestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	rl.callTimestamps = validTimestamps
}

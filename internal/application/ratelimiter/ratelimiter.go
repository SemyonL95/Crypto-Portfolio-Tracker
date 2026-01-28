package ratelimiter

import (
	"context"
	"errors"
	"sync"
	loggeradapter "testtask/internal/adapters/logger"
	"time"

	"go.uber.org/zap"
)

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

type RateLimiter struct {
	mu              sync.Mutex
	maxCalls        int
	windowDuration  time.Duration
	callTimestamps  []time.Time
	cleanupInterval time.Duration
	lastCleanup     time.Time
	logger          *loggeradapter.Logger
}

func NewRateLimiter(maxCalls int, windowDuration time.Duration, logger *loggeradapter.Logger) *RateLimiter {
	if maxCalls <= 0 {
		maxCalls = 1 // Minimum 1 call
	}
	if windowDuration <= 0 {
		windowDuration = time.Minute // Default to 1 minute
	}
	if logger == nil {
		logger = loggeradapter.NewNopLogger()
	}

	cleanupInterval := windowDuration / 10
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	logger.Info("Rate limiter created",
		zap.Int("max_calls", maxCalls),
		zap.Duration("window_duration", windowDuration),
		zap.Duration("cleanup_interval", cleanupInterval),
	)

	return &RateLimiter{
		maxCalls:        maxCalls,
		windowDuration:  windowDuration,
		callTimestamps:  make([]time.Time, 0, maxCalls),
		cleanupInterval: cleanupInterval,
		lastCleanup:     time.Now(),
		logger:          logger,
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
		rl.logger.Warn("Rate limit exceeded",
			zap.Int("current_calls", len(rl.callTimestamps)),
			zap.Int("max_calls", rl.maxCalls),
			zap.Duration("window_duration", rl.windowDuration),
		)
		return ErrRateLimitExceeded
	}

	rl.callTimestamps = append(rl.callTimestamps, now)
	rl.logger.Debug("Rate limit check passed",
		zap.Int("current_calls", len(rl.callTimestamps)),
		zap.Int("max_calls", rl.maxCalls),
	)
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

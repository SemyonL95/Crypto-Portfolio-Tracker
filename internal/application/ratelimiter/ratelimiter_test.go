package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name          string
		maxCalls      int
		windowDuration time.Duration
		wantMaxCalls   int
		wantWindow     time.Duration
	}{
		{
			name:          "positive calls with minute window",
			maxCalls:      10,
			windowDuration: time.Minute,
			wantMaxCalls:   10,
			wantWindow:     time.Minute,
		},
		{
			name:          "zero calls defaults to 1",
			maxCalls:      0,
			windowDuration: time.Minute,
			wantMaxCalls:   1,
			wantWindow:     time.Minute,
		},
		{
			name:          "negative calls defaults to 1",
			maxCalls:      -5,
			windowDuration: time.Minute,
			wantMaxCalls:   1,
			wantWindow:     time.Minute,
		},
		{
			name:          "single call per minute",
			maxCalls:      1,
			windowDuration: time.Minute,
			wantMaxCalls:   1,
			wantWindow:     time.Minute,
		},
		{
			name:          "custom duration - 30 seconds",
			maxCalls:      5,
			windowDuration: 30 * time.Second,
			wantMaxCalls:   5,
			wantWindow:     30 * time.Second,
		},
		{
			name:          "custom duration - 5 minutes",
			maxCalls:      100,
			windowDuration: 5 * time.Minute,
			wantMaxCalls:   100,
			wantWindow:     5 * time.Minute,
		},
		{
			name:          "zero duration defaults to minute",
			maxCalls:      10,
			windowDuration: 0,
			wantMaxCalls:   10,
			wantWindow:     time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.maxCalls, tt.windowDuration, nil)
			if rl.maxCalls != tt.wantMaxCalls {
				t.Errorf("NewRateLimiter() maxCalls = %d, want %d", rl.maxCalls, tt.wantMaxCalls)
			}
			if rl.windowDuration != tt.wantWindow {
				t.Errorf("NewRateLimiter() windowDuration = %v, want %v", rl.windowDuration, tt.wantWindow)
			}
			if rl.callTimestamps == nil {
				t.Error("NewRateLimiter() callTimestamps should not be nil")
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name          string
		maxCalls      int
		windowDuration time.Duration
		numCalls       int
		wantErrors     int
		wantSuccesses  int
	}{
		{
			name:          "allow calls within limit",
			maxCalls:      5,
			windowDuration: time.Minute,
			numCalls:       5,
			wantErrors:     0,
			wantSuccesses:  5,
		},
		{
			name:          "reject calls exceeding limit",
			maxCalls:      3,
			windowDuration: time.Minute,
			numCalls:       5,
			wantErrors:     2,
			wantSuccesses:  3,
		},
		{
			name:          "exactly at limit",
			maxCalls:      10,
			windowDuration: time.Minute,
			numCalls:       10,
			wantErrors:     0,
			wantSuccesses:  10,
		},
		{
			name:          "one over limit",
			maxCalls:      10,
			windowDuration: time.Minute,
			numCalls:       11,
			wantErrors:     1,
			wantSuccesses:  10,
		},
		{
			name:          "single call limit",
			maxCalls:      1,
			windowDuration: time.Minute,
			numCalls:       2,
			wantErrors:     1,
			wantSuccesses:  1,
		},
		{
			name:          "large limit",
			maxCalls:      100,
			windowDuration: time.Minute,
			numCalls:       100,
			wantErrors:     0,
			wantSuccesses:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.maxCalls, tt.windowDuration, nil)
			ctx := context.Background()
			errors := 0
			successes := 0

			for i := 0; i < tt.numCalls; i++ {
				if err := rl.Allow(ctx); err != nil {
					if err != ErrRateLimitExceeded {
						t.Errorf("Allow() error = %v, want ErrRateLimitExceeded", err)
					}
					errors++
				} else {
					successes++
				}
			}

			if errors != tt.wantErrors {
				t.Errorf("Allow() got %d errors, want %d", errors, tt.wantErrors)
			}
			if successes != tt.wantSuccesses {
				t.Errorf("Allow() got %d successes, want %d", successes, tt.wantSuccesses)
			}
		})
	}
}

func TestRateLimiter_Allow_Concurrent(t *testing.T) {
	tests := []struct {
		name          string
		maxCalls      int
		windowDuration time.Duration
		numCalls       int
		wantErrors     int
		wantSuccesses  int
	}{
		{
			name:          "concurrent calls within limit",
			maxCalls:      10,
			windowDuration: time.Minute,
			numCalls:       10,
			wantErrors:     0,
			wantSuccesses:  10,
		},
		{
			name:          "concurrent calls exceeding limit",
			maxCalls:      5,
			windowDuration: time.Minute,
			numCalls:       20,
			wantErrors:     15,
			wantSuccesses:  5,
		},
		{
			name:          "many concurrent calls",
			maxCalls:      3,
			windowDuration: time.Minute,
			numCalls:       50,
			wantErrors:     47,
			wantSuccesses:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.maxCalls, tt.windowDuration, nil)
			ctx := context.Background()
			var wg sync.WaitGroup
			errorsChan := make(chan error, tt.numCalls)
			successes := make(chan bool, tt.numCalls)

			for i := 0; i < tt.numCalls; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := rl.Allow(ctx); err != nil {
						errorsChan <- err
					} else {
						successes <- true
					}
				}()
			}

			wg.Wait()
			close(errorsChan)
			close(successes)

			errorCount := 0
			for err := range errorsChan {
				if !errors.Is(err, ErrRateLimitExceeded) {
					t.Errorf("Allow() error = %v, want ErrRateLimitExceeded", err)
				}
				errorCount++
			}

			successCount := 0
			for range successes {
				successCount++
			}

			if errorCount != tt.wantErrors {
				t.Errorf("Allow() got %d errors, want %d", errorCount, tt.wantErrors)
			}
			if successCount != tt.wantSuccesses {
				t.Errorf("Allow() got %d successes, want %d", successCount, tt.wantSuccesses)
			}
		})
	}
}

func TestRateLimiter_Allow_Context(t *testing.T) {
	tests := []struct {
		name        string
		cancelCtx   bool
		wantSuccess bool
	}{
		{
			name:        "normal context",
			cancelCtx:   false,
			wantSuccess: true,
		},
		{
			name:        "cancelled context",
			cancelCtx:   true,
			wantSuccess: true, // Allow doesn't check context, only Wait does
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(10, time.Minute, nil)
			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err := rl.Allow(ctx)
			if (err == nil) != tt.wantSuccess {
				t.Errorf("Allow() error = %v, want success = %v", err, tt.wantSuccess)
			}
		})
	}
}

func TestRateLimiter_Allow_SequentialBursts(t *testing.T) {
		rl := NewRateLimiter(3, time.Minute, nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := rl.Allow(ctx); err != nil {
			t.Errorf("Allow() call %d failed unexpectedly: %v", i+1, err)
		}
	}

	if err := rl.Allow(ctx); !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Allow() 4th call should fail, got error: %v", err)
	}
}

func TestRateLimiter_Allow_RapidCalls(t *testing.T) {
	rl := NewRateLimiter(5, time.Minute, nil)
	ctx := context.Background()

	errors := 0
	successes := 0
	for i := 0; i < 10; i++ {
		if err := rl.Allow(ctx); err != nil {
			errors++
		} else {
			successes++
		}
	}

	if successes != 5 {
		t.Errorf("Allow() rapid calls got %d successes, want 5", successes)
	}
	if errors != 5 {
		t.Errorf("Allow() rapid calls got %d errors, want 5", errors)
	}
}

func TestRateLimiter_Allow_MultipleInstances(t *testing.T) {
	// Test that different rate limiter instances don't interfere
	rl1 := NewRateLimiter(3, time.Minute, nil)
	rl2 := NewRateLimiter(5, time.Minute, nil)
	ctx := context.Background()

	// Both should work independently
	for i := 0; i < 3; i++ {
		if err := rl1.Allow(ctx); err != nil {
			t.Errorf("Allow() rl1 call %d failed: %v", i+1, err)
		}
	}
	if err := rl1.Allow(ctx); !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Allow() rl1 4th call should fail, got: %v", err)
	}

	// rl2 should still have capacity
	for i := 0; i < 5; i++ {
		if err := rl2.Allow(ctx); err != nil {
			t.Errorf("Allow() rl2 call %d failed: %v", i+1, err)
		}
	}
	if err := rl2.Allow(ctx); !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Allow() rl2 6th call should fail, got: %v", err)
	}
}

func TestRateLimiter_Allow_CustomDuration(t *testing.T) {
	// Test with a 2-second window
	rl := NewRateLimiter(3, 2*time.Second, nil)
	ctx := context.Background()

	// Should allow 3 calls
	for i := 0; i < 3; i++ {
		if err := rl.Allow(ctx); err != nil {
			t.Errorf("Allow() call %d failed unexpectedly: %v", i+1, err)
		}
	}

	// 4th call should fail
	if err := rl.Allow(ctx); !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("Allow() 4th call should fail, got: %v", err)
	}

	// Wait for window to expire (2 seconds + small buffer)
	time.Sleep(2100 * time.Millisecond)

	// Should allow calls again after window expires
	for i := 0; i < 3; i++ {
		if err := rl.Allow(ctx); err != nil {
			t.Errorf("Allow() call %d after wait failed unexpectedly: %v", i+1, err)
		}
	}
}

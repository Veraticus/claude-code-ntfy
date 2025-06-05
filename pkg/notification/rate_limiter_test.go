package notification

import (
	"testing"
	"time"
)

func TestTokenBucketRateLimiter_Allow(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		refillRate time.Duration
		operations []struct {
			delay     time.Duration
			wantAllow bool
		}
	}{
		{
			name:       "allow up to capacity immediately",
			capacity:   3,
			refillRate: time.Hour, // slow refill
			operations: []struct {
				delay     time.Duration
				wantAllow bool
			}{
				{delay: 0, wantAllow: true},  // 1st
				{delay: 0, wantAllow: true},  // 2nd
				{delay: 0, wantAllow: true},  // 3rd
				{delay: 0, wantAllow: false}, // 4th - should be denied
			},
		},
		{
			name:       "refill allows more operations",
			capacity:   2,
			refillRate: 100 * time.Millisecond,
			operations: []struct {
				delay     time.Duration
				wantAllow bool
			}{
				{delay: 0, wantAllow: true},                      // 1st
				{delay: 0, wantAllow: true},                      // 2nd
				{delay: 0, wantAllow: false},                     // 3rd - denied
				{delay: 150 * time.Millisecond, wantAllow: true}, // After refill
				{delay: 0, wantAllow: false},                     // Denied again
			},
		},
		{
			name:       "single token capacity",
			capacity:   1,
			refillRate: 50 * time.Millisecond,
			operations: []struct {
				delay     time.Duration
				wantAllow bool
			}{
				{delay: 0, wantAllow: true},
				{delay: 0, wantAllow: false},
				{delay: 60 * time.Millisecond, wantAllow: true},
				{delay: 0, wantAllow: false},
			},
		},
		{
			name:       "zero capacity always denies",
			capacity:   0,
			refillRate: time.Millisecond,
			operations: []struct {
				delay     time.Duration
				wantAllow bool
			}{
				{delay: 0, wantAllow: false},
				{delay: 10 * time.Millisecond, wantAllow: false},
				{delay: 100 * time.Millisecond, wantAllow: false},
			},
		},
		{
			name:       "fast refill rate",
			capacity:   3,
			refillRate: 10 * time.Millisecond,
			operations: []struct {
				delay     time.Duration
				wantAllow bool
			}{
				{delay: 0, wantAllow: true},
				{delay: 0, wantAllow: true},
				{delay: 0, wantAllow: true},
				{delay: 0, wantAllow: false},
				{delay: 15 * time.Millisecond, wantAllow: true},
				{delay: 15 * time.Millisecond, wantAllow: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewTokenBucketRateLimiter(tt.capacity, tt.refillRate)

			for i, op := range tt.operations {
				if op.delay > 0 {
					time.Sleep(op.delay)
				}

				got := limiter.Allow()
				if got != op.wantAllow {
					t.Errorf("operation[%d]: Allow() = %v, want %v", i, got, op.wantAllow)
				}
			}
		})
	}
}

func TestTokenBucketRateLimiter_Refill(t *testing.T) {
	// Test that tokens are properly capped at capacity during refill
	limiter := NewTokenBucketRateLimiter(2, 50*time.Millisecond)

	// Use all tokens
	limiter.Allow() // tokens = 1
	limiter.Allow() // tokens = 0

	// Wait for multiple refill periods
	time.Sleep(200 * time.Millisecond) // Should refill 4 times

	// Should only allow up to capacity
	if !limiter.Allow() {
		t.Error("Expected Allow() = true after refill")
	}
	if !limiter.Allow() {
		t.Error("Expected Allow() = true for second token")
	}
	if limiter.Allow() {
		t.Error("Expected Allow() = false, tokens should be capped at capacity")
	}
}

func TestNewTokenBucketRateLimiter(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		refillRate time.Duration
	}{
		{
			name:       "normal parameters",
			capacity:   10,
			refillRate: time.Second,
		},
		{
			name:       "zero capacity",
			capacity:   0,
			refillRate: time.Minute,
		},
		{
			name:       "large capacity",
			capacity:   1000000,
			refillRate: time.Hour, // Use slow refill to avoid timing issues
		},
		{
			name:       "negative capacity treated as zero",
			capacity:   -5,
			refillRate: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewTokenBucketRateLimiter(tt.capacity, tt.refillRate)
			if limiter == nil {
				t.Error("NewTokenBucketRateLimiter() returned nil")
			}

			// For negative capacity, it should be treated as 0
			expectedCapacity := tt.capacity
			if expectedCapacity < 0 {
				expectedCapacity = 0
			}

			// Test initial state - should allow up to capacity
			allowed := 0
			for i := 0; i < expectedCapacity+5; i++ {
				if limiter.Allow() {
					allowed++
				}
			}

			if allowed != expectedCapacity {
				t.Errorf("Initial tokens = %d, want %d", allowed, expectedCapacity)
			}
		})
	}
}

func TestTokenBucketRateLimiter_Concurrent(t *testing.T) {
	capacity := 100
	limiter := NewTokenBucketRateLimiter(capacity, time.Hour) // slow refill

	// Run concurrent Allow() calls
	done := make(chan bool, capacity*2)
	allowed := make(chan bool, capacity*2)

	for i := 0; i < capacity*2; i++ {
		go func() {
			result := limiter.Allow()
			allowed <- result
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < capacity*2; i++ {
		<-done
	}
	close(allowed)

	// Count allowed operations
	allowedCount := 0
	for result := range allowed {
		if result {
			allowedCount++
		}
	}

	// Should allow exactly capacity operations
	if allowedCount != capacity {
		t.Errorf("Concurrent Allow() count = %d, want %d", allowedCount, capacity)
	}
}

package security

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter is a simple per-tool rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	hits     map[string][]time.Time
	maxHits  int
	window   time.Duration
}

// NewRateLimiter creates a rate limiter.
func NewRateLimiter(maxHits int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		hits:    make(map[string][]time.Time),
		maxHits: maxHits,
		window:  window,
	}
}

// Allow checks if a tool call is allowed.
func (r *RateLimiter) Allow(tool string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-r.window)
	var valid []time.Time
	for _, t := range r.hits[tool] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= r.maxHits {
		return fmt.Errorf("rate limit exceeded for tool %s", tool)
	}
	valid = append(valid, now)
	r.hits[tool] = valid
	return nil
}

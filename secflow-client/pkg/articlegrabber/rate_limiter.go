// Package articlegrabber provides article crawlers for security news and articles.
package articlegrabber

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides adaptive rate limiting with exponential backoff per source.
type RateLimiter struct {
	mu         sync.Mutex
	requests   map[string]*rateLimitEntry
	maxRetries int
}

// rateLimitEntry tracks rate limit state for a single source.
type rateLimitEntry struct {
	lastRequest time.Time
	retryCount  int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

// NewRateLimiter creates a new RateLimiter with default settings.
// Default max retries: 5, base delay: 1s, max delay: 60s.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests:   make(map[string]*rateLimitEntry),
		maxRetries: 5,
	}
}

// NewRateLimiterWithConfig creates a RateLimiter with custom configuration.
func NewRateLimiterWithConfig(maxRetries int, baseDelay, maxDelay time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:   make(map[string]*rateLimitEntry),
		maxRetries: maxRetries,
	}
}

// Wait blocks until the context allows proceeding or context is cancelled.
// Returns nil if waiting succeeded, or context error if cancelled/timed out.
func (r *RateLimiter) Wait(ctx context.Context, source string) error {
	delay := r.GetDelay(source)

	r.mu.Lock()
	entry, exists := r.requests[source]
	if !exists {
		entry = &rateLimitEntry{
			baseDelay: 1 * time.Second,
			maxDelay:  60 * time.Second,
		}
		r.requests[source] = entry
	}
	entry.lastRequest = time.Now()
	r.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil
}

// RecordSuccess resets the retry count for a source after a successful request.
func (r *RateLimiter) RecordSuccess(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.requests[source]; exists {
		entry.retryCount = 0
	}
}

// RecordFailure increments the retry count for a source after a failed request.
// Returns the delay to wait before next retry.
func (r *RateLimiter) RecordFailure(source string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.requests[source]
	if !exists {
		entry = &rateLimitEntry{
			baseDelay: 1 * time.Second,
			maxDelay:  60 * time.Second,
		}
		r.requests[source] = entry
	}

	if entry.retryCount < r.maxRetries {
		entry.retryCount++
	}

	delay := r.calculateDelay(entry)
	return delay
}

// GetDelay returns the current delay for a source based on retry count.
// Returns 0 if no delays are needed (first request or after success).
func (r *RateLimiter) GetDelay(source string) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.requests[source]
	if !exists {
		return 0
	}

	return r.calculateDelay(entry)
}

// calculateDelay computes exponential backoff delay: baseDelay * 2^retryCount (capped at maxDelay).
func (r *RateLimiter) calculateDelay(entry *rateLimitEntry) time.Duration {
	if entry.retryCount == 0 {
		return 0
	}

	// Calculate delay with exponential backoff
	delay := entry.baseDelay
	for i := 0; i < entry.retryCount; i++ {
		delay *= 2
		if delay >= entry.maxDelay {
			return entry.maxDelay
		}
	}

	// Cap at max delay
	if delay > entry.maxDelay {
		delay = entry.maxDelay
	}

	return delay
}

// SetDelays sets custom base and max delays for a source.
func (r *RateLimiter) SetDelays(source string, baseDelay, maxDelay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.requests[source]
	if !exists {
		entry = &rateLimitEntry{}
		r.requests[source] = entry
	}
	entry.baseDelay = baseDelay
	entry.maxDelay = maxDelay
}

// GetRetryCount returns the current retry count for a source.
func (r *RateLimiter) GetRetryCount(source string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.requests[source]; exists {
		return entry.retryCount
	}
	return 0
}

// Reset clears all rate limit state.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = make(map[string]*rateLimitEntry)
}

// ResetSource clears rate limit state for a specific source.
func (r *RateLimiter) ResetSource(source string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.requests, source)
}

// Stats returns statistics about all tracked sources.
func (r *RateLimiter) Stats() map[string]struct {
	RetryCount int
	LastRequest time.Time
	NextDelay  time.Duration
} {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := make(map[string]struct {
		RetryCount  int
		LastRequest time.Time
		NextDelay   time.Duration
	})

	for source, entry := range r.requests {
		stats[source] = struct {
			RetryCount  int
			LastRequest time.Time
			NextDelay   time.Duration
		}{
			RetryCount:  entry.retryCount,
			LastRequest: entry.lastRequest,
			NextDelay:   r.calculateDelay(entry),
		}
	}

	return stats
}
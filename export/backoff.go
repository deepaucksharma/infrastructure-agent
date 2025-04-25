package export

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// BackoffRetrier implements retry logic with exponential backoff.
type BackoffRetrier struct {
	config RetryConfig
	rand   *rand.Rand
}

// NewBackoffRetrier creates a new BackoffRetrier with the given configuration.
func NewBackoffRetrier(config RetryConfig) *BackoffRetrier {
	source := rand.NewSource(time.Now().UnixNano())
	return &BackoffRetrier{
		config: config,
		rand:   rand.New(source),
	}
}

// nextBackoff calculates the next backoff duration with jitter.
func (br *BackoffRetrier) nextBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate backoff with multiplier
	backoff := br.config.InitialInterval * time.Duration(math.Pow(br.config.Multiplier, float64(attempt-1)))
	
	// Cap at max interval
	if backoff > br.config.MaxInterval {
		backoff = br.config.MaxInterval
	}
	
	// Add jitter (±20%)
	jitter := float64(backoff) * (0.8 + 0.4*br.rand.Float64())
	return time.Duration(jitter)
}

// Do executes the given function with retries according to the retry configuration.
func (br *BackoffRetrier) Do(ctx context.Context, fn func() error) error {
	if !br.config.Enabled {
		return fn() // If retries are disabled, just execute once
	}

	var err error
	var attempt int

	for attempt = 1; attempt <= br.config.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil // Success, no need to retry
		}

		// Check if we've reached max attempts
		if attempt >= br.config.MaxAttempts {
			break
		}

		// Calculate backoff time
		backoff := br.nextBackoff(attempt)

		// Create timer and wait for either backoff or context done
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err() // Context was cancelled
		case <-timer.C:
			// Continue to next attempt
		}
	}

	return err // Return the last error
}

// DoWithFallback executes the given function with retries and falls back to another function if all retries fail.
func (br *BackoffRetrier) DoWithFallback(ctx context.Context, fn func() error, fallback func(error) error) error {
	err := br.Do(ctx, fn)
	if err != nil {
		return fallback(err)
	}
	return nil
}

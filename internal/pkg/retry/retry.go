package goretry

import (
	"context"
	"time"
)

// Option tunes retry behavior.
type Option func(*Config)

// Config controls retry parameters.
type Config struct {
	MaxAttempts int
	Backoff     Backoff
	Jitter      JitterMode
	RetryIf     func(err error) bool
	OnRetry     func(attempt int, err error, delay time.Duration)
}

// DefaultConfig provides resilient production defaults.
var DefaultConfig = Config{
	MaxAttempts: 3,
	Backoff:     Exponential(100*time.Millisecond, 2*time.Second, 2.0),
	Jitter:      FullJitter,
	RetryIf:     nil, // Nil retries all non-nil errors
	OnRetry:     nil,
}

// WithMaxAttempts sets total attempts (1 initial attempt + (n-1) retries).
func WithMaxAttempts(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.MaxAttempts = n
		}
	}
}

// WithBackoff sets custom backoff strategy.
func WithBackoff(b Backoff) Option {
	return func(c *Config) {
		if b != nil {
			c.Backoff = b
		}
	}
}

// WithJitter sets the jitter mode (FullJitter, EqualJitter, NoJitter).
func WithJitter(mode JitterMode) Option {
	return func(c *Config) {
		c.Jitter = mode
	}
}

// WithRetryIf specifies a predicate that determines if an error is retriable.
func WithRetryIf(fn func(err error) bool) Option {
	return func(c *Config) {
		c.RetryIf = fn
	}
}

// WithOnRetry registers a callback invoked on each retry attempt.
func WithOnRetry(fn func(attempt int, err error, delay time.Duration)) Option {
	return func(c *Config) {
		c.OnRetry = fn
	}
}

// Do executes an action with retry logic and context cancellation awareness.
func Do(ctx context.Context, action func(ctx context.Context) error, opts ...Option) error {
	cfg := DefaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before executing
		if err := ctx.Err(); err != nil {
			return err
		}

		err := action(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// If error is not retriable, exit immediately
		if cfg.RetryIf != nil && !cfg.RetryIf(err) {
			return err
		}

		// Don't wait after the last attempt
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		baseDelay := cfg.Backoff.NextDelay(attempt)
		delay := ApplyJitter(baseDelay, cfg.Jitter)

		if cfg.OnRetry != nil {
			cfg.OnRetry(attempt+1, err, delay)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return &RetryError{
		Attempts: cfg.MaxAttempts,
		LastErr:  lastErr,
	}
}

// DoWithData executes an action returning a value with retry logic.
func DoWithData[T any](ctx context.Context, action func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	var result T
	err := Do(ctx, func(ctx context.Context) error {
		val, err := action(ctx)
		if err != nil {
			return err
		}
		result = val
		return nil
	}, opts...)

	return result, err
}

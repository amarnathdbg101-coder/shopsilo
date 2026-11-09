// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"context"
	"time"

	gocb "github.com/amarnathdbg101-coder/go/go-circuitbreaker"
	goretry "github.com/amarnathdbg101-coder/go/go-retry"
	"go.uber.org/zap"
)

// Global circuit breakers for downstream services
var (
	// R2StorageBreaker protects Cloudflare R2 / S3 storage operations.
	R2StorageBreaker = gocb.New(gocb.Config{
		Name:                   "cloudflare-r2",
		MaxConsecutiveFailures: 3,
		CooldownDuration:       15 * time.Second,
		OnStateChange: func(name string, from, to gocb.State) {
			zap.L().Warn("circuit breaker state changed",
				zap.String("service", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	})

	// GeminiAIBreaker protects Gemini LLM external requests.
	GeminiAIBreaker = gocb.New(gocb.Config{
		Name:                   "gemini-ai",
		MaxConsecutiveFailures: 3,
		CooldownDuration:       20 * time.Second,
		OnStateChange: func(name string, from, to gocb.State) {
			zap.L().Warn("circuit breaker state changed",
				zap.String("service", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	})
)

// ExecuteWithRetry wraps any operation with exponential backoff and anti-thundering-herd full jitter.
func ExecuteWithRetry[T any](ctx context.Context, maxAttempts int, fn func(ctx context.Context) (T, error)) (T, error) {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	return goretry.DoWithData(
		ctx,
		fn,
		goretry.WithMaxAttempts(maxAttempts),
		goretry.WithBackoff(goretry.Exponential(50*time.Millisecond, 1*time.Second, 2.0)),
		goretry.WithJitter(goretry.FullJitter),
	)
}

// ExecuteResilient executes an operation protected by BOTH a Circuit Breaker (fast fail during outages)
// and an Exponential Backoff Retry engine (handles transient network hiccups).
func ExecuteResilient[T any](ctx context.Context, cb *gocb.CircuitBreaker, maxAttempts int, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T

	err := cb.Execute(ctx, func(execCtx context.Context) error {
		res, retryErr := ExecuteWithRetry(execCtx, maxAttempts, fn)
		if retryErr != nil {
			return retryErr
		}
		result = res
		return nil
	})

	if err != nil {
		var zero T
		return zero, err
	}
	return result, nil
}

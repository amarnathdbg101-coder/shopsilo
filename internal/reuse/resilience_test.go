package reuse_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	gocb "github.com/amarnathdbg101-coder/go/go-circuitbreaker"
	"shopMe/internal/reuse"
)

func TestExecuteWithRetry_SuccessAfterTransientErrors(t *testing.T) {
	ctx := context.Background()
	var attempts int32

	result, err := reuse.ExecuteWithRetry(ctx, 4, func(c context.Context) (string, error) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			return "", errors.New("simulated transient 503 error")
		}
		return "success_payload", nil
	})

	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if result != "success_payload" {
		t.Fatalf("expected 'success_payload', got %q", result)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected exactly 3 attempts, got %d", attempts)
	}
}

func TestExecuteResilient_CircuitBreakerTripsOnOutage(t *testing.T) {
	ctx := context.Background()

	// Isolated test circuit breaker
	testBreaker := gocb.New(gocb.Config{
		Name:                   "test-downstream",
		MaxConsecutiveFailures: 2,
		CooldownDuration:       500 * time.Millisecond,
	})

	failingFn := func(c context.Context) (string, error) {
		return "", errors.New("downstream service completely down")
	}

	// 1. Failure 1
	_, err := reuse.ExecuteResilient(ctx, testBreaker, 1, failingFn)
	if err == nil {
		t.Fatal("expected failure on attempt 1")
	}

	// 2. Failure 2 -> Trips breaker to OPEN
	_, err = reuse.ExecuteResilient(ctx, testBreaker, 1, failingFn)
	if err == nil {
		t.Fatal("expected failure on attempt 2")
	}

	// 3. Attempt 3 must fast-fail immediately without executing fn
	var executed int32
	_, err = reuse.ExecuteResilient(ctx, testBreaker, 1, func(c context.Context) (string, error) {
		atomic.StoreInt32(&executed, 1)
		return "should_not_run", nil
	})

	if err == nil {
		t.Fatal("expected circuit breaker open error, got nil")
	}
	if atomic.LoadInt32(&executed) == 1 {
		t.Fatal("function was executed while circuit breaker was OPEN!")
	}
}

package gocb

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreaker_TripsAndRecovers(t *testing.T) {
	var stateTransitions int32

	cb := New(Config{
		Name:                   "test-breaker",
		MaxConsecutiveFailures: 3,
		CooldownDuration:       50 * time.Millisecond,
		OnStateChange: func(name string, from, to State) {
			atomic.AddInt32(&stateTransitions, 1)
		},
	})

	ctx := context.Background()
	failErr := errors.New("downstream error")

	// Fail 1
	_ = cb.Execute(ctx, func(ctx context.Context) error { return failErr })
	if cb.State() != StateClosed {
		t.Fatal("should still be CLOSED after 1 fail")
	}

	// Fail 2
	_ = cb.Execute(ctx, func(ctx context.Context) error { return failErr })
	if cb.State() != StateClosed {
		t.Fatal("should still be CLOSED after 2 fails")
	}

	// Fail 3 -> Trips to OPEN
	_ = cb.Execute(ctx, func(ctx context.Context) error { return failErr })
	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %s", cb.State())
	}

	// Fast Fail: should not execute action at all
	var actionCalled bool
	err := cb.Execute(ctx, func(ctx context.Context) error {
		actionCalled = true
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if actionCalled {
		t.Fatal("action should NOT have been invoked while OPEN")
	}

	// Wait for cooldown to expire
	time.Sleep(60 * time.Millisecond)

	// Canary probe: success should recover to CLOSED
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected successful probe, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after successful canary, got %s", cb.State())
	}
}

func TestCircuitBreaker_Fallback(t *testing.T) {
	cb := New(Config{
		MaxConsecutiveFailures: 1,
		CooldownDuration:       time.Minute,
	})

	ctx := context.Background()
	// Trip breaker
	_ = cb.Execute(ctx, func(ctx context.Context) error { return errors.New("boom") })

	var fallbackExecuted bool
	err := cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			t.Fatal("action should not be called")
			return nil
		},
		func(ctx context.Context, triggerErr error) error {
			fallbackExecuted = true
			if !errors.Is(triggerErr, ErrCircuitOpen) {
				t.Fatalf("expected fallback triggerErr to be ErrCircuitOpen, got %v", triggerErr)
			}
			return nil
		},
	)

	if err != nil {
		t.Fatalf("fallback returned error: %v", err)
	}
	if !fallbackExecuted {
		t.Fatal("fallback function was not executed")
	}
}

func TestCircuitBreaker_PanicRecovery(t *testing.T) {
	cb := New(Config{
		MaxConsecutiveFailures: 1,
	})

	ctx := context.Background()
	err := cb.Execute(ctx, func(ctx context.Context) error {
		panic("nil pointer dereference inside action")
	})

	if err == nil {
		t.Fatal("expected panic to be returned as error, got nil")
	}

	// Should have tripped breaker
	if cb.State() != StateOpen {
		t.Fatalf("expected breaker to trip on panic, got %s", cb.State())
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	cb1 := reg.GetOrRegister("payment", Config{MaxConsecutiveFailures: 5})
	cb2 := reg.GetOrRegister("payment", Config{MaxConsecutiveFailures: 10})

	if cb1 != cb2 {
		t.Fatal("GetOrRegister should return existing instance")
	}

	all := reg.All()
	if len(all) != 1 || all["payment"] == nil {
		t.Fatalf("All() returned unexpected map: %v", all)
	}
}

package supervisor_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"shopMe/internal/pkg/supervisor"
)

func TestPanicRecoveryAndRestart(t *testing.T) {
	var panicCount int32
	var runs int32

	sup := supervisor.New(
		supervisor.WithOnPanic(func(name string, panicVal any, stack []byte) {
			atomic.AddInt32(&panicCount, 1)
		}),
	)

	// Worker that panics on its first run, then succeeds
	err := sup.Add(supervisor.WorkerSpec{
		Name: "panic_worker",
		Func: func(ctx context.Context) error {
			count := atomic.AddInt32(&runs, 1)
			if count == 1 {
				panic("simulated nil pointer crash")
			}
			<-ctx.Done()
			return nil
		},
		Policy:  supervisor.PolicyPermanent,
		Backoff: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to add worker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go sup.Start(ctx)

	// Wait for panic and restart
	time.Sleep(30 * time.Millisecond)
	cancel()
	sup.Stop()

	if atomic.LoadInt32(&panicCount) != 1 {
		t.Fatalf("expected 1 captured panic, got %d", atomic.LoadInt32(&panicCount))
	}
	if atomic.LoadInt32(&runs) < 2 {
		t.Fatalf("expected at least 2 runs (original + restart), got %d", atomic.LoadInt32(&runs))
	}
}

func TestTransientPolicy(t *testing.T) {
	sup := supervisor.New()
	var cleanRuns int32
	var errorRuns int32

	// Clean worker (exits nil) -> should NOT restart
	_ = sup.Add(supervisor.WorkerSpec{
		Name: "clean_worker",
		Func: func(ctx context.Context) error {
			atomic.AddInt32(&cleanRuns, 1)
			return nil
		},
		Policy: supervisor.PolicyTransient,
	})

	// Failing worker (exits error) -> should restart
	_ = sup.Add(supervisor.WorkerSpec{
		Name: "error_worker",
		Func: func(ctx context.Context) error {
			val := atomic.AddInt32(&errorRuns, 1)
			if val == 1 {
				return errors.New("glitch")
			}
			<-ctx.Done()
			return nil
		},
		Policy:  supervisor.PolicyTransient,
		Backoff: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go sup.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel()
	sup.Stop()

	if atomic.LoadInt32(&cleanRuns) != 1 {
		t.Errorf("clean transient worker should execute only once, got %d", atomic.LoadInt32(&cleanRuns))
	}
	if atomic.LoadInt32(&errorRuns) < 2 {
		t.Errorf("failing transient worker should restart, got %d", atomic.LoadInt32(&errorRuns))
	}
}

func TestCrashLoopProtection(t *testing.T) {
	sup := supervisor.New(
		supervisor.WithCrashLoopProtection(2, 500*time.Millisecond),
	)

	var runs int32
	_ = sup.Add(supervisor.WorkerSpec{
		Name: "looping_worker",
		Func: func(ctx context.Context) error {
			atomic.AddInt32(&runs, 1)
			return errors.New("perpetual failure")
		},
		Policy:  supervisor.PolicyPermanent,
		Backoff: 2 * time.Millisecond,
	})

	ctx := context.Background()
	_ = sup.Start(ctx)

	// Since max restarts is 2, runs should be capped at 3 (1 initial + 2 restarts)
	totalRuns := atomic.LoadInt32(&runs)
	if totalRuns > 3 {
		t.Fatalf("expected crash loop circuit breaker to stop at <=3 runs, got %d", totalRuns)
	}
}


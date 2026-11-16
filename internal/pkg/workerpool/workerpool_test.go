package goworkerpool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Submit_100Jobs(t *testing.T) {
	pool := New(4, 200)
	defer pool.Stop()

	var counter int64
	var wg sync.WaitGroup
	const totalJobs = 100

	wg.Add(totalJobs)
	for i := 0; i < totalJobs; i++ {
		err := pool.Submit(func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		})
		if err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}

	wg.Wait()

	if atomic.LoadInt64(&counter) != totalJobs {
		t.Fatalf("expected counter %d, got %d", totalJobs, atomic.LoadInt64(&counter))
	}
}

func TestPool_TrySubmit_Backpressure(t *testing.T) {
	// 1 worker, 2 queue slots
	pool := New(1, 2)
	defer pool.StopNow()

	// Block the single worker
	blocker := make(chan struct{})
	_ = pool.Submit(func(ctx context.Context) {
		<-blocker
	})

	// Fill the 2 buffer slots
	_ = pool.Submit(func(ctx context.Context) {})
	_ = pool.Submit(func(ctx context.Context) {})

	// 4th submit should fail with ErrQueueFull
	err := pool.TrySubmit(func(ctx context.Context) {})
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull on full buffer, got %v", err)
	}

	close(blocker)
}

func TestPool_SubmitWait(t *testing.T) {
	pool := New(2, 10)
	defer pool.Stop()

	ctx := context.Background()

	// Case 1: Success
	var calculated int
	err := pool.SubmitWait(ctx, func() error {
		calculated = 40 + 2
		return nil
	})
	if err != nil || calculated != 42 {
		t.Fatalf("SubmitWait failed: err=%v, val=%d", err, calculated)
	}

	// Case 2: Action returns error
	customErr := errors.New("business error")
	err = pool.SubmitWait(ctx, func() error {
		return customErr
	})
	if !errors.Is(err, customErr) {
		t.Fatalf("expected customErr, got %v", err)
	}

	// Case 3: Context timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err = pool.SubmitWait(timeoutCtx, func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestPool_PanicRecovery(t *testing.T) {
	panicChan := make(chan struct{}, 1)

	pool := New(2, 10, WithOnPanic(func(r any) {
		panicChan <- struct{}{}
	}))
	defer pool.Stop()

	var healthyFinished atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	// Panicking job
	_ = pool.Submit(func(ctx context.Context) {
		panic("nil pointer dereference inside job")
	})

	// Healthy job
	_ = pool.Submit(func(ctx context.Context) {
		defer wg.Done()
		healthyFinished.Store(true)
	})

	// Wait for healthy job and panic report
	wg.Wait()

	select {
	case <-panicChan:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected panic to be reported to WithOnPanic callback")
	}

	if !healthyFinished.Load() {
		t.Fatal("healthy job should finish even if another job panics")
	}
}

func TestPool_GracefulStop(t *testing.T) {
	pool := New(2, 50)

	var count int64
	for i := 0; i < 20; i++ {
		_ = pool.Submit(func(ctx context.Context) {
			time.Sleep(2 * time.Millisecond)
			atomic.AddInt64(&count, 1)
		})
	}

	// Stop waits for all 20 jobs to complete
	_ = pool.Stop()

	if atomic.LoadInt64(&count) != 20 {
		t.Fatalf("expected all 20 jobs to finish during Stop, got %d", count)
	}

	// Submitting after stop must fail
	err := pool.Submit(func(ctx context.Context) {})
	if !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("expected ErrPoolClosed, got %v", err)
	}
}

func TestPool_ObservabilityHooks(t *testing.T) {
	var startCount int64
	var completedCount int64

	pool := New(2, 10, WithHooks(Hooks{
		OnJobStarted: func() {
			atomic.AddInt64(&startCount, 1)
		},
		OnJobCompleted: func(dur time.Duration) {
			atomic.AddInt64(&completedCount, 1)
		},
	}))

	_ = pool.Submit(func(ctx context.Context) {
		time.Sleep(5 * time.Millisecond)
	})
	_ = pool.Submit(func(ctx context.Context) {
		time.Sleep(5 * time.Millisecond)
	})

	_ = pool.Stop()

	if atomic.LoadInt64(&startCount) != 2 {
		t.Fatalf("expected OnJobStarted 2 times, got %d", startCount)
	}
	if atomic.LoadInt64(&completedCount) != 2 {
		t.Fatalf("expected OnJobCompleted 2 times, got %d", completedCount)
	}
}

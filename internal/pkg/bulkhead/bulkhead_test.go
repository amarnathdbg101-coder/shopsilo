package bulkhead_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shopMe/internal/pkg/bulkhead"
)

func TestBulkheadBasicExecution(t *testing.T) {
	bh := bulkhead.New(bulkhead.WithMaxConcurrent(2))
	defer bh.Close()

	ctx := context.Background()
	var executed int32

	err := bh.Execute(ctx, func() error {
		atomic.AddInt32(&executed, 1)
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&executed) != 1 {
		t.Fatalf("expected executed 1, got %d", executed)
	}
}

func TestBulkheadRejectionWhenFull(t *testing.T) {
	// 2 concurrent slots, 1 queue slot -> 4th request must be rejected immediately!
	var rejections int32
	bh := bulkhead.New(
		bulkhead.WithMaxConcurrent(2),
		bulkhead.WithMaxQueue(1),
		bulkhead.WithQueueTimeout(100*time.Millisecond),
		bulkhead.WithOnRejected(func(reason error) {
			atomic.AddInt32(&rejections, 1)
		}),
	)
	defer bh.Close()

	ctx := context.Background()
	release := make(chan struct{})
	var wg sync.WaitGroup

	// Fill the 2 active slots
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bh.Execute(ctx, func() error {
				<-release
				return nil
			})
		}()
	}

	time.Sleep(10 * time.Millisecond)

	// Fill the 1 queue slot
	wg.Add(1)
	go func() {
		defer wg.Done()
		bh.Execute(ctx, func() error {
			<-release
			return nil
		})
	}()

	time.Sleep(10 * time.Millisecond)

	// 4th request: must be rejected with ErrBulkheadFull!
	err := bh.Execute(ctx, func() error {
		return nil
	})

	if !errors.Is(err, bulkhead.ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull, got %v", err)
	}

	close(release)
	wg.Wait()

	if atomic.LoadInt32(&rejections) != 1 {
		t.Fatalf("expected 1 rejection callback, got %d", rejections)
	}
}

func TestBulkheadQueueTimeout(t *testing.T) {
	// 1 slot, 1 queue slot, but active slot is blocked for 100ms and queue timeout is 20ms
	bh := bulkhead.New(
		bulkhead.WithMaxConcurrent(1),
		bulkhead.WithMaxQueue(1),
		bulkhead.WithQueueTimeout(20*time.Millisecond),
	)
	defer bh.Close()

	ctx := context.Background()
	release := make(chan struct{})
	var wg sync.WaitGroup

	// Block the active slot
	wg.Add(1)
	go func() {
		defer wg.Done()
		bh.Execute(ctx, func() error {
			<-release
			return nil
		})
	}()

	time.Sleep(5 * time.Millisecond)

	// Second request enters queue and should timeout after 20ms
	err := bh.Execute(ctx, func() error {
		return nil
	})

	if !errors.Is(err, bulkhead.ErrQueueTimeout) {
		t.Fatalf("expected ErrQueueTimeout, got %v", err)
	}

	close(release)
	wg.Wait()
}

func TestTryExecute(t *testing.T) {
	bh := bulkhead.New(bulkhead.WithMaxConcurrent(1))
	defer bh.Close()

	blocker := make(chan struct{})
	go func() {
		bh.Execute(context.Background(), func() error {
			<-blocker
			return nil
		})
	}()

	time.Sleep(5 * time.Millisecond)

	// TryExecute should fail immediately since 1 slot is occupied
	err := bh.TryExecute(func() error {
		return nil
	})

	if !errors.Is(err, bulkhead.ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull on TryExecute, got %v", err)
	}

	close(blocker)
}

func TestExecuteWithResult(t *testing.T) {
	bh := bulkhead.New()
	defer bh.Close()

	val, err := bulkhead.ExecuteWithResult(bh, context.Background(), func() (string, error) {
		return "ShopSilo-Processed", nil
	})

	if err != nil || val != "ShopSilo-Processed" {
		t.Fatalf("expected 'ShopSilo-Processed', got %s err=%v", val, err)
	}
}


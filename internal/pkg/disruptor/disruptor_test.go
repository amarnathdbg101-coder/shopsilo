package disruptor_test

import (
	"sync"
	"testing"
	"time"

	"shopMe/internal/pkg/disruptor"
)

func TestDisruptor_BasicPublishConsume(t *testing.T) {
	rb, err := disruptor.New[string](16)
	if err != nil {
		t.Fatalf("failed to create disruptor: %v", err)
	}
	defer rb.Close()

	if rb.Len() != 0 || rb.Cap() != 16 {
		t.Fatalf("unexpected len/cap: %d / %d", rb.Len(), rb.Cap())
	}

	_ = rb.Publish("event-1")
	_ = rb.Publish("event-2")

	if rb.Len() != 2 {
		t.Fatalf("expected 2 unconsumed items, got %d", rb.Len())
	}

	item1, ok := rb.TryConsume()
	if !ok || item1 != "event-1" {
		t.Fatalf("expected event-1, got %s (ok=%v)", item1, ok)
	}

	item2, ok := rb.TryConsume()
	if !ok || item2 != "event-2" {
		t.Fatalf("expected event-2, got %s (ok=%v)", item2, ok)
	}

	_, ok = rb.TryConsume()
	if ok {
		t.Fatal("expected no more items to consume")
	}
}

func TestDisruptor_BatchOperations(t *testing.T) {
	rb, err := disruptor.New[int](64)
	if err != nil {
		t.Fatal(err)
	}
	defer rb.Close()

	batch := []int{10, 20, 30, 40, 50}
	if err := rb.PublishBatch(batch); err != nil {
		t.Fatalf("publish batch failed: %v", err)
	}

	drained := rb.Drain(10)
	if len(drained) != 5 {
		t.Fatalf("expected 5 drained items, got %d", len(drained))
	}
	for i, v := range batch {
		if drained[i] != v {
			t.Fatalf("batch element %d mismatch: expected %d, got %d", i, v, drained[i])
		}
	}
}

func TestDisruptor_BufferFullRejection(t *testing.T) {
	rb, _ := disruptor.New[int](4)
	defer rb.Close()

	// Fill to capacity 4
	for i := 0; i < 4; i++ {
		if !rb.TryPublish(i) {
			t.Fatalf("failed to publish item %d", i)
		}
	}

	// 5th item should fail TryPublish
	if rb.TryPublish(99) {
		t.Fatal("expected TryPublish to return false when ring buffer is full")
	}

	// Consume 1 item
	_, _ = rb.TryConsume()

	// Now TryPublish should succeed
	if !rb.TryPublish(99) {
		t.Fatal("expected TryPublish to succeed after consuming an item")
	}
}

func TestDisruptor_HighThroughputConcurrency(t *testing.T) {
	const capacity = 1024
	const totalItems = 10000

	rb, _ := disruptor.New[int](capacity, disruptor.WithWaitStrategy(disruptor.WaitStrategyYield))
	defer rb.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		for i := 0; i < totalItems; i++ {
			_ = rb.Publish(i)
		}
	}()

	// Consumer
	consumed := 0
	go func() {
		defer wg.Done()
		for consumed < totalItems {
			batch := rb.Drain(64)
			consumed += len(batch)
			if len(batch) == 0 {
				time.Sleep(10 * time.Microsecond)
			}
		}
	}()

	wg.Wait()

	if consumed != totalItems {
		t.Fatalf("expected %d consumed items, got %d", totalItems, consumed)
	}
}


package disruptor

import (
	"sync"
)

// RingBuffer implements the high-performance LMAX Disruptor lock-free circular queue.
type RingBuffer[T any] struct {
	mu             sync.RWMutex
	capacity       uint64
	mask           uint64
	buffer         []T
	cursor         *Sequence // Producer sequence
	gatingSequence *Sequence // Consumer sequence
	waiter         *waiter
	closed         bool
	closeChan      chan struct{}
}

// New creates an initialized Disruptor RingBuffer. Capacity must be a power of two.
func New[T any](capacity int, opts ...Option) (*RingBuffer[T], error) {
	if capacity <= 0 || (capacity&(capacity-1)) != 0 {
		return nil, ErrInvalidCapacity
	}

	cfg := DefaultConfig(capacity)
	for _, opt := range opts {
		opt(&cfg)
	}

	capU := uint64(capacity)
	return &RingBuffer[T]{
		capacity:       capU,
		mask:           capU - 1,
		buffer:         make([]T, capU),
		cursor:         NewSequence(0),
		gatingSequence: NewSequence(0),
		waiter:         newWaiter(cfg.WaitStrategy),
		closeChan:      make(chan struct{}),
	}, nil
}

// TryPublish attempts non-blocking insertion. Returns false if ring buffer is full.
func (rb *RingBuffer[T]) TryPublish(item T) bool {
	rb.mu.RLock()
	if rb.closed {
		rb.mu.RUnlock()
		return false
	}

	current := rb.cursor.Get()
	next := current + 1
	gating := rb.gatingSequence.Get()

	// Check if wrapping around past slowest consumer
	if next-gating > rb.capacity {
		rb.mu.RUnlock()
		return false
	}

	rb.buffer[next&rb.mask] = item
	rb.cursor.Set(next)
	rb.mu.RUnlock()

	rb.waiter.signal()
	return true
}

// Publish writes an item into the ring buffer, blocking if capacity is saturated.
func (rb *RingBuffer[T]) Publish(item T) error {
	spin := 0
	for {
		rb.mu.RLock()
		if rb.closed {
			rb.mu.RUnlock()
			return ErrDisruptorClosed
		}

		current := rb.cursor.Get()
		next := current + 1
		gating := rb.gatingSequence.Get()

		if next-gating <= rb.capacity {
			rb.buffer[next&rb.mask] = item
			rb.cursor.Set(next)
			rb.mu.RUnlock()
			rb.waiter.signal()
			return nil
		}
		rb.mu.RUnlock()

		spin = rb.waiter.wait(spin)
	}
}

// PublishBatch writes a batch of items into the ring buffer with a single sequence reservation.
func (rb *RingBuffer[T]) PublishBatch(items []T) error {
	batchLen := uint64(len(items))
	if batchLen == 0 {
		return nil
	}

	spin := 0
	for {
		rb.mu.RLock()
		if rb.closed {
			rb.mu.RUnlock()
			return ErrDisruptorClosed
		}

		current := rb.cursor.Get()
		next := current + batchLen
		gating := rb.gatingSequence.Get()

		if next-gating <= rb.capacity {
			for i, it := range items {
				rb.buffer[(current+1+uint64(i))&rb.mask] = it
			}
			rb.cursor.Set(next)
			rb.mu.RUnlock()
			rb.waiter.signal()
			return nil
		}
		rb.mu.RUnlock()

		spin = rb.waiter.wait(spin)
	}
}

// TryConsume extracts the next available item without blocking.
func (rb *RingBuffer[T]) TryConsume() (T, bool) {
	currentGating := rb.gatingSequence.Get()
	cursor := rb.cursor.Get()

	if currentGating >= cursor {
		var zero T
		return zero, false
	}

	next := currentGating + 1
	item := rb.buffer[next&rb.mask]
	rb.gatingSequence.Set(next)
	return item, true
}

// Drain extracts all currently available items up to maxBatch.
func (rb *RingBuffer[T]) Drain(maxBatch int) []T {
	if maxBatch <= 0 {
		maxBatch = int(rb.capacity)
	}

	currentGating := rb.gatingSequence.Get()
	cursor := rb.cursor.Get()

	available := int(cursor - currentGating)
	if available <= 0 {
		return nil
	}
	if available > maxBatch {
		available = maxBatch
	}

	res := make([]T, available)
	for i := 0; i < available; i++ {
		next := currentGating + 1 + uint64(i)
		res[i] = rb.buffer[next&rb.mask]
	}

	rb.gatingSequence.Set(currentGating + uint64(available))
	return res
}

// Len returns the current number of unconsumed items in the buffer.
func (rb *RingBuffer[T]) Len() uint64 {
	cursor := rb.cursor.Get()
	gating := rb.gatingSequence.Get()
	if cursor >= gating {
		return cursor - gating
	}
	return 0
}

// Cap returns the total ring buffer capacity.
func (rb *RingBuffer[T]) Cap() uint64 {
	return rb.capacity
}

// Close gracefully stops the disruptor.
func (rb *RingBuffer[T]) Close() error {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		return nil
	}
	rb.closed = true
	close(rb.closeChan)
	rb.waiter.signal()
	return nil
}

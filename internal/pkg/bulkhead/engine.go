package bulkhead

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Bulkhead isolates microservice execution capacity using bounded concurrency and queues.
type Bulkhead struct {
	cfg Config

	mu      sync.Mutex
	sem     chan struct{}
	closed  bool
	queueMu sync.Mutex
	queued  int

	// Telemetry counters
	totalExecuted     uint64
	totalRejected     uint64
	totalQueueTimeout uint64
}

// New creates a new Bulkhead partition with the supplied options.
func New(opts ...Option) *Bulkhead {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Bulkhead{
		cfg: cfg,
		sem: make(chan struct{}, cfg.MaxConcurrent),
	}
}

// Execute runs fn within the bulkhead boundaries.
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrBulkheadClosed
	}
	b.mu.Unlock()

	// 1. Fast path: try to acquire execution permit immediately
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		atomic.AddUint64(&b.totalExecuted, 1)
		return fn()
	default:
		// Capacity reached, proceed to queuing evaluation
	}

	// 2. Queue path: check queue capacity
	b.queueMu.Lock()
	if b.queued >= b.cfg.MaxQueue {
		b.queueMu.Unlock()
		atomic.AddUint64(&b.totalRejected, 1)
		if b.cfg.OnRejected != nil {
			b.cfg.OnRejected(ErrBulkheadFull)
		}
		return ErrBulkheadFull
	}
	b.queued++
	b.queueMu.Unlock()

	defer func() {
		b.queueMu.Lock()
		b.queued--
		b.queueMu.Unlock()
	}()

	// 3. Wait for permit, queue timeout, or context cancellation
	timer := time.NewTimer(b.cfg.QueueTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		atomic.AddUint64(&b.totalQueueTimeout, 1)
		atomic.AddUint64(&b.totalRejected, 1)
		if b.cfg.OnRejected != nil {
			b.cfg.OnRejected(ErrQueueTimeout)
		}
		return ErrQueueTimeout
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		atomic.AddUint64(&b.totalExecuted, 1)
		return fn()
	}
}

// TryExecute attempts to run fn immediately without entering the waiting queue.
func (b *Bulkhead) TryExecute(fn func() error) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrBulkheadClosed
	}
	b.mu.Unlock()

	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		atomic.AddUint64(&b.totalExecuted, 1)
		return fn()
	default:
		atomic.AddUint64(&b.totalRejected, 1)
		if b.cfg.OnRejected != nil {
			b.cfg.OnRejected(ErrBulkheadFull)
		}
		return ErrBulkheadFull
	}
}

// ActiveCount returns current concurrent executions running inside the bulkhead.
func (b *Bulkhead) ActiveCount() int {
	return len(b.sem)
}

// QueuedCount returns current tasks waiting in the queue.
func (b *Bulkhead) QueuedCount() int {
	b.queueMu.Lock()
	defer b.queueMu.Unlock()
	return b.queued
}

// Metrics returns a snapshot of runtime metrics.
func (b *Bulkhead) Metrics() Metrics {
	b.queueMu.Lock()
	q := b.queued
	b.queueMu.Unlock()

	return Metrics{
		MaxConcurrent:     b.cfg.MaxConcurrent,
		ActiveCount:       len(b.sem),
		MaxQueue:          b.cfg.MaxQueue,
		QueuedCount:       q,
		TotalExecuted:     atomic.LoadUint64(&b.totalExecuted),
		TotalRejected:     atomic.LoadUint64(&b.totalRejected),
		TotalQueueTimeout: atomic.LoadUint64(&b.totalQueueTimeout),
	}
}

// Close gracefully closes the bulkhead.
func (b *Bulkhead) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

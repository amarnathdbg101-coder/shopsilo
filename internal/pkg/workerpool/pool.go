package goworkerpool

import (
	"context"
	"sync"
	"time"
)

// Job defines an executable unit of work within the pool.
type Job func(ctx context.Context)

// Hooks defines zero-dependency observability callbacks for worker pool lifecycle and latency telemetry.
type Hooks struct {
	// OnJobStarted is called immediately before a worker begins executing a job.
	OnJobStarted func()

	// OnJobCompleted is called when a job finishes, reporting the execution duration.
	OnJobCompleted func(duration time.Duration)

	// OnPanic is called if a job triggers an unhandled panic.
	OnPanic func(recovered any)
}

// Option configures the WorkerPool.
type Option func(*Pool)

// WithOnPanic registers a panic handler to safely log or alert without crashing the worker pool.
func WithOnPanic(fn func(recovered any)) Option {
	return func(p *Pool) {
		p.onPanic = fn
	}
}

// WithHooks registers observability callbacks for task execution lifecycle.
func WithHooks(hooks Hooks) Option {
	return func(p *Pool) {
		p.hooks = hooks
	}
}

// Pool coordinates a fixed or dynamic set of worker goroutines consuming jobs from a bounded queue.
type Pool struct {
	mu      sync.RWMutex
	jobs    chan Job
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	closed  bool
	workers int
	onPanic func(recovered any)
	hooks   Hooks
}

// New initializes and launches a worker pool with the specified number of workers and job buffer.
func New(workers, queueSize int, opts ...Option) *Pool {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 1024
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		jobs:    make(chan Job, queueSize),
		ctx:     ctx,
		cancel:  cancel,
		workers: workers,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Context canceled (StopNow)
			return

		case job, ok := <-p.jobs:
			if !ok {
				// Queue closed and drained
				return
			}
			p.safeExecute(job)
		}
	}
}

func (p *Pool) safeExecute(job Job) {
	start := time.Now()
	if p.hooks.OnJobStarted != nil {
		p.hooks.OnJobStarted()
	}

	defer func() {
		dur := time.Since(start)
		if r := recover(); r != nil {
			if p.hooks.OnPanic != nil {
				p.hooks.OnPanic(r)
			} else if p.onPanic != nil {
				p.onPanic(r)
			}
		}
		if p.hooks.OnJobCompleted != nil {
			p.hooks.OnJobCompleted(dur)
		}
	}()

	job(p.ctx)
}

// Submit enqueues a job into the pool. Blocks if the queue is currently full until space frees up.
func (p *Pool) Submit(job Job) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrPoolClosed
	}
	p.mu.RUnlock()

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case p.jobs <- job:
		return nil
	}
}

// TrySubmit enqueues a job non-blockingly. Returns ErrQueueFull immediately if the buffer is at capacity.
func (p *Pool) TrySubmit(job Job) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrPoolClosed
	}
	p.mu.RUnlock()

	select {
	case <-p.ctx.Done():
		return ErrPoolClosed
	case p.jobs <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitWait submits a job and blocks the caller until the job completes or the caller context expires.
func (p *Pool) SubmitWait(ctx context.Context, action func() error) error {
	done := make(chan error, 1)

	err := p.Submit(func(poolCtx context.Context) {
		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return
		default:
		}

		done <- action()
	})

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Stop closes the job queue and waits for all queued jobs to finish executing cleanly.
func (p *Pool) Stop() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.jobs)
	p.mu.Unlock()

	p.wg.Wait()
	p.cancel()
	return nil
}

// StopNow cancels the context immediately, halting worker processing of pending jobs.
func (p *Pool) StopNow() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.cancel()
	close(p.jobs)
	p.mu.Unlock()

	p.wg.Wait()
	return nil
}

// WorkerCount returns number of active worker goroutines.
func (p *Pool) WorkerCount() int {
	return p.workers
}

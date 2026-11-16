package gooutbox

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Option tunes Engine parameters.
type Option func(*Config)

// Config configures polling, batch sizes, and retry backoff.
type Config struct {
	PollInterval   time.Duration
	BatchSize      int
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	OnError        func(event *Event, err error)
	OnSuccess      func(event *Event)
}

// DefaultConfig provides resilient production defaults.
func DefaultConfig() Config {
	return Config{
		PollInterval:   500 * time.Millisecond,
		BatchSize:      50,
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     1 * time.Minute,
	}
}

// WithPollInterval sets the polling loop frequency.
func WithPollInterval(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.PollInterval = d
		}
	}
}

// WithBatchSize sets maximum items fetched per loop tick.
func WithBatchSize(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.BatchSize = n
		}
	}
}

// WithMaxRetries sets maximum failure attempts before routing to Dead Letter Queue.
func WithMaxRetries(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.MaxRetries = n
		}
	}
}

// WithCallbacks registers success and error observation hooks.
func WithCallbacks(onSuccess func(*Event), onError func(*Event, error)) Option {
	return func(c *Config) {
		c.OnSuccess = onSuccess
		c.OnError = onError
	}
}

// Engine coordinates the background Transactional Outbox polling and dispatching loop.
type Engine struct {
	store     Store
	publisher Publisher
	cfg       Config
	isClosed  atomic.Bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// New initializes an Outbox Engine.
func New(store Store, publisher Publisher, opts ...Option) (*Engine, error) {
	if store == nil {
		return nil, ErrStoreNil
	}
	if publisher == nil {
		return nil, ErrPublisherNil
	}

	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Engine{
		store:     store,
		publisher: publisher,
		cfg:       cfg,
		stopCh:    make(chan struct{}),
	}, nil
}

// Enqueue creates and saves an outbox event.
// Pass an active database transaction (*sql.Tx, etc.) as tx to guarantee atomic consistency with domain writes.
func (e *Engine) Enqueue(ctx context.Context, tx any, topic, aggregateID string, payload any) (*Event, error) {
	if e.isClosed.Load() {
		return nil, ErrEngineClosed
	}

	event, err := NewEvent(topic, aggregateID, payload)
	if err != nil {
		return nil, err
	}
	event.MaxRetries = e.cfg.MaxRetries

	if err := e.store.Save(ctx, tx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// ProcessBatch polls a single batch of events, publishes them, and updates their lifecycle state.
// Returns count of events processed.
func (e *Engine) ProcessBatch(ctx context.Context) (int, error) {
	events, err := e.store.FetchAndLock(ctx, e.cfg.BatchSize)
	if err != nil {
		return 0, err
	}

	for _, event := range events {
		pubErr := e.publisher.Publish(ctx, event)
		if pubErr == nil {
			_ = e.store.MarkCompleted(ctx, event.ID)
			if e.cfg.OnSuccess != nil {
				e.cfg.OnSuccess(event)
			}
			continue
		}

		// Handle publishing failure
		if e.cfg.OnError != nil {
			e.cfg.OnError(event, pubErr)
		}

		if event.RetryCount+1 >= event.MaxRetries {
			_ = e.store.MarkDeadLetter(ctx, event.ID, pubErr)
		} else {
			backoff := e.calculateBackoff(event.RetryCount)
			_ = e.store.MarkFailed(ctx, event.ID, pubErr, backoff)
		}
	}

	return len(events), nil
}

func (e *Engine) calculateBackoff(retryCount int) time.Duration {
	delay := float64(e.cfg.InitialBackoff) * math.Pow(2, float64(retryCount))
	if delay > float64(e.cfg.MaxBackoff) {
		delay = float64(e.cfg.MaxBackoff)
	}
	return time.Duration(delay)
}

// Start runs the asynchronous background polling loop in a dedicated goroutine.
func (e *Engine) Start(ctx context.Context) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		ticker := time.NewTicker(e.cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopCh:
				return
			case <-ticker.C:
				_, _ = e.ProcessBatch(ctx)
			}
		}
	}()
}

// Stop signals the background worker to stop and waits for in-flight batches to finish.
func (e *Engine) Stop() {
	if e.isClosed.CompareAndSwap(false, true) {
		close(e.stopCh)
		e.wg.Wait()
	}
}

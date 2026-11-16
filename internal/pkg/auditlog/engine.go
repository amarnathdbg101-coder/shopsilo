package auditlog

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Engine is the central asynchronous audit logging orchestrator.
// High-throughput batching buffer use karta hai taaki database calls user API response time ko slow na karein.
type Engine struct {
	store    Store
	cfg      Config
	ch       chan *Entry
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	isClosed atomic.Bool
}

// New initializes an audit logging Engine and starts the background batch flusher.
func New(store Store, opts ...Option) (*Engine, error) {
	if store == nil {
		return nil, ErrStoreNil
	}

	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx, cancel := context.WithCancel(context.Background())

	e := &Engine{
		store:  store,
		cfg:    cfg,
		ch:     make(chan *Entry, cfg.BufferSize),
		ctx:    ctx,
		cancel: cancel,
	}

	// Concurrency pattern: Start background batching worker
	e.wg.Add(1)
	go e.batchLoop()

	return e, nil
}

// Log enqueues an audit entry asynchronously in sub-microsecond time.
// Sensitive fields in OldState and NewState are automatically masked before queuing.
func (e *Engine) Log(entry *Entry) error {
	if e.isClosed.Load() {
		return ErrEngineClosed
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}

	// Automatic sensitive data masking
	entry.OldState = MaskData(entry.OldState, e.cfg.SensitiveKeys...)
	entry.NewState = MaskData(entry.NewState, e.cfg.SensitiveKeys...)

	select {
	case e.ch <- entry:
		return nil
	default:
		return ErrBufferFull
	}
}

// Record is a fluent helper that creates, auto-masks, and enqueues an audit event.
func (e *Engine) Record(action, entity, entityID string, oldState, newState any) (*Entry, error) {
	entry := NewEntry(action, entity, entityID)
	entry.OldState = oldState
	entry.NewState = newState

	if err := e.Log(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// Query forwards historical audit trail search directly to underlying store.
func (e *Engine) Query(ctx context.Context, filter Filter) ([]*Entry, error) {
	return e.store.Query(ctx, filter)
}

// batchLoop continuously accumulates entries and flushes in batches either when
// BatchSize is reached or FlushInterval ticker triggers.
func (e *Engine) batchLoop() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]*Entry, 0, e.cfg.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toSave := make([]*Entry, len(batch))
		copy(toSave, batch)
		batch = batch[:0]

		if err := e.store.SaveBatch(context.Background(), toSave); err != nil {
			log.Printf("[auditlog] error saving batch of %d entries: %v\n", len(toSave), err)
		}
	}

	for {
		select {
		case entry, ok := <-e.ch:
			if !ok {
				flush() // Drain remaining
				return
			}
			batch = append(batch, entry)
			if len(batch) >= e.cfg.BatchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-e.ctx.Done():
			// Drain remaining in channel
			for {
				select {
				case entry, ok := <-e.ch:
					if ok {
						batch = append(batch, entry)
					} else {
						flush()
						return
					}
				default:
					flush()
					return
				}
			}
		}
	}
}

// Close gracefully flushes all buffered entries and terminates background workers.
func (e *Engine) Close() error {
	if !e.isClosed.CompareAndSwap(false, true) {
		return nil
	}

	close(e.ch)
	e.cancel()
	e.wg.Wait()
	return e.store.Close()
}

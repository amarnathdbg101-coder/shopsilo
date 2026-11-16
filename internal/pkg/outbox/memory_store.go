package gooutbox

import (
	"context"
	"sync"
	"time"
)

// MemoryStore provides a thread-safe, in-memory implementation of Store for testing and microservices.
type MemoryStore struct {
	mu     sync.RWMutex
	events map[string]*Event
	order  []string
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: make(map[string]*Event),
		order:  make([]string, 0),
	}
}

// Save stores an event in memory.
func (s *MemoryStore) Save(ctx context.Context, tx any, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[event.ID] = event
	s.order = append(s.order, event.ID)
	return nil
}

// FetchAndLock locks and returns up to batchSize events in StatusPending or StatusFailed whose NextRetryAt has passed.
func (s *MemoryStore) FetchAndLock(ctx context.Context, batchSize int) ([]*Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var batch []*Event

	for _, id := range s.order {
		e := s.events[id]
		if (e.Status == StatusPending || e.Status == StatusFailed) && (e.NextRetryAt.IsZero() || !now.Before(e.NextRetryAt)) {
			e.Status = StatusProcessing
			batch = append(batch, e)
			if len(batch) >= batchSize {
				break
			}
		}
	}

	return batch, nil
}

// MarkCompleted marks the event as completed.
func (s *MemoryStore) MarkCompleted(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.events[id]
	if !ok {
		return ErrEventNotFound
	}

	now := time.Now().UTC()
	e.Status = StatusCompleted
	e.ProcessedAt = &now
	return nil
}

// MarkFailed increments retry counter and sets next retry schedule.
func (s *MemoryStore) MarkFailed(ctx context.Context, id string, err error, nextRetryAfter time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.events[id]
	if !ok {
		return ErrEventNotFound
	}

	e.RetryCount++
	e.LastError = err.Error()
	e.Status = StatusFailed
	e.NextRetryAt = time.Now().UTC().Add(nextRetryAfter)
	return nil
}

// MarkDeadLetter moves an event to dead letter queue.
func (s *MemoryStore) MarkDeadLetter(ctx context.Context, id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.events[id]
	if !ok {
		return ErrEventNotFound
	}

	e.Status = StatusDeadLetter
	e.LastError = err.Error()
	return nil
}

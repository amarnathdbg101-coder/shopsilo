package auditlog

import (
	"context"
	"strings"
	"sync"
)

// MemoryStore is an in-memory, thread-safe implementation of Store.
// Perfect for local development, fast unit tests, and integration testing without database setup.
type MemoryStore struct {
	mu      sync.RWMutex
	entries []*Entry
}

// NewMemoryStore initializes an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: make([]*Entry, 0, 100),
	}
}

// SaveBatch appends audit entries to memory.
func (s *MemoryStore) SaveBatch(ctx context.Context, entries []*Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range entries {
		cloned := *e
		s.entries = append(s.entries, &cloned)
	}
	return nil
}

// Query filters in-memory entries matching filter options.
func (s *MemoryStore) Query(ctx context.Context, filter Filter) ([]*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Entry

	for i := len(s.entries) - 1; i >= 0; i-- { // Most recent first
		e := s.entries[i]

		if filter.Entity != "" && !strings.EqualFold(e.Entity, filter.Entity) {
			continue
		}
		if filter.EntityID != "" && e.EntityID != filter.EntityID {
			continue
		}
		if filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter.Action != "" && !strings.EqualFold(e.Action, filter.Action) {
			continue
		}

		result = append(result, e)

		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result, nil
}

// Close resets in-memory entries.
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	return nil
}

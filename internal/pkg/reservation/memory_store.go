package goreservation

import (
	"context"
	"sync"
	"time"
)

// MemoryStore provides a thread-safe in-memory reservation store.
type MemoryStore struct {
	mu           sync.RWMutex
	reservations map[string]*Reservation
}

// NewMemoryStore creates an initialized in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		reservations: make(map[string]*Reservation),
	}
}

// Save stores the reservation.
func (m *MemoryStore) Save(ctx context.Context, r *Reservation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Defensive copy
	copied := *r
	copied.Items = make([]Item, len(r.Items))
	copy(copied.Items, r.Items)

	m.reservations[r.ID] = &copied
	return nil
}

// Get retrieves the reservation by ID.
func (m *MemoryStore) Get(ctx context.Context, id string) (*Reservation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	r, exists := m.reservations[id]
	if !exists {
		return nil, ErrReservationNotFound
	}

	copied := *r
	copied.Items = make([]Item, len(r.Items))
	copy(copied.Items, r.Items)
	return &copied, nil
}

// UpdateStatus atomically updates status if current matches 'from'.
func (m *MemoryStore) UpdateStatus(ctx context.Context, id string, from Status, to Status, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.reservations[id]
	if !exists {
		return ErrReservationNotFound
	}

	if r.Status != from {
		return ErrReservationNotPending
	}

	r.Status = to
	if to == StatusConfirmed {
		r.ConfirmedAt = &timestamp
	} else if to == StatusReleased || to == StatusExpired {
		r.ReleasedAt = &timestamp
	}

	return nil
}

// Extend updates expiration timestamp.
func (m *MemoryStore) Extend(ctx context.Context, id string, newExpiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.reservations[id]
	if !exists {
		return ErrReservationNotFound
	}
	if r.Status != StatusPending {
		return ErrReservationNotPending
	}

	r.ExpiresAt = newExpiresAt
	return nil
}

// FetchExpired returns up to limit pending reservations where ExpiresAt <= asOf.
func (m *MemoryStore) FetchExpired(ctx context.Context, asOf time.Time, limit int) ([]*Reservation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	var results []*Reservation
	for _, r := range m.reservations {
		if r.Status == StatusPending && !r.ExpiresAt.After(asOf) {
			copied := *r
			copied.Items = make([]Item, len(r.Items))
			copy(copied.Items, r.Items)
			results = append(results, &copied)
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

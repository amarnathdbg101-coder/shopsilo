package goreservation

import (
	"context"
	"time"
)

// Store defines persistence contracts for reservations.
type Store interface {
	// Save creates or updates a reservation.
	Save(ctx context.Context, r *Reservation) error

	// Get retrieves a reservation by ID.
	Get(ctx context.Context, id string) (*Reservation, error)

	// UpdateStatus atomically transitions status from expected 'from' state to 'to' state.
	UpdateStatus(ctx context.Context, id string, from Status, to Status, timestamp time.Time) error

	// Extend updates the lease expiration time.
	Extend(ctx context.Context, id string, newExpiresAt time.Time) error

	// FetchExpired finds reservations in StatusPending where ExpiresAt <= asOf.
	FetchExpired(ctx context.Context, asOf time.Time, limit int) ([]*Reservation, error)
}

// InventoryManager abstracts stock locking and restoring operations.
type InventoryManager interface {
	// Reserve attempts to hold the required quantity for each item. Must return ErrInsufficientStock if unavailable.
	Reserve(ctx context.Context, items []Item) error

	// Commit permanently deducts the held stock upon purchase confirmation.
	Commit(ctx context.Context, items []Item) error

	// Release restores the held stock back to available inventory on cancel/expiry.
	Release(ctx context.Context, items []Item) error
}

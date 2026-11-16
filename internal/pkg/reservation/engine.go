package goreservation

import (
	"context"
	"sync"
	"time"
)

// HoldRequest defines parameters for acquiring an inventory lease.
type HoldRequest struct {
	ID       string            `json:"id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	ScopeID  string            `json:"scope_id,omitempty"`
	Items    []Item            `json:"items"`
	Duration time.Duration     `json:"duration"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Engine coordinates the two-phase inventory reservation and lease lock lifecycle.
type Engine struct {
	store     Store
	inventory InventoryManager
	mu        sync.Mutex
}

// NewEngine creates an initialized Engine.
func NewEngine(store Store, inventory InventoryManager) *Engine {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Engine{
		store:     store,
		inventory: inventory,
	}
}

// Hold attempts Phase 1: acquires inventory hold and creates a pending reservation with TTL.
func (e *Engine) Hold(ctx context.Context, req HoldRequest) (*Reservation, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyItems
	}
	if req.Duration <= 0 {
		return nil, ErrInvalidDuration
	}

	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return nil, ErrZeroOrNegativeQuantity
		}
	}

	id := req.ID
	if id == "" {
		id = GenerateID()
	}

	now := time.Now().UTC()
	res := &Reservation{
		ID:        id,
		UserID:    req.UserID,
		ScopeID:   req.ScopeID,
		Items:     req.Items,
		Status:    StatusPending,
		Metadata:  req.Metadata,
		CreatedAt: now,
		ExpiresAt: now.Add(req.Duration),
	}

	// 1. Hold stock in inventory
	if e.inventory != nil {
		if err := e.inventory.Reserve(ctx, req.Items); err != nil {
			return nil, err
		}
	}

	// 2. Persist reservation
	if err := e.store.Save(ctx, res); err != nil {
		// Rollback inventory hold on storage failure
		if e.inventory != nil {
			_ = e.inventory.Release(ctx, req.Items)
		}
		return nil, err
	}

	return res, nil
}

// Confirm executes Phase 2A: verifies active lease and permanently commits the inventory.
func (e *Engine) Confirm(ctx context.Context, id string) error {
	res, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if !res.IsActive(now) {
		return ErrReservationExpired
	}

	// 1. Commit inventory
	if e.inventory != nil {
		if err := e.inventory.Commit(ctx, res.Items); err != nil {
			return err
		}
	}

	// 2. Update status
	return e.store.UpdateStatus(ctx, id, StatusPending, StatusConfirmed, now)
}

// Release executes Phase 2B: manually cancels hold and restores inventory.
func (e *Engine) Release(ctx context.Context, id string) error {
	res, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}

	if res.Status != StatusPending {
		return ErrReservationNotPending
	}

	now := time.Now().UTC()

	// 1. Restore inventory
	if e.inventory != nil {
		if err := e.inventory.Release(ctx, res.Items); err != nil {
			return err
		}
	}

	// 2. Update status
	return e.store.UpdateStatus(ctx, id, StatusPending, StatusReleased, now)
}

// Extend elongates the reservation expiration timestamp before it lapses.
func (e *Engine) Extend(ctx context.Context, id string, additionalDuration time.Duration) error {
	if additionalDuration <= 0 {
		return ErrInvalidDuration
	}

	res, err := e.store.Get(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if !res.IsActive(now) {
		return ErrReservationExpired
	}

	newExpiry := res.ExpiresAt.Add(additionalDuration)
	return e.store.Extend(ctx, id, newExpiry)
}

// SweepExpired finds all elapsed pending leases, restores their inventory, and marks them expired.
func (e *Engine) SweepExpired(ctx context.Context, limit int) (int, error) {
	now := time.Now().UTC()
	expired, err := e.store.FetchExpired(ctx, now, limit)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, res := range expired {
		if e.inventory != nil {
			_ = e.inventory.Release(ctx, res.Items)
		}
		if err := e.store.UpdateStatus(ctx, res.ID, StatusPending, StatusExpired, now); err == nil {
			count++
		}
	}

	return count, nil
}

// StartAutoSweeper runs a background sweeper loop that periodically restores elapsed leases.
func (e *Engine) StartAutoSweeper(ctx context.Context, interval time.Duration) func() {
	if interval <= 0 {
		interval = 1 * time.Minute
	}

	stopCh := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				_, _ = e.SweepExpired(ctx, 100)
			}
		}
	}()

	return func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
}

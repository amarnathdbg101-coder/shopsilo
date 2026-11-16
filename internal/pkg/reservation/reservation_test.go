package goreservation_test

import (
	"context"
	"sync"
	"testing"
	"time"

	goreservation "shopMe/internal/pkg/reservation"
)

type mockInventory struct {
	mu        sync.Mutex
	available map[string]int
	reserved  map[string]int
}

func newMockInventory(stock map[string]int) *mockInventory {
	return &mockInventory{
		available: stock,
		reserved:  make(map[string]int),
	}
}

func (m *mockInventory) Reserve(ctx context.Context, items []goreservation.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify all items have enough stock
	for _, it := range items {
		if m.available[it.ResourceID] < it.Quantity {
			return goreservation.ErrInsufficientStock
		}
	}

	for _, it := range items {
		m.available[it.ResourceID] -= it.Quantity
		m.reserved[it.ResourceID] += it.Quantity
	}
	return nil
}

func (m *mockInventory) Commit(ctx context.Context, items []goreservation.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, it := range items {
		m.reserved[it.ResourceID] -= it.Quantity
	}
	return nil
}

func (m *mockInventory) Release(ctx context.Context, items []goreservation.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, it := range items {
		m.reserved[it.ResourceID] -= it.Quantity
		m.available[it.ResourceID] += it.Quantity
	}
	return nil
}

func TestEngine_SuccessfulHoldAndConfirm(t *testing.T) {
	inv := newMockInventory(map[string]int{"item_rice": 10})
	engine := goreservation.NewEngine(nil, inv)
	ctx := context.Background()

	// 1. Acquire Hold (Phase 1)
	res, err := engine.Hold(ctx, goreservation.HoldRequest{
		UserID:   "usr_101",
		ScopeID:  "shop_999",
		Duration: 15 * time.Minute,
		Items: []goreservation.Item{
			{ResourceID: "item_rice", Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("failed to hold inventory: %v", err)
	}

	if inv.available["item_rice"] != 7 || inv.reserved["item_rice"] != 3 {
		t.Fatalf("expected available=7, reserved=3, got av=%d res=%d", inv.available["item_rice"], inv.reserved["item_rice"])
	}

	// 2. Confirm (Phase 2A)
	if err := engine.Confirm(ctx, res.ID); err != nil {
		t.Fatalf("failed to confirm reservation: %v", err)
	}

	if inv.reserved["item_rice"] != 0 || inv.available["item_rice"] != 7 {
		t.Fatalf("expected reserved=0, available=7 after commit, got av=%d res=%d", inv.available["item_rice"], inv.reserved["item_rice"])
	}
}

func TestEngine_HoldAndManualRelease(t *testing.T) {
	inv := newMockInventory(map[string]int{"item_oil": 5})
	engine := goreservation.NewEngine(nil, inv)
	ctx := context.Background()

	res, err := engine.Hold(ctx, goreservation.HoldRequest{
		UserID:   "usr_102",
		Duration: 10 * time.Minute,
		Items: []goreservation.Item{
			{ResourceID: "item_oil", Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("failed to hold: %v", err)
	}

	if inv.available["item_oil"] != 3 {
		t.Fatalf("expected 3 available, got %d", inv.available["item_oil"])
	}

	// Manual cancellation / release
	if err := engine.Release(ctx, res.ID); err != nil {
		t.Fatalf("failed to release hold: %v", err)
	}

	// Available stock restored
	if inv.available["item_oil"] != 5 || inv.reserved["item_oil"] != 0 {
		t.Fatalf("expected restored stock=5, got av=%d res=%d", inv.available["item_oil"], inv.reserved["item_oil"])
	}
}

func TestEngine_InsufficientStock(t *testing.T) {
	inv := newMockInventory(map[string]int{"item_milk": 2})
	engine := goreservation.NewEngine(nil, inv)
	ctx := context.Background()

	_, err := engine.Hold(ctx, goreservation.HoldRequest{
		UserID:   "usr_103",
		Duration: 5 * time.Minute,
		Items: []goreservation.Item{
			{ResourceID: "item_milk", Quantity: 5}, // Demanding 5 when only 2 available
		},
	})

	if err != goreservation.ErrInsufficientStock {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestEngine_SweepExpired(t *testing.T) {
	inv := newMockInventory(map[string]int{"item_atta": 8})
	engine := goreservation.NewEngine(nil, inv)
	ctx := context.Background()

	// Short lease of 20ms
	res, err := engine.Hold(ctx, goreservation.HoldRequest{
		UserID:   "usr_104",
		Duration: 20 * time.Millisecond,
		Items: []goreservation.Item{
			{ResourceID: "item_atta", Quantity: 4},
		},
	})
	if err != nil {
		t.Fatalf("failed to hold: %v", err)
	}

	// Wait for expiration
	time.Sleep(35 * time.Millisecond)

	// Confirm should fail
	if err := engine.Confirm(ctx, res.ID); err != goreservation.ErrReservationExpired {
		t.Fatalf("expected ErrReservationExpired, got %v", err)
	}

	// Sweep expired leases
	swept, err := engine.SweepExpired(ctx, 10)
	if err != nil {
		t.Fatalf("failed to sweep: %v", err)
	}
	if swept != 1 {
		t.Fatalf("expected exactly 1 reservation swept, got %d", swept)
	}

	// Verify inventory automatically restored
	if inv.available["item_atta"] != 8 || inv.reserved["item_atta"] != 0 {
		t.Fatalf("expected restored stock 8, got av=%d res=%d", inv.available["item_atta"], inv.reserved["item_atta"])
	}
}

func TestEngine_ExtendHold(t *testing.T) {
	inv := newMockInventory(map[string]int{"item_biscuit": 20})
	engine := goreservation.NewEngine(nil, inv)
	ctx := context.Background()

	res, _ := engine.Hold(ctx, goreservation.HoldRequest{
		UserID:   "usr_105",
		Duration: 50 * time.Millisecond,
		Items:    []goreservation.Item{{ResourceID: "item_biscuit", Quantity: 2}},
	})

	// Extend hold
	if err := engine.Extend(ctx, res.ID, 300*time.Millisecond); err != nil {
		t.Fatalf("failed to extend: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	// Should still be active because we extended by 300ms
	if err := engine.Confirm(ctx, res.ID); err != nil {
		t.Fatalf("expected confirm to succeed after extension, got %v", err)
	}
}

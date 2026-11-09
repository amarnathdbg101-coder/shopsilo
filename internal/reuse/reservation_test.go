package reuse_test

import (
	"context"
	"testing"
	"time"

	"shopMe/internal/reuse"
)

type dummyInventory struct {
	stock int
}

func (d *dummyInventory) Reserve(ctx context.Context, items []reuse.ReservationItem) error {
	for _, it := range items {
		if d.stock < it.Quantity {
			return reuse.ErrInsufficientStock
		}
		d.stock -= it.Quantity
	}
	return nil
}

func (d *dummyInventory) Commit(ctx context.Context, items []reuse.ReservationItem) error {
	return nil
}

func (d *dummyInventory) Release(ctx context.Context, items []reuse.ReservationItem) error {
	for _, it := range items {
		d.stock += it.Quantity
	}
	return nil
}

func TestReuse_ReservationEngine(t *testing.T) {
	inv := &dummyInventory{stock: 10}
	store := reuse.NewMemoryReservationStore()
	engine := reuse.NewReservationEngine(store, inv)

	ctx := context.Background()

	// 1. Hold
	res, err := engine.Hold(ctx, reuse.HoldRequest{
		UserID:   "usr_abc",
		Duration: 100 * time.Millisecond,
		Items: []reuse.ReservationItem{
			{ResourceID: "prod_atta", Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("failed to hold reservation: %v", err)
	}

	if inv.stock != 7 {
		t.Fatalf("expected stock 7, got %d", inv.stock)
	}

	// 2. Confirm
	if err := engine.Confirm(ctx, res.ID); err != nil {
		t.Fatalf("failed to confirm: %v", err)
	}

	state, _ := store.Get(ctx, res.ID)
	if state.Status != reuse.ReservationStatusConfirmed {
		t.Fatalf("expected CONFIRMED status, got %s", state.Status)
	}
}

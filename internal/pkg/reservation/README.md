# go-reservation

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-reservation.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-reservation)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-ready, zero-dependency Two-Phase Inventory Reservation & Lease Lock engine with auto-expiration sweepers.

---

## Features
- **Eliminates Overselling**: Atomically locks temporary inventory holds across concurrent shoppers.
- **Two-Phase Lifecycle**: Clean state machine (`PENDING` -> `CONFIRMED` / `RELEASED` / `EXPIRED`).
- **Auto-Expiration Sweepers**: Background ticker reclaims abandoned shopping carts and expired store pickup reservations.
- **Lease Extensions**: Safely extend hold time while customer is on the payment screen.
- **Pluggable Architecture**: Implements `InventoryManager` and `Store` interfaces for easy integration with PostgreSQL, Redis, or in-memory stores.
- **Zero Third-Party Dependencies**: Pure Go standard library (`context`, `sync`, `time`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-reservation@v1.0.0
```

---

## Quickstart

### 1. Implement Inventory Manager

```go
type MyInventory struct { /* ... */ }

func (m *MyInventory) Reserve(ctx context.Context, items []goreservation.Item) error {
    // Decrement available, increment reserved in PostgreSQL
    return nil
}

func (m *MyInventory) Commit(ctx context.Context, items []goreservation.Item) error {
    // Decrement reserved permanently
    return nil
}

func (m *MyInventory) Release(ctx context.Context, items []goreservation.Item) error {
    // Decrement reserved, restore available
    return nil
}
```

### 2. Initialize Engine & Background Sweeper

```go
engine := goreservation.NewEngine(nil, &MyInventory{})

// Start background sweeper checking every 30s
stopSweeper := engine.StartAutoSweeper(context.Background(), 30*time.Second)
defer stopSweeper()
```

### 3. Hold and Confirm

```go
// Phase 1: Hold 2 units for 15 minutes
res, err := engine.Hold(ctx, goreservation.HoldRequest{
    UserID:   "user_123",
    Duration: 15 * time.Minute,
    Items: []goreservation.Item{
        {ResourceID: "prod_rice_5kg", Quantity: 2},
    },
})

// Phase 2A: Customer completes payment
err = engine.Confirm(ctx, res.ID)

// Or Phase 2B: Customer cancels
err = engine.Release(ctx, res.ID)
```

---

## License
MIT License.

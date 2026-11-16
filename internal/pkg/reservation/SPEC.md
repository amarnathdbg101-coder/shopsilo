# Technical Specification: go-reservation

## 1. Domain & Purpose
`go-reservation` is a high-concurrency, zero-dependency Two-Phase Inventory Reservation & Lease Lock engine for Go. It eliminates overselling, handles temporary cart/pickup holds with automatic time-to-live (TTL), provides hold extensions, and guarantees atomic stock restoration via background lease expiration sweepers.

## 2. Architecture & Design Principles
- **Two-Phase Commit Protocol**:
  - `Phase 1 (Hold)`: Validates available inventory, places an atomic hold, and issues a pending lease with an expiration deadline.
  - `Phase 2A (Confirm)`: Finalizes checkout/pickup, permanently committing held units.
  - `Phase 2B (Release)`: Manually cancels the reservation and restores available inventory.
  - `Phase 2C (Sweep)`: Periodic sweeper detects elapsed pending leases, transitions them to `EXPIRED`, and auto-restores stock.
- **Race Condition Prevention**: Enforces atomic state transitions (`PENDING -> CONFIRMED`, `PENDING -> RELEASED`, `PENDING -> EXPIRED`).
- **Pluggable Inventory Interface**: Decouples from specific database ORMs via the `InventoryManager` interface (`Reserve`, `Commit`, `Release`).
- **Zero Third-Party Dependencies**: Pure standard library (`context`, `sync`, `time`, `errors`, `crypto/rand`).

## 3. API Surface
- `type Status string (StatusPending, StatusConfirmed, StatusReleased, StatusExpired)`
- `type Item struct { ResourceID string, Quantity int }`
- `type Reservation struct`
- `type HoldRequest struct`
- `type Store interface`
- `type MemoryStore struct`
- `type InventoryManager interface`
- `type Engine struct`
- `func NewEngine(store Store, inventory InventoryManager) *Engine`
- `func (e *Engine) Hold(ctx context.Context, req HoldRequest) (*Reservation, error)`
- `func (e *Engine) Confirm(ctx context.Context, id string) error`
- `func (e *Engine) Release(ctx context.Context, id string) error`
- `func (e *Engine) Extend(ctx context.Context, id string, additionalDuration time.Duration) error`
- `func (e *Engine) SweepExpired(ctx context.Context, limit int) (int, error)`
- `func (e *Engine) StartAutoSweeper(ctx context.Context, interval time.Duration) func()`

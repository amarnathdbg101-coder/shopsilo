package reuse

import (
	goreservation "github.com/amarnathdbg101-coder/go/go-reservation"
)

// Re-export core reservation types for clean domain use.
type Reservation = goreservation.Reservation
type ReservationItem = goreservation.Item
type ReservationStatus = goreservation.Status
type HoldRequest = goreservation.HoldRequest
type ReservationStore = goreservation.Store
type InventoryManager = goreservation.InventoryManager
type ReservationEngine = goreservation.Engine

const (
	ReservationStatusPending   = goreservation.StatusPending
	ReservationStatusConfirmed = goreservation.StatusConfirmed
	ReservationStatusReleased  = goreservation.StatusReleased
	ReservationStatusExpired   = goreservation.StatusExpired
)

var (
	ErrInsufficientStock     = goreservation.ErrInsufficientStock
	ErrReservationNotFound   = goreservation.ErrReservationNotFound
	ErrReservationExpired    = goreservation.ErrReservationExpired
	ErrReservationNotPending = goreservation.ErrReservationNotPending
)

// NewReservationEngine creates an instance of the Two-Phase inventory lease coordinator.
func NewReservationEngine(store ReservationStore, inventory InventoryManager) *ReservationEngine {
	return goreservation.NewEngine(store, inventory)
}

// NewMemoryReservationStore creates an in-memory reservation storage for testing or localized microservices.
func NewMemoryReservationStore() *goreservation.MemoryStore {
	return goreservation.NewMemoryStore()
}

// GenerateReservationID creates a cryptographically unique hex token.
func GenerateReservationID() string {
	return goreservation.GenerateID()
}

package goreservation

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Status represents the two-phase lifecycle of an inventory lease.
type Status string

const (
	// StatusPending indicates active hold on inventory awaiting checkout or store pickup.
	StatusPending Status = "PENDING"

	// StatusConfirmed indicates customer completed payment/pickup; inventory hold is finalized.
	StatusConfirmed Status = "CONFIRMED"

	// StatusReleased indicates user/shop cancelled hold; held inventory is restored.
	StatusReleased Status = "RELEASED"

	// StatusExpired indicates hold TTL elapsed without confirmation; swept and restored.
	StatusExpired Status = "EXPIRED"
)

// Item represents a held inventory unit.
type Item struct {
	ResourceID string `json:"resource_id"`
	Quantity   int    `json:"quantity"`
}

// Reservation represents an atomic inventory hold lock with time-to-live.
type Reservation struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id,omitempty"`
	ScopeID     string            `json:"scope_id,omitempty"` // E.g. ShopID, WarehouseID
	Items       []Item            `json:"items"`
	Status      Status            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
	ConfirmedAt *time.Time        `json:"confirmed_at,omitempty"`
	ReleasedAt  *time.Time        `json:"released_at,omitempty"`
}

// GenerateID produces a random hex identifier.
func GenerateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// IsActive returns true if the reservation is pending and has not yet expired.
func (r *Reservation) IsActive(now time.Time) bool {
	if r.Status != StatusPending {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Before(r.ExpiresAt)
}

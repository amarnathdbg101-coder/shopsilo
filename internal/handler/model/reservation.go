// Package model handle database struct.
package model

import "time"

type Reservation struct {
	ID                     string     `json:"id"`
	ReservationNumber      string     `json:"reservation_number"`
	UserID                 string     `json:"user_id"`
	ShopID                 string     `json:"shop_id"`
	ProductID              string     `json:"product_id"`
	Quantity               int        `json:"quantity"`
	PickupCode             string     `json:"pickup_code"`
	Status                 string     `json:"status"` // 'active', 'completed', 'cancelled', 'expired'
	ExpiresAt              time.Time  `json:"expires_at"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
	Notes                  string     `json:"notes,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`

	// Enriched details
	Shop                   *Shop      `json:"shop,omitempty"`
	Product                *Product   `json:"product,omitempty"`
	TimeRemainingMinutes   int        `json:"time_remaining_minutes"`
}

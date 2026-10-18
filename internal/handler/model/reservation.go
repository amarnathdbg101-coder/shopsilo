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
	CustomerName           string     `json:"customer_name,omitempty"`
	CustomerPhone          string     `json:"customer_phone,omitempty"`
	Shop                   *Shop      `json:"shop,omitempty"`
	Product                *Product   `json:"product,omitempty"`
	User                   *User      `json:"user,omitempty"`
	TimeRemainingMinutes   int        `json:"time_remaining_minutes"`

	// Top-level direct access fields for seamless cross-client hydration
	ProductName            string     `json:"product_name,omitempty"`
	ProductImage           string     `json:"product_image,omitempty"`
	ProductImages          []string   `json:"product_images,omitempty"`
	ProductPrice           float64    `json:"product_price,omitempty"`
	ProductSKU             string     `json:"product_sku,omitempty"`
	TotalAmount            float64    `json:"total_amount,omitempty"`
}

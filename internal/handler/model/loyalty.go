// Package model handle database struct.
package model

import "time"

type ProductReturn struct {
	ID            string    `json:"id"`
	ShopID        string    `json:"shop_id"`
	UserID        *string   `json:"user_id,omitempty"`
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name,omitempty"`
	ReservationID *string   `json:"reservation_id,omitempty"`
	Quantity      int       `json:"quantity"`
	RefundAmount  float64   `json:"refund_amount"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type StoreOffer struct {
	ID                string     `json:"id"`
	ShopID            string     `json:"shop_id"`
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	DiscountText      string     `json:"discount_text"`
	MinPointsRequired int        `json:"min_points_required"`
	IsActive          bool       `json:"is_active"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	IsUnlocked        bool       `json:"is_unlocked"` // true if customer has enough points
	ShopName          string     `json:"shop_name,omitempty"`
	ShopSlug          string     `json:"shop_slug,omitempty"`
	ShopCategory      string     `json:"shop_category,omitempty"`
	ShopLogoURL       *string    `json:"shop_logo_url,omitempty"`
	ShopLatitude      *float64   `json:"shop_latitude,omitempty"`
	ShopLongitude     *float64   `json:"shop_longitude,omitempty"`
}

type LoyaltySummary struct {
	UserID               string `json:"user_id"`
	UserName             string `json:"user_name"`
	Points               int    `json:"points"`
	Tier                 string `json:"tier"` // Bronze (0-99), Silver (100-299), Gold VIP (300+)
	NextTierPointsNeeded int    `json:"next_tier_points_needed"`
}

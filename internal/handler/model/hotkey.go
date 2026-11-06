// Package model defines database entity models.
package model

import "time"

// ShopHotkey represents a 1-tap fast-grid button on the POS counter billing screen (Zero-scan rush hour).
type ShopHotkey struct {
	ID          string    `json:"id"`
	ShopID      string    `json:"shop_id"`
	ProductID   string    `json:"product_id"`
	VariantID   *string   `json:"variant_id,omitempty"`
	DisplayName string    `json:"display_name"`
	ColorHex    string    `json:"color_hex"`
	SortOrder   int       `json:"sort_order"`
	TimeSlot    string    `json:"time_slot"` // "ALL_DAY", "MORNING", "EVENING"
	IsActive    bool      `json:"is_active"`
	Price       float64   `json:"price,omitempty"`       // Populated dynamically from product/variant
	CostPrice   float64   `json:"cost_price,omitempty"`  // Populated dynamically
	FloorPrice  float64   `json:"floor_price,omitempty"` // Populated dynamically
	SKU         string    `json:"sku,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}


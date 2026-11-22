// Package model handles database structs.
package model

import "time"

// ProcurementItem represents a Mandi purchase checklist item for the merchant shop.
type ProcurementItem struct {
	ID             string    `json:"id"`
	ShopID         string    `json:"shop_id"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	Qty            string    `json:"qty"`
	EstimatedPrice float64   `json:"estimated_price"`
	ActualPrice    float64   `json:"actual_price"`
	DealerName     string    `json:"dealer_name"`
	DealerPhone    string    `json:"dealer_phone"`
	Notes          string    `json:"notes"`
	IsCompleted    bool      `json:"is_completed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Wholesaler represents a registered permanent wholesale supplier/dealer for the merchant shop.
type Wholesaler struct {
	ID        string    `json:"id"`
	ShopID    string    `json:"shop_id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	GSTIN     string    `json:"gstin"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Package model contains core database entity structs.
package model

import "time"

// BatchStatus represents lifecycle state of an inventory lot.
type BatchStatus string

const (
	BatchStatusActive     BatchStatus = "ACTIVE"
	BatchStatusNearExpiry BatchStatus = "NEAR_EXPIRY"
	BatchStatusExpired    BatchStatus = "EXPIRED"
	BatchStatusSoldOut    BatchStatus = "SOLD_OUT"
)

// ProductBatch represents a dated procurement lot of a product with expiry tracking.
type ProductBatch struct {
	ID                   string      `json:"id"`
	ShopID               string      `json:"shop_id"`
	ProductID            string      `json:"product_id"`
	ProductName          string      `json:"product_name,omitempty"`
	VariantID            *string     `json:"variant_id,omitempty"`
	BatchNumber          string      `json:"batch_number"`
	MfgDate              *time.Time  `json:"mfg_date,omitempty"`
	ExpiryDate           time.Time   `json:"expiry_date"`
	Quantity             int         `json:"quantity"`
	CostPrice            float64     `json:"cost_price"`
	OriginalSellingPrice float64     `json:"original_selling_price"`
	CurrentSellingPrice  float64     `json:"current_selling_price"`
	MarkdownPercent      float64     `json:"markdown_percent"`
	IsMarkdownApplied    bool        `json:"is_markdown_applied"`
	Status               BatchStatus `json:"status"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}


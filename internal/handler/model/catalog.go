package model

import "time"

// MasterCatalogProduct represents an enterprise master barcode catalog entry
// for instant 0ms FMCG/Retail auto-fill across all merchants.
type MasterCatalogProduct struct {
	ID           string    `json:"id"`
	Barcode      string    `json:"barcode"`
	Name         string    `json:"name"`
	Brand        string    `json:"brand"`
	Category     string    `json:"category"`
	Unit         string    `json:"unit"`
	MRP          float64   `json:"mrp"`
	SellingPrice float64   `json:"selling_price"`
	Description  string    `json:"description,omitempty"`
	ImageURL     string    `json:"image_url,omitempty"`
	Source       string    `json:"source"` // 'admin', 'crowdsourced', 'gs1'
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

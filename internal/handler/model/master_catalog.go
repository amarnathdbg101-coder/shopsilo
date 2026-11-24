package model

import "time"

// MasterCatalogProduct represents a standardized, verified product in the global catalog.
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
	ImageURL     string    `json:"image_url"`
	HSNCode      string    `json:"hsn_code,omitempty"`
	GSTRate      float64   `json:"gst_rate"`
	Source       string    `json:"source"`
	TrustScore   int       `json:"trust_score"`
	IsVerified   bool      `json:"is_verified"`
	ScanCount    int       `json:"scan_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

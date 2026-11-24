package dto

import "shopMe/internal/handler/model"

// CatalogLookupResponse is returned when a merchant scans a barcode in their POS or Inventory.
type CatalogLookupResponse struct {
	Barcode            string                      `json:"barcode"`
	CanonicalBarcode   string                      `json:"canonical_barcode,omitempty"`
	IsValidChecksum    bool                        `json:"is_valid_checksum"`
	ExistsInShop       bool                        `json:"exists_in_shop"`
	ShopProduct        *model.Product              `json:"shop_product,omitempty"`
	MasterFound        bool                        `json:"master_found"`
	MasterProduct      *model.MasterCatalogProduct `json:"master_product,omitempty"`
	SuggestedMarginPct float64                     `json:"suggested_margin_pct,omitempty"`
}

// MasterCatalogProductInput represents a single product input for admin bulk seeding.
type MasterCatalogProductInput struct {
	Barcode      string  `json:"barcode"`
	Name         string  `json:"name"`
	Brand        string  `json:"brand"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	MRP          float64 `json:"mrp"`
	SellingPrice float64 `json:"selling_price"`
	ImageURL     string  `json:"image_url"`
	HSNCode      string  `json:"hsn_code"`
	GSTRate      float64 `json:"gst_rate"`
	Description  string  `json:"description"`
}

// MasterCatalogBulkSeedRequest accepts up to thousands of products in a single batch.
type MasterCatalogBulkSeedRequest struct {
	Products []MasterCatalogProductInput `json:"products"`
}

// MasterCatalogBulkSeedResponse reports ingestion statistics.
type MasterCatalogBulkSeedResponse struct {
	TotalProcessed int      `json:"total_processed"`
	Inserted       int      `json:"inserted"`
	Updated        int      `json:"updated"`
	Errors         []string `json:"errors,omitempty"`
}

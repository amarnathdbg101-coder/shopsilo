package dto

// CatalogLookupResponse is the standardized payload returned to Flutter
type CatalogLookupResponse struct {
	Barcode      string  `json:"barcode"`
	Name         string  `json:"name"`
	Brand        string  `json:"brand"`
	Category     string  `json:"category"`
	Unit         string  `json:"unit"`
	MRP          float64 `json:"mrp"`
	SellingPrice float64 `json:"selling_price"`
	Description  string  `json:"description,omitempty"`
	ImageURL     string  `json:"image_url,omitempty"`
	Source       string  `json:"source"`
	IsFound      bool    `json:"is_found"`
}

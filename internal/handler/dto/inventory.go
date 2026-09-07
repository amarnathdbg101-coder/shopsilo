// Package dto handle request and response struct.
package dto

type AdjustStockRequest struct {
	ProductID         string `json:"product_id" validate:"required,uuid"`
	Adjustment        int    `json:"adjustment" validate:"required"` // e.g. +50 or -5
	LowStockThreshold *int   `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0,lte=1000"`
	Notes             string `json:"notes,omitempty" validate:"omitempty,max=200"`
}

type LowStockProduct struct {
	ProductID           string  `json:"product_id"`
	Name                string  `json:"name"`
	SKU                 string  `json:"sku"`
	CurrentStock        int     `json:"current_stock"`
	ReservedStock       int     `json:"reserved_stock"`
	AvailableStock      int     `json:"available_stock"`
	LowStockThreshold   int     `json:"low_stock_threshold"`
	Price               float64 `json:"price"`
	SuggestedReorderQty int     `json:"suggested_reorder_qty"`
}

type LowStockResponse struct {
	TotalLowStockItems int                `json:"total_low_stock_items"`
	Items              []*LowStockProduct `json:"items"`
}

package dto

import "time"

// CustomerProductResponseDTO provides a clean, public customer-facing product payload.
// Explicitly omits internal wholesale cost prices, supplier info, and reorder logs.
type CustomerProductResponseDTO struct {
	ID                 string                 `json:"id"`
	ShopID             string                 `json:"shop_id"`
	ShopName           string                 `json:"shop_name,omitempty"`
	Name               string                 `json:"name"`
	Slug               string                 `json:"slug"`
	Description        string                 `json:"description,omitempty"`
	SellingPrice       float64                `json:"selling_price"`
	ComparePrice       float64                `json:"compare_price,omitempty"`
	DiscountPercentage float64                `json:"discount_percentage,omitempty"`
	MRP                float64                `json:"mrp,omitempty"`
	CategoryID         string                 `json:"category_id,omitempty"`
	CategoryName       string                 `json:"category_name,omitempty"`
	Images             []string               `json:"images,omitempty"`
	IsInStock          bool                   `json:"is_in_stock"`
	StockBadge         string                 `json:"stock_badge"` // "In Stock", "Only Few Left", "Out of Stock"
	DistanceKm         float64                `json:"distance_km,omitempty"`
	AllowBargain       bool                   `json:"allow_bargain"`
	Rating             float64                `json:"rating,omitempty"`
	ReviewCount        int                    `json:"review_count,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// MerchantProductResponseDTO provides full inventory, profit margin %, cost price, and SKU details for shop owners & cashiers.
type MerchantProductResponseDTO struct {
	ID                 string                 `json:"id"`
	ShopID             string                 `json:"shop_id"`
	Name               string                 `json:"name"`
	Slug               string                 `json:"slug"`
	Description        string                 `json:"description,omitempty"`
	SKU                string                 `json:"sku"`
	CostPrice          float64                `json:"cost_price"`
	SellingPrice       float64                `json:"selling_price"`
	FloorPrice         float64                `json:"floor_price"`
	ComparePrice       float64                `json:"compare_price,omitempty"`
	MRP                float64                `json:"mrp,omitempty"`
	ProfitMarginAmount float64                `json:"profit_margin_amount"`
	ProfitMarginPercent float64               `json:"profit_margin_percent"`
	CategoryID         string                 `json:"category_id,omitempty"`
	CategoryName       string                 `json:"category_name,omitempty"`
	ExactStockQuantity int                    `json:"exact_stock_quantity"`
	MinStock           int                    `json:"min_stock"`
	LowStockThreshold  int                    `json:"low_stock_threshold"`
	IsLowStock         bool                   `json:"is_low_stock"`
	Images             []string               `json:"images,omitempty"`
	Weight             float64                `json:"weight,omitempty"`
	IsActive           bool                   `json:"is_active"`
	IsFeatured         bool                   `json:"is_featured"`
	AllowBargain       bool                   `json:"allow_bargain"`
	Tags               []string               `json:"tags,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type CustomerProductPaginationResponse struct {
	Products   []*CustomerProductResponseDTO `json:"products"`
	TotalCount int                           `json:"total_count"`
	Page       int                           `json:"page"`
	Limit      int                           `json:"limit"`
	TotalPages int                           `json:"total_pages"`
}

type MerchantProductPaginationResponse struct {
	Products   []*MerchantProductResponseDTO `json:"products"`
	TotalCount int                           `json:"total_count"`
	Page       int                           `json:"page"`
	Limit      int                           `json:"limit"`
	TotalPages int                           `json:"total_pages"`
}

// Package dto handler request and response struct.
package dto

import (
	"shopMe/internal/handler/model"
)

type CreateProductRequest struct {
	Name          string   `json:"name" validate:"required,min=2,max=200"`
	Description   string   `json:"description,omitempty"`
	SKU           string   `json:"sku" validate:"required,min=2,max=100"`
	Price         float64  `json:"price" validate:"required,gt=0"`
	CostPrice     float64  `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice  float64  `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	MRP           float64  `json:"mrp,omitempty" validate:"omitempty,gte=0"`
	CategoryID    string   `json:"category_id" validate:"required"`
	StockQuantity     int                    `json:"stock_quantity" validate:"gte=0"`
	MinStock          *int                   `json:"min_stock,omitempty" validate:"omitempty,gte=0"` // Shop owner configured minimum stock threshold (default 1)
	LowStockThreshold *int                   `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0"`
	Images            []string               `json:"images,omitempty" validate:"omitempty,max=10"` // Max 4 images professional standard
	Weight            float64                `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive          bool                   `json:"is_active"`
	IsFeatured        bool                   `json:"is_featured"`
	IsPricePublic     *bool                  `json:"is_price_public,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"` // Brand, Model, Size, Color, Gender, Season, etc.
}

type UpdateProductRequest struct {
	Name              *string                 `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description       *string                 `json:"description,omitempty"`
	SKU               *string                 `json:"sku,omitempty" validate:"omitempty,min=2,max=100"`
	Price             *float64                `json:"price,omitempty" validate:"omitempty,gt=0"`
	CostPrice         *float64                `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice      *float64                `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	MRP               *float64                `json:"mrp,omitempty" validate:"omitempty,gte=0"`
	CategoryID        *string                 `json:"category_id,omitempty"`
	StockQuantity     *int                    `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	MinStock          *int                    `json:"min_stock,omitempty" validate:"omitempty,gte=0"`
	LowStockThreshold *int                    `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0"`
	Images            *[]string               `json:"images,omitempty" validate:"omitempty,max=10"` // Max 4 images
	Weight            *float64                `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive          *bool                   `json:"is_active,omitempty"`
	IsFeatured        *bool                   `json:"is_featured,omitempty"`
	IsPricePublic     *bool                   `json:"is_price_public,omitempty"`
	Tags              *[]string               `json:"tags,omitempty"`
	Attributes        *map[string]interface{} `json:"attributes,omitempty"` // Dynamic JSONB attributes
}

type ProductFilter struct {
	Search     string   `json:"search,omitempty"`
	Barcode    string   `json:"barcode,omitempty"`
	CategoryID string   `json:"category_id,omitempty"`
	ShopID     string   `json:"shop_id,omitempty"`
	MinPrice   float64  `json:"min_price,omitempty"`
	MaxPrice   float64  `json:"max_price,omitempty"`
	IsFeatured *bool    `json:"is_featured,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"` // price_asc, price_desc, newest, stock_desc, oldest
	Page       int      `json:"page,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Lat        *float64 `json:"lat,omitempty"`
	Lng        *float64 `json:"lng,omitempty"`
	RadiusKm   *float64 `json:"radius_km,omitempty"`
	City       string   `json:"city,omitempty"`
	Size       string   `json:"size,omitempty"`
	Color      string   `json:"color,omitempty"`
}

type ProductPaginationResponse struct {
	Products   []*model.Product `json:"products"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

type NearbyProductItem struct {
	ID                string                 `json:"id"`
	ProductID         string                 `json:"product_id"`
	Name              string                 `json:"name"`
	Slug              string                 `json:"slug"`
	Price             float64                `json:"price"`
	ComparePrice      float64                `json:"compare_price,omitempty"`
	IsPricePublic     bool                   `json:"is_price_public"`
	Images            []string               `json:"images"`
	CategoryID        string                 `json:"category_id,omitempty"`
	CategoryName      string                 `json:"category_name,omitempty"`
	CategorySlug      string                 `json:"category_slug,omitempty"`
	SKU               string                 `json:"sku,omitempty"`
	AvailableQuantity int                    `json:"available_quantity"`
	IsInStock         bool                   `json:"is_in_stock"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"`
	ShopID            string                 `json:"shop_id"`
	ShopName          string                 `json:"shop_name"`
	ShopSlug          string                 `json:"shop_slug"`
	ShopAddress       string                 `json:"shop_address"`
	ShopPhone         string                 `json:"shop_phone"`
	ShopCity          string                 `json:"shop_city,omitempty"`
	ShopLatitude      float64                `json:"shop_latitude,omitempty"`
	ShopLongitude     float64                `json:"shop_longitude,omitempty"`
	DistanceKm        float64                `json:"distance_km"`
	IsOpen            bool                   `json:"is_open"`
	IsCurrentlyOpen   bool                   `json:"is_currently_open"`
}

type NearbyProductResponse struct {
	Products   []*NearbyProductItem `json:"products"`
	TotalCount int                  `json:"total_count"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
	RadiusKm   float64              `json:"radius_km"`
}

// ApplyClearanceMarkdownRequest is used to markdown dead/stagnant stock with custom discount and broadcast.
type ApplyClearanceMarkdownRequest struct {
	DiscountPercentage float64 `json:"discount_percentage" validate:"required,gt=0,lt=100"`
	CustomMessage      string  `json:"custom_message,omitempty" validate:"omitempty,max=255"`
}

// ClearanceMarkdownResponse returns the discounted product details and ready-to-broadcast WhatsApp copy.
type ClearanceMarkdownResponse struct {
	ProductID          string   `json:"product_id"`
	Name               string   `json:"name"`
	OriginalPrice      float64  `json:"original_price"`
	DiscountedPrice    float64  `json:"discounted_price"`
	DiscountPercentage float64  `json:"discount_percentage"`
	StockQuantity      int      `json:"stock_quantity"`
	Tags               []string `json:"tags"`
	BroadcastMessage   string   `json:"broadcast_message"`
	WhatsAppShareURL   string   `json:"whatsapp_share_url"`
}

type BulkImportProductItem struct {
	Name          string  `json:"name"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode,omitempty"`
	Brand         string  `json:"brand,omitempty"`
	ImageURL      string  `json:"image_url,omitempty"`
	HSNCode       string  `json:"hsn_code,omitempty"`
	GSTRate       float64 `json:"gst_rate,omitempty"`
	Price         float64 `json:"price"`
	CostPrice     float64 `json:"cost_price,omitempty"`
	ComparePrice  float64 `json:"compare_price,omitempty"`
	StockQuantity int     `json:"stock_quantity"`
	MinStock      int     `json:"min_stock,omitempty"`
	CategoryName  string  `json:"category_name,omitempty"`
	CategoryID    string  `json:"category_id,omitempty"`
	Unit          string  `json:"unit,omitempty"`
	Description   string  `json:"description,omitempty"`
}

type BulkImportRequest struct {
	Items          []BulkImportProductItem `json:"items"`
	UpdateExisting bool                    `json:"update_existing"`
}

type BulkImportResponse struct {
	TotalRows     int      `json:"total_rows"`
	ImportedCount int      `json:"imported_count"`
	UpdatedCount  int      `json:"updated_count"`
	SkippedCount  int      `json:"skipped_count"`
	Errors        []string `json:"errors,omitempty"`
}

type BatchDeleteProductsRequest struct {
	ProductIDs []string `json:"product_ids" validate:"required,min=1,max=100"`
}

type BatchToggleProductStatusRequest struct {
	ProductIDs []string `json:"product_ids" validate:"required,min=1,max=100"`
	IsActive   bool     `json:"is_active"`
}

type BatchOperationResponse struct {
	AffectedCount int      `json:"affected_count"`
	Message       string   `json:"message"`
	Errors        []string `json:"errors,omitempty"`
}


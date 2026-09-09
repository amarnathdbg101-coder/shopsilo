// Package dto handler request and response struct.
package dto

import (
	"time"

	"shopMe/internal/handler/model"
)

type CreateProductRequest struct {
	Name          string   `json:"name" validate:"required,min=2,max=200"`
	Description   string   `json:"description,omitempty"`
	SKU           string   `json:"sku" validate:"required,min=2,max=100"`
	Price         float64  `json:"price" validate:"required,gt=0"`
	CostPrice     float64  `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	FloorPrice    float64  `json:"floor_price,omitempty" validate:"omitempty,gte=0"`
	AllowBargain  *bool    `json:"allow_bargain,omitempty"`
	ComparePrice  float64  `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	CategoryID    string   `json:"category_id" validate:"required"`
	StockQuantity     int                    `json:"stock_quantity" validate:"gte=0"`
	MinStock          *int                   `json:"min_stock,omitempty" validate:"omitempty,gte=0"` // Shop owner configured minimum stock threshold (default 1)
	LowStockThreshold *int                   `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0"`
	Images            []string               `json:"images,omitempty" validate:"omitempty,max=4"` // Max 4 images professional standard
	Weight            float64                `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive          bool                   `json:"is_active"`
	IsFeatured        bool                   `json:"is_featured"`
	Tags              []string               `json:"tags,omitempty"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"` // Brand, Model, Size, Color, Gender, Season, etc.
}

type UpdateProductRequest struct {
	Name              *string                 `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description       *string                 `json:"description,omitempty"`
	SKU               *string                 `json:"sku,omitempty" validate:"omitempty,min=2,max=100"`
	Price             *float64                `json:"price,omitempty" validate:"omitempty,gt=0"`
	CostPrice         *float64                `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	FloorPrice        *float64                `json:"floor_price,omitempty" validate:"omitempty,gte=0"`
	AllowBargain      *bool                   `json:"allow_bargain,omitempty"`
	ComparePrice      *float64                `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	CategoryID        *string                 `json:"category_id,omitempty"`
	StockQuantity     *int                    `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	MinStock          *int                    `json:"min_stock,omitempty" validate:"omitempty,gte=0"`
	LowStockThreshold *int                    `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0"`
	Images            *[]string               `json:"images,omitempty" validate:"omitempty,max=4"` // Max 4 images
	Weight            *float64                `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive          *bool                   `json:"is_active,omitempty"`
	IsFeatured        *bool                   `json:"is_featured,omitempty"`
	Tags              *[]string               `json:"tags,omitempty"`
	Attributes        *map[string]interface{} `json:"attributes,omitempty"` // Dynamic JSONB attributes
}

type ProductFilter struct {
	Search     string   `json:"search,omitempty"`
	CategoryID string   `json:"category_id,omitempty"`
	ShopID     string   `json:"shop_id,omitempty"`
	MinPrice   float64  `json:"min_price,omitempty"`
	MaxPrice   float64  `json:"max_price,omitempty"`
	IsFeatured *bool    `json:"is_featured,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"` // price_asc, price_desc, newest, stock_desc, oldest
	Page       int      `json:"page,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

type ProductPaginationResponse struct {
	Products   []*model.Product `json:"products"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}

type NearbyProductItem struct {
	ProductID         string   `json:"product_id"`
	Name              string   `json:"name"`
	Slug              string   `json:"slug"`
	Price             float64  `json:"price"`
	Images            []string `json:"images"`
	AvailableQuantity int      `json:"available_quantity"`
	ShopID            string   `json:"shop_id"`
	ShopName          string   `json:"shop_name"`
	ShopSlug          string   `json:"shop_slug"`
	ShopAddress       string   `json:"shop_address"`
	ShopPhone         string   `json:"shop_phone"`
	DistanceKm        float64  `json:"distance_km"`
	IsOpen            bool     `json:"is_open"`
	IsCurrentlyOpen   bool     `json:"is_currently_open"`
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

// MakeOfferRequest is submitted by shoppers proposing a custom price for a product.
type MakeOfferRequest struct {
	OfferedPrice  float64 `json:"offered_price" validate:"required,gt=0"`
	Quantity      int     `json:"quantity" validate:"omitempty,gte=1"`
	CustomerPhone string  `json:"customer_phone" validate:"required,min=10,max=15"`
	CustomerName  string  `json:"customer_name,omitempty"`
}

// BundleUpsellSuggestion suggests volume purchasing to reach the customer's desired per-unit price without destroying shop margins.
type BundleUpsellSuggestion struct {
	BundleQuantity int     `json:"bundle_quantity"`
	UnitPrice      float64 `json:"unit_price"`
	TotalBundlePrice float64 `json:"total_bundle_price"`
	TotalSavings   float64 `json:"total_savings"`
	Description    string  `json:"description"`
}

// BargainNegotiationResponse returns algorithmic negotiation results (accepted, counter-offer, or disabled).
type BargainNegotiationResponse struct {
	Status             string                  `json:"status"` // "DEAL_ACCEPTED", "COUNTER_OFFER", "BARGAIN_DISABLED"
	ProductID          string                  `json:"product_id"`
	ProductName        string                  `json:"product_name"`
	OriginalPrice      float64                 `json:"original_price"`
	OfferedPrice       float64                 `json:"offered_price"`
	AgreedPrice        float64                 `json:"agreed_price"`
	SavingsAmount      float64                 `json:"savings_amount"`
	SavingsPercentage  float64                 `json:"savings_percentage"`
	DealCode           string                  `json:"deal_code,omitempty"`
	ExpiresAt          *time.Time              `json:"expires_at,omitempty"`
	Message            string                  `json:"message"`
	BundleUpsell       *BundleUpsellSuggestion `json:"bundle_upsell,omitempty"`
	WhatsAppOrderURL   string                  `json:"whatsapp_order_url,omitempty"`
}

// POSBargainAssistResponse gives real-time color-coded negotiation guidance to cashiers standing at the counter.
type POSBargainAssistResponse struct {
	ProductID        string  `json:"product_id"`
	ProductName      string  `json:"product_name"`
	DisplayPrice     float64 `json:"display_price"`
	CostPrice        float64 `json:"cost_price"`
	FloorPrice       float64 `json:"floor_price"`
	ProposedPrice    float64 `json:"proposed_price"`
	MarginPercentage float64 `json:"margin_percentage"`
	ProfitAmount     float64 `json:"profit_amount"`
	StatusColor      string  `json:"status_color"` // "green", "yellow", "red"
	Advice           string  `json:"advice"`
}


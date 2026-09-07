// Package dto handler request and response struct.
package dto

import "shopMe/internal/handler/model"

type CreateProductRequest struct {
	Name          string   `json:"name" validate:"required,min=2,max=200"`
	Description   string   `json:"description,omitempty"`
	SKU           string   `json:"sku" validate:"required,min=2,max=100"`
	Price         float64  `json:"price" validate:"required,gt=0"`
	CostPrice     float64  `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice  float64  `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	CategoryID    string   `json:"category_id" validate:"required"`
	StockQuantity int      `json:"stock_quantity" validate:"gte=0"`
	Images        []string `json:"images,omitempty" validate:"omitempty,max=4,dive,url"` // Max 4 images professional standard
	Weight        float64  `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive      bool     `json:"is_active"`
	IsFeatured    bool     `json:"is_featured"`
	Tags          []string `json:"tags,omitempty"`
}

type UpdateProductRequest struct {
	Name          *string   `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description   *string   `json:"description,omitempty"`
	SKU           *string   `json:"sku,omitempty" validate:"omitempty,min=2,max=100"`
	Price         *float64  `json:"price,omitempty" validate:"omitempty,gt=0"`
	CostPrice     *float64  `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice  *float64  `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	CategoryID    *string   `json:"category_id,omitempty"`
	StockQuantity *int      `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	Images        *[]string `json:"images,omitempty" validate:"omitempty,max=4,dive,url"` // Max 4 images
	Weight        *float64  `json:"weight,omitempty" validate:"omitempty,gte=0"`
	IsActive      *bool     `json:"is_active,omitempty"`
	IsFeatured    *bool     `json:"is_featured,omitempty"`
	Tags          *[]string `json:"tags,omitempty"`
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

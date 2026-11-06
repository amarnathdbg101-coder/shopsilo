// Package dto handles HTTP request and response structures.
package dto

import (
	"time"

	"shopMe/internal/handler/model"
)

// CreateLocationRequest contains fields for creating a new godown or showroom.
type CreateLocationRequest struct {
	Name          string             `json:"name" validate:"required,min=2,max=100"`
	Code          string             `json:"code" validate:"required,min=2,max=50"`
	Type          model.LocationType `json:"type" validate:"required,oneof=RETAIL_STORE WAREHOUSE GODOWN OUTLET"`
	Address       string             `json:"address,omitempty" validate:"omitempty,max=500"`
	ContactPerson string             `json:"contact_person,omitempty" validate:"omitempty,max=100"`
	ContactPhone  string             `json:"contact_phone,omitempty" validate:"omitempty,max=20"`
	IsDefault     bool               `json:"is_default"`
}

// UpdateLocationRequest contains fields for updating a godown or showroom.
type UpdateLocationRequest struct {
	Name          *string             `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Type          *model.LocationType `json:"type,omitempty" validate:"omitempty,oneof=RETAIL_STORE WAREHOUSE GODOWN OUTLET"`
	Address       *string             `json:"address,omitempty" validate:"omitempty,max=500"`
	ContactPerson *string             `json:"contact_person,omitempty" validate:"omitempty,max=100"`
	ContactPhone  *string             `json:"contact_phone,omitempty" validate:"omitempty,max=20"`
	IsDefault     *bool               `json:"is_default,omitempty"`
	IsActive      *bool               `json:"is_active,omitempty"`
}

// CreateVariantRequest contains fields for defining a single child SKU (size/color/shade).
type CreateVariantRequest struct {
	SKU          string     `json:"sku" validate:"required,min=2,max=100"`
	Barcode      string     `json:"barcode,omitempty" validate:"omitempty,max=100"`
	Title        string     `json:"title" validate:"required,min=1,max=200"`
	Size         string     `json:"size,omitempty" validate:"omitempty,max=30"`
	Color        string     `json:"color,omitempty" validate:"omitempty,max=50"`
	ColorHex     string     `json:"color_hex,omitempty" validate:"omitempty,max=10"`
	ShadeCode    string     `json:"shade_code,omitempty" validate:"omitempty,max=50"`
	BatchNumber  string     `json:"batch_number,omitempty" validate:"omitempty,max=50"`
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`
	Price        float64    `json:"price" validate:"required,gt=0"`
	CostPrice    float64    `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice float64    `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	FloorPrice   float64    `json:"floor_price,omitempty" validate:"omitempty,gte=0"`
	Weight       float64    `json:"weight,omitempty" validate:"omitempty,gte=0"`
	ImageURL     string     `json:"image_url,omitempty"`
	InitialStock int        `json:"initial_stock,omitempty" validate:"omitempty,gte=0"`
	LocationID   string     `json:"location_id,omitempty"` // Defaults to shop's main store if empty
}

// ColorOption represents a color entry with name and optional hex code for swatch rendering.
type ColorOption struct {
	Name string `json:"name" validate:"required"`
	Hex  string `json:"hex,omitempty"`
}

// GenerateVariantMatrixRequest enables 1-tap bulk creation of all combinations (Size x Color).
// e.g., Sizes: ["S", "M", "L"], Colors: ["Navy", "Maroon"] creates 6 variant SKUs instantly.
type GenerateVariantMatrixRequest struct {
	Sizes            []string      `json:"sizes,omitempty"`                      // e.g. ["S", "M", "L", "XL"]
	Colors           []ColorOption `json:"colors,omitempty"`                     // e.g. [{"name": "Navy Blue", "hex": "#000080"}]
	Shades           []string      `json:"shades,omitempty"`                     // Cosmetics: ["#101 Fair", "#102 Medium"]
	Price            float64       `json:"price" validate:"required,gt=0"`       // Default selling price
	CostPrice        float64       `json:"cost_price,omitempty"`                 // Wholesale kharid price
	ComparePrice     float64       `json:"compare_price,omitempty"`              // MRP
	FloorPrice       float64       `json:"floor_price,omitempty"`                // Bargain lower bound
	InitialStockEach int           `json:"initial_stock_each,omitempty"`         // Seed stock for each generated variant
	LocationID       string        `json:"location_id,omitempty"`                // Default location to seed stock into
}

// UpdateVariantRequest contains editable fields of a single SKU variant.
type UpdateVariantRequest struct {
	Barcode      *string    `json:"barcode,omitempty" validate:"omitempty,max=100"`
	Title        *string    `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Size         *string    `json:"size,omitempty" validate:"omitempty,max=30"`
	Color        *string    `json:"color,omitempty" validate:"omitempty,max=50"`
	ColorHex     *string    `json:"color_hex,omitempty" validate:"omitempty,max=10"`
	ShadeCode    *string    `json:"shade_code,omitempty" validate:"omitempty,max=50"`
	BatchNumber  *string    `json:"batch_number,omitempty" validate:"omitempty,max=50"`
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`
	Price        *float64   `json:"price,omitempty" validate:"omitempty,gt=0"`
	CostPrice    *float64   `json:"cost_price,omitempty" validate:"omitempty,gte=0"`
	ComparePrice *float64   `json:"compare_price,omitempty" validate:"omitempty,gte=0"`
	FloorPrice   *float64   `json:"floor_price,omitempty" validate:"omitempty,gte=0"`
	ImageURL     *string    `json:"image_url,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
}

// StockTransferItemRequest specifies a single item line to transfer.
type StockTransferItemRequest struct {
	ProductID    string  `json:"product_id" validate:"required,uuid"`
	VariantID    *string `json:"variant_id,omitempty"`
	RequestedQty int     `json:"requested_qty" validate:"required,gt=0"`
	Notes        string  `json:"notes,omitempty" validate:"omitempty,max=200"`
}

// CreateStockTransferRequest submits a transfer requisition from Godown to Showroom or vice-versa.
type CreateStockTransferRequest struct {
	FromLocationID string                     `json:"from_location_id" validate:"required,uuid"`
	ToLocationID   string                     `json:"to_location_id" validate:"required,uuid"`
	Notes          string                     `json:"notes,omitempty" validate:"omitempty,max=500"`
	Items          []StockTransferItemRequest `json:"items" validate:"required,min=1"`
}

// UpdateTransferStatusRequest transitions the transfer along the state machine.
// PENDING -> APPROVED -> DISPATCHED -> IN_TRANSIT -> RECEIVED -> RECONCILED / CANCELLED
type UpdateTransferStatusRequest struct {
	Status model.TransferStatus `json:"status" validate:"required,oneof=APPROVED DISPATCHED IN_TRANSIT RECEIVED RECONCILED CANCELLED"`
	Notes  string               `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// AdjustLocationStockRequest allows targeted stock audit or wholesale arrival per location.
type AdjustLocationStockRequest struct {
	LocationID string  `json:"location_id" validate:"required,uuid"`
	ProductID  string  `json:"product_id" validate:"required,uuid"`
	VariantID  *string `json:"variant_id,omitempty"`
	Adjustment int     `json:"adjustment" validate:"required"` // e.g. +20 or -5
	Notes      string  `json:"notes,omitempty" validate:"omitempty,max=200"`
}

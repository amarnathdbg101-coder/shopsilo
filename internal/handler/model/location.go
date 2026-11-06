// Package model defines database entity models.
package model

import (
	"time"
)

// LocationType represents the physical role of a business facility.
type LocationType string

const (
	LocationTypeRetailStore LocationType = "RETAIL_STORE" // Main showroom / shop floor where POS checkout happens
	LocationTypeWarehouse   LocationType = "WAREHOUSE"    // Central storage / wholesale warehouse
	LocationTypeGodown      LocationType = "GODOWN"       // Local backroom / basement / secondary godown
	LocationTypeOutlet      LocationType = "OUTLET"       // Branch / secondary sales counter
)

// Location represents a physical business premise (Showroom, Godown, Warehouse) for multi-location inventory.
type Location struct {
	ID            string       `json:"id"`
	ShopID        string       `json:"shop_id"`
	Name          string       `json:"name"`           // e.g. "Main Showroom", "Basement Godown #1"
	Code          string       `json:"code"`           // e.g. "MAIN", "GDN-1"
	Type          LocationType `json:"type"`           // RETAIL_STORE, WAREHOUSE, GODOWN, OUTLET
	Address       string       `json:"address,omitempty"`
	ContactPerson string       `json:"contact_person,omitempty"`
	ContactPhone  string       `json:"contact_phone,omitempty"`
	IsDefault     bool         `json:"is_default"`     // Primary sales counter for default checkout
	IsActive      bool         `json:"is_active"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ProductVariant represents a specific size, color, shade, or batch SKU of a parent product.
// Specially tailored for Garment (size/color) and Cosmetics (shade/expiry/batch) retail.
type ProductVariant struct {
	ID           string     `json:"id"`
	ProductID    string     `json:"product_id"`
	ShopID       string     `json:"shop_id"`
	SKU          string     `json:"sku"`                    // e.g. "RAYON-KURTI-NAVY-L"
	Barcode      string     `json:"barcode,omitempty"`      // Scannable EAN/UPC or in-house sticker
	Title        string     `json:"title"`                  // e.g. "Navy Blue / L"
	Size         string     `json:"size,omitempty"`         // "S", "M", "L", "XL", "38", "40", "Free Size"
	Color        string     `json:"color,omitempty"`        // "Navy Blue", "Maroon", "Bottle Green"
	ColorHex     string     `json:"color_hex,omitempty"`    // "#000080" for UI swatch circles
	ShadeCode    string     `json:"shade_code,omitempty"`   // "#220 Warm Sand" (cosmetics)
	BatchNumber  string     `json:"batch_number,omitempty"` // "B-2026-X1" (cosmetics safety)
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`  // Expiry date for creams, perfumes, lotions
	Price        float64    `json:"price"`                  // Selling Price (₹)
	CostPrice    float64    `json:"cost_price"`             // Wholesale Purchase / Kharid Price (₹)
	ComparePrice float64    `json:"compare_price"`          // MRP (₹)
	FloorPrice   float64    `json:"floor_price"`            // Stealth bargaining lower limit (₹)
	Weight       float64    `json:"weight,omitempty"`
	ImageURL     string     `json:"image_url,omitempty"`    // Variant-specific photo
	IsActive     bool       `json:"is_active"`
	AvailableQty int        `json:"available_qty,omitempty"`// Joined stock quantity across selected location
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// LocationInventory tracks physical stock at a specific location for either a parent product or variant SKU.
type LocationInventory struct {
	ID                string     `json:"id"`
	ShopID            string     `json:"shop_id"`
	LocationID        string     `json:"location_id"`
	LocationName      string     `json:"location_name,omitempty"`
	ProductID         string     `json:"product_id"`
	ProductName       string     `json:"product_name,omitempty"`
	VariantID         *string    `json:"variant_id,omitempty"`
	VariantTitle      string     `json:"variant_title,omitempty"`
	VariantSKU        string     `json:"variant_sku,omitempty"`
	Quantity          int        `json:"quantity"`           // Physical count on shelf / rack
	ReservedQuantity  int        `json:"reserved_quantity"`  // Held in pending cart / customer reserve
	AvailableQuantity int        `json:"available_quantity"` // Quantity - ReservedQuantity
	LowStockThreshold int        `json:"low_stock_threshold"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TransferStatus represents the workflow stage of inter-store or godown-to-counter stock movements.
type TransferStatus string

const (
	TransferStatusPending    TransferStatus = "PENDING"    // Transfer requisition created
	TransferStatusApproved   TransferStatus = "APPROVED"   // Manager approved transfer
	TransferStatusDispatched TransferStatus = "DISPATCHED" // Stock packed and left source godown
	TransferStatusInTransit  TransferStatus = "IN_TRANSIT"  // On delivery vehicle / cart
	TransferStatusReceived   TransferStatus = "RECEIVED"   // Physically counted at destination
	TransferStatusReconciled TransferStatus = "RECONCILED" // Stock balances permanently updated
	TransferStatusCancelled  TransferStatus = "CANCELLED"  // Requisition aborted
)

// StockTransfer tracks moving inventory between Godown and Showroom with an audit trail.
type StockTransfer struct {
	ID               string              `json:"id"`
	ShopID           string              `json:"shop_id"`
	TransferNumber   string              `json:"transfer_number"`    // e.g. "TRF-202611-0001"
	FromLocationID   string              `json:"from_location_id"`
	FromLocationName string              `json:"from_location_name,omitempty"`
	ToLocationID     string              `json:"to_location_id"`
	ToLocationName   string              `json:"to_location_name,omitempty"`
	Status           TransferStatus      `json:"status"`
	Notes            string              `json:"notes,omitempty"`
	CreatedBy        *string             `json:"created_by,omitempty"`
	CreatedByName    string              `json:"created_by_name,omitempty"`
	ReceivedBy       *string             `json:"received_by,omitempty"`
	ReceivedByName   string              `json:"received_by_name,omitempty"`
	DispatchedAt     *time.Time          `json:"dispatched_at,omitempty"`
	ReceivedAt       *time.Time          `json:"received_at,omitempty"`
	Items            []StockTransferItem `json:"items,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// StockTransferItem represents an individual product/variant line in a stock transfer requisition.
type StockTransferItem struct {
	ID           string  `json:"id"`
	TransferID   string  `json:"transfer_id"`
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name,omitempty"`
	VariantID    *string `json:"variant_id,omitempty"`
	VariantTitle string  `json:"variant_title,omitempty"`
	VariantSKU   string  `json:"variant_sku,omitempty"`
	RequestedQty int     `json:"requested_qty"` // Quantity requested by showroom
	SentQty      int     `json:"sent_qty"`      // Quantity loaded from godown
	ReceivedQty  int     `json:"received_qty"`  // Quantity verified upon arrival
	Notes        string  `json:"notes,omitempty"`
}

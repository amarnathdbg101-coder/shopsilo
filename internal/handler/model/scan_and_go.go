// Package model contains core database entity structs.
package model

import "time"

// ScanAndGoStatus represents the lifecycle state of an in-store self checkout cart.
type ScanAndGoStatus string

const (
	ScanAndGoActive             ScanAndGoStatus = "ACTIVE"
	ScanAndGoVerifiedAtCounter  ScanAndGoStatus = "VERIFIED_AT_COUNTER"
	ScanAndGoConvertedToBill    ScanAndGoStatus = "CONVERTED_TO_BILL"
	ScanAndGoExpired            ScanAndGoStatus = "EXPIRED"
	ScanAndGoCancelled          ScanAndGoStatus = "CANCELLED"
)

// ScanAndGoSession represents a customer's self-checkout session created on mobile inside the shop.
type ScanAndGoSession struct {
	ID              string          `json:"id"`
	ShopID          string          `json:"shop_id"`
	CustomerUserID  *string         `json:"customer_user_id,omitempty"`
	CustomerName    string          `json:"customer_name"`
	CustomerPhone   string          `json:"customer_phone"`
	SessionCode     string          `json:"session_code"` // 6-digit numeric verification code (e.g. 849201)
	QRToken         string          `json:"qr_token"`     // Cryptographic token encoded into session QR
	Status          ScanAndGoStatus `json:"status"`
	TotalItems      int             `json:"total_items"`
	Subtotal        float64         `json:"subtotal"`
	ExpiresAt       time.Time       `json:"expires_at"`
	ConvertedBillID *string         `json:"converted_bill_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	Items           []*ScanAndGoItem `json:"items,omitempty"`
}

// ScanAndGoItem represents an individual item scanned by customer on their phone camera.
type ScanAndGoItem struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	ProductID   string    `json:"product_id"`
	VariantID   *string   `json:"variant_id,omitempty"`
	ProductName string    `json:"product_name"`
	Barcode     string    `json:"barcode"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	TotalPrice  float64   `json:"total_price"`
	CreatedAt   time.Time `json:"created_at"`
}


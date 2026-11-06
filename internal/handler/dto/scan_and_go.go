// Package dto defines request and response payloads.
package dto

import (
	"time"

	"shopMe/internal/handler/model"
)

// StartScanSessionRequest initializes a new in-store self checkout cart.
type StartScanSessionRequest struct {
	ShopID        string `json:"shop_id" validate:"required"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
}

// AddItemToCartRequest adds a scanned item to customer's self-checkout cart.
type AddItemToCartRequest struct {
	Barcode   string  `json:"barcode"`
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity" validate:"required,min=1"`
}

// UpdateItemQuantityRequest changes the count of an item in the cart.
type UpdateItemQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required,min=0"`
}

// ConvertToBillRequest instructs the counter POS to finalize and convert the Scan & Go cart into an official POS bill.
type ConvertToBillRequest struct {
	PaymentMethod string  `json:"payment_method" validate:"required,oneof=cash upi split"`
	SplitCash     float64 `json:"split_cash_amount"`
	SplitUPI      float64 `json:"split_upi_amount"`
	CustomerPhone string  `json:"customer_phone"`
	CustomerName  string  `json:"customer_name"`
}

// ScanSessionResponse represents the full scan & go cart with all line items and verification code.
type ScanSessionResponse struct {
	SessionID       string                 `json:"session_id"`
	ShopID          string                 `json:"shop_id"`
	CustomerName    string                 `json:"customer_name"`
	CustomerPhone   string                 `json:"customer_phone"`
	SessionCode     string                 `json:"session_code"`
	QRToken         string                 `json:"qr_token"`
	Status          model.ScanAndGoStatus  `json:"status"`
	TotalItems      int                    `json:"total_items"`
	Subtotal        float64                `json:"subtotal"`
	ExpiresAt       time.Time              `json:"expires_at"`
	ConvertedBillID *string                `json:"converted_bill_id,omitempty"`
	Items           []*model.ScanAndGoItem `json:"items"`
}


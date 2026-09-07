// Package dto handler request and response struct.
package dto

import (
	"shopMe/internal/handler/model"
	"time"
)

type CreateOrderRequest struct {
	CartID          string        `json:"cart_id" validate:"required"`
	ShippingAddress model.Address `json:"shipping_address" validate:"required"`
	BillingAddress  model.Address `json:"billing_address,omitempty"`
	Notes           string        `json:"notes,omitempty"`
}

type OrderFilter struct {
	UserID   string            `json:"user_id,omitempty"`
	Status   model.OrderStatus `json:"status,omitempty"`
	FromDate time.Time         `json:"from_date,omitempty"`
	ToDate   time.Time         `json:"to_date,omitempty"`
	Page     int               `json:"page,omitempty"`
	Limit    int               `json:"limit,omitempty"`
}

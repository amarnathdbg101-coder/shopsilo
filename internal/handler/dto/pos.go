// Package dto handles request and response struct.
package dto

import "shopMe/internal/handler/model"

type POSSaleItemRequest struct {
	ProductID   string   `json:"product_id" validate:"required,uuid"`
	Quantity    int      `json:"quantity" validate:"required,gt=0"`
	CustomPrice *float64 `json:"custom_price,omitempty" validate:"omitempty,gte=0"`
}

type CreatePOSSaleRequest struct {
	CustomerPhone  string               `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	Items          []POSSaleItemRequest `json:"items" validate:"required,min=1,dive"`
	DiscountAmount float64              `json:"discount_amount" validate:"gte=0"`
	PaymentMethod  string               `json:"payment_method" validate:"required,oneof=cash upi card credit"`
}

type POSSaleResponse struct {
	Bill                   *model.POSBill `json:"bill"`
	ReceiptURL             string         `json:"receipt_url"`
	LoyaltyPointsCredited  int            `json:"loyalty_points_credited"`
}

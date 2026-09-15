// Package dto handles request and response struct.
package dto

import "shopMe/internal/handler/model"

type CreateReturnRequest struct {
	ProductID     string  `json:"product_id" validate:"required,uuid"`
	ReservationID *string `json:"reservation_id,omitempty" validate:"omitempty,uuid"`
	Quantity      int     `json:"quantity" validate:"required,gt=0"`
	RefundAmount  float64 `json:"refund_amount" validate:"gte=0"`
	Reason        string  `json:"reason" validate:"required,min=2,max=255"`
}

type CreateOfferRequest struct {
	Title             string `json:"title" validate:"required,min=2,max=200"`
	Description       string `json:"description,omitempty" validate:"omitempty,max=500"`
	DiscountText      string `json:"discount_text" validate:"required,min=2,max=100"`
	MinPointsRequired int    `json:"min_points_required" validate:"gte=0"`
	ExpiresInDays     int    `json:"expires_in_days,omitempty" validate:"omitempty,min=1,max=365"`
}

type UpdateOfferRequest struct {
	Title             string `json:"title" validate:"required,min=2,max=200"`
	Description       string `json:"description,omitempty" validate:"omitempty,max=500"`
	DiscountText      string `json:"discount_text" validate:"required,min=2,max=100"`
	MinPointsRequired int    `json:"min_points_required" validate:"gte=0"`
	ExpiresInDays     int    `json:"expires_in_days,omitempty" validate:"omitempty,min=1,max=365"`
	IsActive          *bool  `json:"is_active,omitempty"`
}

type OfferAuditRecord struct {
	ID                   string `json:"id"`
	OfferID              string `json:"offer_id"`
	ShopID               string `json:"shop_id"`
	Action               string `json:"action"` // CREATED, UPDATED, DELETED
	PreviousTitle        string `json:"previous_title,omitempty"`
	PreviousDiscountText string `json:"previous_discount_text,omitempty"`
	PreviousDescription  string `json:"previous_description,omitempty"`
	NewTitle             string `json:"new_title,omitempty"`
	NewDiscountText      string `json:"new_discount_text,omitempty"`
	NewDescription       string `json:"new_description,omitempty"`
	ChangedByUserID      string `json:"changed_by_user_id,omitempty"`
	CreatedAt            string `json:"created_at"`
}

type ScanProductResponse struct {
	Product      *model.Product `json:"product"`
	PointsReward int            `json:"points_reward"` // e.g. 1 pt per Rs 20
}

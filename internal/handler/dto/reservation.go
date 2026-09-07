// Package dto handle request and response struct.
package dto

import "shopMe/internal/handler/model"

type CreateReservationRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,min=1,max=10"`
	HoldHours int    `json:"hold_hours,omitempty" validate:"omitempty,min=1,max=24"`
	Notes     string `json:"notes,omitempty" validate:"omitempty,max=300"`
}

type VerifyReservationRequest struct {
	PickupCode        string `json:"pickup_code,omitempty" validate:"omitempty,min=4,max=10"`
	ReservationNumber string `json:"reservation_number,omitempty" validate:"omitempty,min=4,max=50"`
}

type ReservationFilter struct {
	Status string `json:"status,omitempty"`
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type ReservationPaginationResponse struct {
	Reservations []*model.Reservation `json:"reservations"`
	TotalCount   int                  `json:"total_count"`
	Page         int                  `json:"page"`
	Limit        int                  `json:"limit"`
	TotalPages   int                  `json:"total_pages"`
}

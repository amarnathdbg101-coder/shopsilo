// Package dto contains Data Transfer Objects for API request/response.
package dto

// EvaluateBargainRequest is sent when customer negotiates at POS counter.
type EvaluateBargainRequest struct {
	ProductID            string  `json:"product_id" validate:"required"`
	CustomerOfferedPrice float64 `json:"customer_offered_price" validate:"required,gt=0"`
	CustomerPhone        string  `json:"customer_phone"`
}

// ExecutePrescriptionRequest is sent when the merchant taps "Apply Now" on a prescription card.
type ExecutePrescriptionRequest struct {
	Notes string `json:"notes"`
}

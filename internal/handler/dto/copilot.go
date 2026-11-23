// Package dto contains Data Transfer Objects for API request/response.
package dto

// EvaluateBargainRequest is sent when customer negotiates at POS counter.
type EvaluateBargainRequest struct {
	ProductID            string  `json:"product_id" validate:"required"`
	CustomerOfferedPrice float64 `json:"customer_offered_price" validate:"required,gt=0"`
	CustomerPhone        string  `json:"customer_phone"`
}

// SearchSubstituteRequest is sent for natural language conversational product & substitute discovery.
type SearchSubstituteRequest struct {
	Query     string  `json:"query" validate:"required"`
	Category  string  `json:"category"`
	MaxPrice  float64 `json:"max_price"`
	SkinType  string  `json:"skin_type"`  // For cosmetics: oily, dry, acne, etc.
	Fabric    string  `json:"fabric"`    // For clothing: pure_cotton, rayon, etc.
	Occasion  string  `json:"occasion"`  // haldi, party, daily_wear
	Limit     int     `json:"limit"`
}

// ExecutePrescriptionRequest is sent when the merchant taps "Apply Now" on a prescription card.
type ExecutePrescriptionRequest struct {
	Notes string `json:"notes"`
}

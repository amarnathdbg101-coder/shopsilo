// Package dto handles request/response transfer objects.
package dto

// CreateProcurementItemRequest defines the input required to add an item to the procurement list.
type CreateProcurementItemRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=255"`
	Category       string  `json:"category"`
	Qty            string  `json:"qty" validate:"required,min=1,max=100"`
	EstimatedPrice float64 `json:"estimated_price"`
	ActualPrice    float64 `json:"actual_price"`
	DealerName     string  `json:"dealer_name"`
	DealerPhone    string  `json:"dealer_phone"`
	Notes          string  `json:"notes"`
	IsCompleted    bool    `json:"is_completed"`
}

// UpdateProcurementItemRequest defines the input to modify an existing procurement item.
type UpdateProcurementItemRequest struct {
	Name           *string  `json:"name,omitempty"`
	Category       *string  `json:"category,omitempty"`
	Qty            *string  `json:"qty,omitempty"`
	EstimatedPrice *float64 `json:"estimated_price,omitempty"`
	ActualPrice    *float64 `json:"actual_price,omitempty"`
	DealerName     *string  `json:"dealer_name,omitempty"`
	DealerPhone    *string  `json:"dealer_phone,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	IsCompleted    *bool    `json:"is_completed,omitempty"`
}

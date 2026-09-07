// Package dto handles request and response structs.
package dto

import "shopMe/internal/handler/model"

// RecordCreditTransactionRequest is used to give udhar / add credit to a customer.
type RecordCreditTransactionRequest struct {
	CustomerName   string  `json:"customer_name" validate:"required,min=2,max=150"`
	CustomerMobile string  `json:"customer_mobile" validate:"required,min=10,max=15"`
	Amount         float64 `json:"amount" validate:"required,gt=0"`
	Notes          string  `json:"notes,omitempty" validate:"omitempty,max=255"`
	BillNumber     string  `json:"bill_number,omitempty"`
}

// RecordPaymentRequest is used when a customer pays their dues (jama).
type RecordPaymentRequest struct {
	CustomerName   string  `json:"customer_name,omitempty" validate:"omitempty,min=2,max=150"`
	Amount         float64 `json:"amount" validate:"required,gt=0"`
	PaymentMode    string  `json:"payment_mode" validate:"required,oneof=cash upi card other"`
	Notes          string  `json:"notes,omitempty" validate:"omitempty,max=255"`
}

// KhataCustomerResponse represents a customer with their total outstanding credit balance.
type KhataCustomerResponse struct {
	ID             string  `json:"id"`
	CustomerName   string  `json:"customer_name"`
	CustomerMobile string  `json:"customer_mobile"`
	CurrentBalance float64 `json:"current_balance"`
	LastActivityAt string  `json:"last_activity_at"`
}

// KhataDetailResponse represents a customer's detailed passbook with full transaction history.
type KhataDetailResponse struct {
	Customer     model.CustomerKhata      `json:"customer"`
	Transactions []model.KhataTransaction `json:"transactions"`
}

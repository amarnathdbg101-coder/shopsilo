// Package dto defines request and response payloads.
package dto

import "shopMe/internal/handler/model"

// CreateMicroPlanRequest creates a daily micro-repayment schedule for an existing debtor.
type CreateMicroPlanRequest struct {
	CustomerPhone string  `json:"customer_phone" validate:"required"`
	CustomerName  string  `json:"customer_name"`
	TotalDebt     float64 `json:"total_debt" validate:"required,min=1"`
	DailyAmount   float64 `json:"daily_amount" validate:"required,min=1"`
	Frequency     string  `json:"frequency"` // DAILY, WEEKLY, MONTHLY (default DAILY)
}

// RecordMicroPaymentRequest logs a single micro installment payment.
type RecordMicroPaymentRequest struct {
	Amount               float64 `json:"amount" validate:"required,min=1"`
	PaymentMode          string  `json:"payment_mode" validate:"required,oneof=CASH UPI QR"`
	TransactionReference string  `json:"transaction_reference"`
	Notes                string  `json:"notes"`
}

// MicroPlanSummaryResponse bundles plan metrics with payment history and progress.
type MicroPlanSummaryResponse struct {
	Plan            *model.KhataMicroPlan      `json:"plan"`
	Payments        []*model.KhataMicroPayment `json:"payments"`
	ProgressPercent float64                    `json:"progress_percent"` // 0.0 - 100.0%
}


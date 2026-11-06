// Package model contains core database entity structs.
package model

import "time"

// PlanStatus defines the active state of a micro-installment plan.
type PlanStatus string

const (
	PlanStatusActive    PlanStatus = "ACTIVE"
	PlanStatusCompleted PlanStatus = "COMPLETED"
	PlanStatusPaused    PlanStatus = "PAUSED"
	PlanStatusDefaulted PlanStatus = "DEFAULTED"
)

// KhataMicroPlan represents a structured micro-installment plan for large debt recovery.
type KhataMicroPlan struct {
	ID                string     `json:"id"`
	ShopID            string     `json:"shop_id"`
	CustomerID        *string    `json:"customer_id,omitempty"`
	CustomerPhone     string     `json:"customer_phone"`
	CustomerName      string     `json:"customer_name"`
	TotalDebt         float64    `json:"total_debt"`
	DailyAmount       float64    `json:"daily_amount"`
	Frequency         string     `json:"frequency"` // DAILY, WEEKLY, MONTHLY
	TotalInstallments int        `json:"total_installments"`
	PaidInstallments  int        `json:"paid_installments"`
	TotalPaidAmount   float64    `json:"total_paid_amount"`
	RemainingBalance  float64    `json:"remaining_balance"`
	NextDueDate       time.Time  `json:"next_due_date"`
	Status            PlanStatus `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// KhataMicroPayment records an installment payment made towards a micro plan.
type KhataMicroPayment struct {
	ID                   string    `json:"id"`
	PlanID               string    `json:"plan_id"`
	ShopID               string    `json:"shop_id"`
	InstallmentNumber    int       `json:"installment_number"`
	Amount               float64   `json:"amount"`
	PaymentMode          string    `json:"payment_mode"` // CASH, UPI, QR
	TransactionReference string    `json:"transaction_reference"`
	Notes                string    `json:"notes"`
	CreatedAt            time.Time `json:"created_at"`
}


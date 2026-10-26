// Package dto handles request and response structs.
package dto

import "shopMe/internal/handler/model"

// CreateExpenseRequest is used to log a daily store expense.
type CreateExpenseRequest struct {
	Category      string  `json:"category" validate:"required,oneof=rent electricity staff_salary tea_snacks packaging other chai_nashta transport salary maintenance"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
	Notes         string  `json:"notes,omitempty" validate:"omitempty,max=255"`
	PaymentMethod string  `json:"payment_method,omitempty" validate:"omitempty,oneof=cash upi card other online"`
	ExpenseDate   string  `json:"expense_date,omitempty"` // YYYY-MM-DD, defaults to today
}

// ExpenseListResponse represents expenses with summary totals.
type ExpenseListResponse struct {
	Expenses      []model.ShopExpense        `json:"expenses"`
	TotalAmount   float64                    `json:"total_amount"`
	CategoryTotal map[string]float64         `json:"category_total"`
	Month         int                        `json:"month"`
	Year          int                        `json:"year"`
}

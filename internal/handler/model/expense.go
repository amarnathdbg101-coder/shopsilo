// Package model handles database structs.
package model

import "time"

// Common expense categories for retail shops
const (
	ExpenseCategoryRent        = "rent"
	ExpenseCategoryElectricity = "electricity"
	ExpenseCategorySalary      = "staff_salary"
	ExpenseCategoryTeaSnacks   = "tea_snacks"
	ExpenseCategoryPackaging   = "packaging"
	ExpenseCategoryOther       = "other"
)

// ShopExpense represents a day-to-day store operational expense.
type ShopExpense struct {
	ID            string    `json:"id"`
	ShopID        string    `json:"shop_id"`
	Category      string    `json:"category"`
	Amount        float64   `json:"amount"`
	Notes         string    `json:"notes,omitempty"`
	PaymentMethod string    `json:"payment_method"` // cash, upi, etc.
	ExpenseDate   string    `json:"expense_date"`   // YYYY-MM-DD
	CreatedAt     time.Time `json:"created_at"`
}

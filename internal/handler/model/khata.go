// Package model handles database structs.
package model

import "time"

const (
	KhataTxTypeGiveCredit     = "GIVE_CREDIT"     // Udhar Diya
	KhataTxTypeReceivePayment = "RECEIVE_PAYMENT" // Jama Liya
)

// CustomerKhata represents a customer credit account with a shop.
type CustomerKhata struct {
	ID             string    `json:"id"`
	ShopID         string    `json:"shop_id"`
	CustomerName   string    `json:"customer_name"`
	CustomerMobile string    `json:"customer_mobile"`
	CurrentBalance float64   `json:"current_balance"` // How much money customer owes the shopkeeper
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// KhataTransaction represents an entry in the customer's passbook/ledger.
type KhataTransaction struct {
	ID           string    `json:"id"`
	KhataID      string    `json:"khata_id"`
	ShopID       string    `json:"shop_id"`
	Type         string    `json:"type"` // GIVE_CREDIT or RECEIVE_PAYMENT
	Amount       float64   `json:"amount"`
	BalanceAfter float64   `json:"balance_after"`
	Notes        string    `json:"notes,omitempty"`
	BillNumber   string    `json:"bill_number,omitempty"`
	PaymentMode  string    `json:"payment_mode,omitempty"` // cash, upi, etc.
	CreatedAt    time.Time `json:"created_at"`
}

// KhataSummary represents total market credit balance and indebted customers count.
type KhataSummary struct {
	TotalOutstandingAmount float64 `json:"total_outstanding_amount"`
	TotalCustomers         int     `json:"total_customers"`
	TotalTransactionsCount int     `json:"total_transactions_count"`
}

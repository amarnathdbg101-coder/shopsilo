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
	CreditLimit    float64 `json:"credit_limit"`
	CurrentBalance float64 `json:"current_balance"`
	LastActivityAt string  `json:"last_activity_at"`
}

// SetCreditLimitRequest sets a maximum credit cap on a customer's khata account.
type SetCreditLimitRequest struct {
	CreditLimit float64 `json:"credit_limit" validate:"min=0"`
}

// OverdueCustomerItem represents a customer with unpaid credit balance in an aging bracket.
type OverdueCustomerItem struct {
	KhataID        string  `json:"khata_id"`
	CustomerName   string  `json:"customer_name"`
	CustomerMobile string  `json:"customer_mobile"`
	CurrentBalance float64 `json:"current_balance"`
	CreditLimit    float64 `json:"credit_limit"`
	DaysOverdue    int     `json:"days_overdue"`
	LastActivityAt string  `json:"last_activity_at"`
}

// KhataAgingReport categorizes store credit into aging brackets to spot bad debt risks early.
type KhataAgingReport struct {
	TotalOutstanding float64               `json:"total_outstanding"`
	TotalCustomers   int                   `json:"total_customers"`
	Bucket0To30      float64               `json:"bucket_0_to_30"`  // 0 to 30 days old
	Bucket31To60     float64               `json:"bucket_31_to_60"` // 31 to 60 days old
	Bucket60Plus     float64               `json:"bucket_60_plus"`  // Over 60 days old (high risk)
	OverdueList      []OverdueCustomerItem `json:"overdue_list"`
}

// KhataDetailResponse represents a customer's detailed passbook with full transaction history.
type KhataDetailResponse struct {
	Customer     model.CustomerKhata      `json:"customer"`
	Transactions []model.KhataTransaction `json:"transactions"`
}

// KhataReminderResponse holds the generated WhatsApp payment reminder for a customer.
type KhataReminderResponse struct {
	CustomerName   string  `json:"customer_name"`
	CustomerMobile string  `json:"customer_mobile"`
	MaskedMobile   string  `json:"masked_mobile"`
	DueAmount      float64 `json:"due_amount"`
	ReminderText   string  `json:"reminder_text"`
	WhatsAppURL    string  `json:"whatsapp_url"`
}

// KhataStatementShareResponse holds the generated WhatsApp statement link and details for a customer.
type KhataStatementShareResponse struct {
	CustomerName      string  `json:"customer_name"`
	CustomerMobile    string  `json:"customer_mobile"`
	MaskedMobile      string  `json:"masked_mobile"`
	CurrentBalance    float64 `json:"current_balance"`
	TotalTransactions int     `json:"total_transactions"`
	StatementURL      string  `json:"statement_url"`
	WhatsAppShareURL  string  `json:"whatsapp_share_url"`
}

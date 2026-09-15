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
	ParchiImageURL string  `json:"parchi_image_url,omitempty"`
	ItemsSummary   string  `json:"items_summary,omitempty"`
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
	ID                 string  `json:"id"`
	CustomerName       string  `json:"customer_name"`
	CustomerMobile     string  `json:"customer_mobile"`
	CreditLimit        float64 `json:"credit_limit"`
	CurrentBalance     float64 `json:"current_balance"`
	ClosureStatus      string  `json:"closure_status,omitempty"`
	ClosureRequestedBy string  `json:"closure_requested_by,omitempty"`
	ClosureOTP         string  `json:"closure_otp,omitempty"`
	PromiseToPayDate   string  `json:"promise_to_pay_date,omitempty"`
	InstallmentTarget  float64 `json:"installment_target"`
	TrustScore         int     `json:"trust_score"`
	TrustBadge         string  `json:"trust_badge"`
	IsRegistered       bool    `json:"is_registered"`
	LastActivityAt     string  `json:"last_activity_at"`
}

// SetCreditLimitRequest sets a maximum credit cap on a customer's khata account.
type SetCreditLimitRequest struct {
	CreditLimit float64 `json:"credit_limit" validate:"gte=0"`
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

// CustomerShopKhataItem represents an individual shop where the customer has an udhar/khata account.
type CustomerShopKhataItem struct {
	KhataID            string  `json:"khata_id"`
	ShopID             string  `json:"shop_id"`
	ShopName           string  `json:"shop_name"`
	ShopSlug           string  `json:"shop_slug"`
	ShopPhone          string  `json:"shop_phone"`
	ShopWhatsApp       string  `json:"shop_whatsapp,omitempty"`
	ShopAddress        string  `json:"shop_address,omitempty"`
	ShopLogo           string  `json:"shop_logo,omitempty"`
	ShopUPIID          string  `json:"shop_upi_id,omitempty"`
	CurrentBalance     float64 `json:"current_balance"`
	CreditLimit        float64 `json:"credit_limit"`
	CreditOTPRequired  bool    `json:"credit_otp_required"`
	CreditOTPThreshold float64 `json:"credit_otp_threshold"`
	ClosureStatus      string  `json:"closure_status,omitempty"`
	ClosureRequestedBy string  `json:"closure_requested_by,omitempty"`
	ClosureOTP         string  `json:"closure_otp,omitempty"` // populated if customer should see it to share with merchant
	PromiseToPayDate   string  `json:"promise_to_pay_date,omitempty"`
	InstallmentTarget  float64 `json:"installment_target"`
	TrustScore         int     `json:"trust_score"`
	TrustBadge         string  `json:"trust_badge"`
	LastActivityAt     string  `json:"last_activity_at,omitempty"`
}

// CustomerKhataSummaryResponse gives a customer their complete multi-shop credit portfolio.
type CustomerKhataSummaryResponse struct {
	TotalMarketDue  float64                 `json:"total_market_due"`
	TotalShopsCount int                     `json:"total_shops_count"`
	Shops           []CustomerShopKhataItem `json:"shops"`
}

// CustomerKhataPassbookResponse provides the full itemized passbook ledger for a customer at a shop.
type CustomerKhataPassbookResponse struct {
	Shop         CustomerShopKhataItem    `json:"shop"`
	CustomerName string                   `json:"customer_name"`
	Transactions []model.KhataTransaction `json:"transactions"`
}

// DisputeTransactionRequest allows customer to flag an incorrect udhar entry.
type DisputeTransactionRequest struct {
	TransactionID string `json:"transaction_id" validate:"required"`
	Reason        string `json:"reason" validate:"required,min=3,max=300"`
}

// SubmitUPIPaymentRequest allows customer to record their UPI settlement proof.
type SubmitUPIPaymentRequest struct {
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	UPIRefNo string  `json:"upi_ref_no" validate:"required,min=4,max=50"`
	Notes    string  `json:"notes,omitempty" validate:"omitempty,max=255"`
}

// RequestKhataClosureResponse is returned when a closure request is initiated.
type RequestKhataClosureResponse struct {
	KhataID     string `json:"khata_id"`
	Status      string `json:"status"` // PENDING_OTP
	Message     string `json:"message"`
	OTP         string `json:"otp,omitempty"` // shown to the other party to share
	RequestedBy string `json:"requested_by"` // SHOP or CUSTOMER
}

// VerifyKhataClosureOTPRequest is sent to finalize and close the khata.
type VerifyKhataClosureOTPRequest struct {
	OTP string `json:"otp" validate:"required,len=6"`
}

// ResolveDisputeRequest allows shopkeeper to accept or reject a customer's dispute.
type ResolveDisputeRequest struct {
	Action string `json:"action" validate:"required,oneof=ACCEPT REJECT"`
	Notes  string `json:"notes,omitempty" validate:"omitempty,max=300"`
}

// ReverseTransactionRequest allows merchant to record an official audit reversal for an erroneous entry.
type ReverseTransactionRequest struct {
	Reason string `json:"reason" validate:"required,min=3,max=300"`
}

// SetCreditOTPProtectionRequest allows customer to configure approval OTP for credit above a threshold.
type SetCreditOTPProtectionRequest struct {
	Required  bool    `json:"required"`
	Threshold float64 `json:"threshold" validate:"gte=0"`
}

// SetPromiseToPayRequest sets an agreed repayment target date and installment amount.
type SetPromiseToPayRequest struct {
	PromiseDate       string  `json:"promise_date" validate:"required"`
	InstallmentTarget float64 `json:"installment_target" validate:"gte=0"`
}

// CustomerTrustScoreResponse provides a customer's real-time credit score and repayment insights.
type CustomerTrustScoreResponse struct {
	CustomerName       string  `json:"customer_name"`
	CustomerMobile     string  `json:"customer_mobile"`
	TrustScore         int     `json:"trust_score"`
	TrustBadge         string  `json:"trust_badge"` // TRUSTED, MODERATE, HIGH_RISK
	Summary            string  `json:"summary"`
	OnTimeRate         float64 `json:"on_time_rate"`
	AverageDaysToPay   int     `json:"average_days_to_pay"`
	TotalTransactions  int     `json:"total_transactions"`
	CurrentBalance     float64 `json:"current_balance"`
	CreditLimit        float64 `json:"credit_limit"`
	CreditOTPRequired  bool    `json:"credit_otp_required"`
	CreditOTPThreshold float64 `json:"credit_otp_threshold"`
}

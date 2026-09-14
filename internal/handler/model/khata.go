// Package model handles database structs.
package model

import "time"

const (
	KhataTxTypeGiveCredit     = "GIVE_CREDIT"     // Udhar Diya
	KhataTxTypeReceivePayment = "RECEIVE_PAYMENT" // Jama Liya
	KhataTxTypeReversal       = "REVERSAL"        // Galti Sudhar / Reversal

	KhataTxStatusConfirmed        = "CONFIRMED"
	KhataTxStatusDisputed         = "DISPUTED"
	KhataTxStatusResolvedAccepted = "RESOLVED_ACCEPTED"
	KhataTxStatusResolvedRejected = "RESOLVED_REJECTED"

	KhataClosureStatusActive     = "ACTIVE"
	KhataClosureStatusPendingOTP = "PENDING_OTP"
	KhataClosureStatusClosed     = "CLOSED"
)

// CustomerKhata represents a customer credit account with a shop.
type CustomerKhata struct {
	ID                 string     `json:"id"`
	ShopID             string     `json:"shop_id"`
	CustomerID         *string    `json:"customer_id,omitempty"`
	CustomerName       string     `json:"customer_name"`
	CustomerMobile     string     `json:"customer_mobile"`
	CurrentBalance     float64    `json:"current_balance"` // How much money customer owes the shopkeeper
	CreditLimit        float64    `json:"credit_limit"`    // Max credit allowed before warning/blocking (0 = unlimited)
	CreditOTPRequired  bool       `json:"credit_otp_required"`
	CreditOTPThreshold float64    `json:"credit_otp_threshold"`
	ClosureStatus      string     `json:"closure_status"` // ACTIVE, PENDING_OTP, CLOSED
	ClosureOTP         *string    `json:"closure_otp,omitempty"`
	ClosureRequestedBy *string    `json:"closure_requested_by,omitempty"` // SHOP, CUSTOMER
	ClosureRequestedAt *time.Time `json:"closure_requested_at,omitempty"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
	PromiseToPayDate   *time.Time `json:"promise_to_pay_date,omitempty"`
	InstallmentTarget  float64    `json:"installment_target"`
	TrustScore         int        `json:"trust_score"`
	TrustBadge         string     `json:"trust_badge"` // TRUSTED, MODERATE, HIGH_RISK
	IsRegistered       bool       `json:"is_registered,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// KhataTransaction represents an entry in the customer's passbook/ledger.
type KhataTransaction struct {
	ID               string     `json:"id"`
	KhataID          string     `json:"khata_id"`
	ShopID           string     `json:"shop_id"`
	Type             string     `json:"type"` // GIVE_CREDIT, RECEIVE_PAYMENT, REVERSAL
	Amount           float64    `json:"amount"`
	BalanceAfter     float64    `json:"balance_after"`
	Notes            string     `json:"notes,omitempty"`
	BillNumber       string     `json:"bill_number,omitempty"`
	PaymentMode      string     `json:"payment_mode,omitempty"` // cash, upi, etc.
	Status           string     `json:"status"`                 // CONFIRMED, DISPUTED, RESOLVED_ACCEPTED, RESOLVED_REJECTED
	DisputeReason    string     `json:"dispute_reason,omitempty"`
	DisputedAt       *time.Time `json:"disputed_at,omitempty"`
	ResolutionAction *string    `json:"resolution_action,omitempty"` // ACCEPT, REJECT
	ResolutionNotes  *string    `json:"resolution_notes,omitempty"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	ReversalOfID     *string    `json:"reversal_of_id,omitempty"`
	UPIRefNo         string     `json:"upi_ref_no,omitempty"`
	ParchiImageURL   *string    `json:"parchi_image_url,omitempty"`
	ItemsSummary     *string    `json:"items_summary,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// KhataSummary represents total market credit balance and indebted customers count.
type KhataSummary struct {
	TotalOutstandingAmount float64 `json:"total_outstanding_amount"`
	TotalCustomers         int     `json:"total_customers"`
	TotalTransactionsCount int     `json:"total_transactions_count"`
}

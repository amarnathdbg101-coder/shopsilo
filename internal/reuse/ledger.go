package reuse

import (
	"fmt"
	"time"

	goledger "github.com/amarnathdbg101-coder/go/go-ledger"
)

// Re-export core ledger types for clean domain use.
type Ledger = goledger.Ledger
type JournalEntry = goledger.JournalEntry
type Account = goledger.Account
type AccountType = goledger.AccountType
type AgingItem = goledger.AgingItem
type AgingReport = goledger.AgingReport

// NewLedger creates a thread-safe double-entry bookkeeping ledger.
func NewLedger() *Ledger {
	return goledger.NewLedger()
}

// CalculateKhataAging categorizes outstanding khata invoices/bills into delinquency aging brackets.
func CalculateKhataAging(items []AgingItem, asOf time.Time) AgingReport {
	return goledger.CalculateAging(items, asOf)
}

// BuildKhataDoubleEntry constructs a mathematically balanced journal entry for a customer khata operation.
// Translates retail Khata transactions into standard double-entry bookkeeping legs:
// - GIVE_CREDIT (Udhar Diya): Dr Khata (Asset), Cr Sales Revenue (Revenue)
// - RECEIVE_PAYMENT (Jama Liya): Dr Cash/Bank (Asset), Cr Khata (Asset)
func BuildKhataDoubleEntry(
	txID string,
	shopID string,
	customerKhataID string,
	txType string,
	amount float64,
	paymentMode string,
	reference string,
	date time.Time,
) (*JournalEntry, error) {
	if amount <= 0 {
		return nil, goledger.ErrZeroOrNegativeAmount
	}

	khataAccount := fmt.Sprintf("khata:%s", customerKhataID)
	salesAccount := fmt.Sprintf("sales:%s", shopID)

	cashAccount := fmt.Sprintf("cash:%s", shopID)
	if paymentMode == "upi" || paymentMode == "bank" || paymentMode == "card" {
		cashAccount = fmt.Sprintf("bank:%s", shopID)
	}

	entry := goledger.NewJournalEntry(txID, fmt.Sprintf("Khata %s: %.2f", txType, amount), reference, date)

	switch txType {
	case "GIVE_CREDIT":
		// Customer owes more money to merchant
		entry.AddDebitFloat(khataAccount, amount, "Udhar given to customer")
		entry.AddCreditFloat(salesAccount, amount, "Counter credit sale revenue")
	case "RECEIVE_PAYMENT":
		// Customer pays back money
		entry.AddDebitFloat(cashAccount, amount, fmt.Sprintf("Payment received via %s", paymentMode))
		entry.AddCreditFloat(khataAccount, amount, "Customer khata credit settlement")
	default:
		return nil, fmt.Errorf("unsupported khata transaction type for double entry: %s", txType)
	}

	if err := entry.Validate(); err != nil {
		return nil, err
	}

	return entry, nil
}

package goledger

import "fmt"

// AccountType defines standard double-entry chart of accounts categories.
type AccountType string

const (
	// TypeAsset represents resources owned (Cash, Bank, Accounts Receivable/Udhar, Inventory).
	// Normal balance: DEBIT (Debits increase, Credits decrease).
	TypeAsset AccountType = "ASSET"

	// TypeLiability represents obligations owed to third parties (Accounts Payable, Supplier Credit, Loans).
	// Normal balance: CREDIT (Credits increase, Debits decrease).
	TypeLiability AccountType = "LIABILITY"

	// TypeEquity represents owner's residual interest (Capital, Retained Earnings).
	// Normal balance: CREDIT (Credits increase, Debits decrease).
	TypeEquity AccountType = "EQUITY"

	// TypeRevenue represents earnings from business operations (Sales, Service Fees).
	// Normal balance: CREDIT (Credits increase, Debits decrease).
	TypeRevenue AccountType = "REVENUE"

	// TypeExpense represents costs incurred to generate revenue (COGS, Rent, Utilities, Salaries).
	// Normal balance: DEBIT (Debits increase, Credits decrease).
	TypeExpense AccountType = "EXPENSE"
)

// Account represents a general ledger account tracking debits and credits in minor currency subunits (e.g. Paisa/Cents).
type Account struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        AccountType `json:"type"`
	Currency    string      `json:"currency"`
	TotalDebit  int64       `json:"total_debit"`  // Stored in subunits (paisa/cents)
	TotalCredit int64       `json:"total_credit"` // Stored in subunits (paisa/cents)
}

// NewAccount creates an account with initialized zero balance.
func NewAccount(id, name string, accType AccountType, currency string) (*Account, error) {
	if id == "" {
		return nil, ErrEmptyAccountID
	}
	if currency == "" {
		currency = "INR"
	}

	return &Account{
		ID:       id,
		Name:     name,
		Type:     accType,
		Currency: currency,
	}, nil
}

// BalanceSubunits calculates the net balance in minor currency subunits (paisa/cents).
// Respects normal debit/credit balances:
// - ASSET / EXPENSE: Debit - Credit
// - LIABILITY / EQUITY / REVENUE: Credit - Debit
func (a *Account) BalanceSubunits() int64 {
	switch a.Type {
	case TypeAsset, TypeExpense:
		return a.TotalDebit - a.TotalCredit
	case TypeLiability, TypeEquity, TypeRevenue:
		return a.TotalCredit - a.TotalDebit
	default:
		return a.TotalDebit - a.TotalCredit
	}
}

// BalanceFloat returns the net balance in major currency units (e.g. Rupees/Dollars).
func (a *Account) BalanceFloat() float64 {
	return float64(a.BalanceSubunits()) / 100.0
}

// String returns formatted summary.
func (a *Account) String() string {
	return fmt.Sprintf("[%s] %s (%s): %s %.2f", a.Type, a.Name, a.ID, a.Currency, a.BalanceFloat())
}

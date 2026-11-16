package goledger

import (
	"sync"
	"time"
)

// TrialBalanceRow represents an account line item in the trial balance.
type TrialBalanceRow struct {
	AccountID      string      `json:"account_id"`
	AccountName    string      `json:"account_name"`
	Type           AccountType `json:"type"`
	DebitSubunits  int64       `json:"debit_subunits"`
	CreditSubunits int64       `json:"credit_subunits"`
}

// TrialBalanceReport contains all accounts and proves global mathematical equality: Sum(Debits) == Sum(Credits).
type TrialBalanceReport struct {
	Rows                []TrialBalanceRow `json:"rows"`
	TotalDebitSubunits  int64             `json:"total_debit_subunits"`
	TotalCreditSubunits int64             `json:"total_credit_subunits"`
	IsBalanced          bool              `json:"is_balanced"`
}

// TotalDebitFloat returns total debits in major currency units.
func (t TrialBalanceReport) TotalDebitFloat() float64 { return float64(t.TotalDebitSubunits) / 100.0 }

// TotalCreditFloat returns total credits in major currency units.
func (t TrialBalanceReport) TotalCreditFloat() float64 { return float64(t.TotalCreditSubunits) / 100.0 }

// Ledger is a thread-safe, double-entry general ledger engine.
type Ledger struct {
	mu       sync.RWMutex
	accounts map[string]*Account
	entries  map[string]*JournalEntry
}

// NewLedger creates an empty general ledger.
func NewLedger() *Ledger {
	return &Ledger{
		accounts: make(map[string]*Account),
		entries:  make(map[string]*JournalEntry),
	}
}

// RegisterAccount registers a chart-of-accounts entry.
func (l *Ledger) RegisterAccount(acc *Account) error {
	if acc == nil || acc.ID == "" {
		return ErrEmptyAccountID
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.accounts[acc.ID]; exists {
		return ErrAccountAlreadyExists
	}

	// Create defensive copy
	l.accounts[acc.ID] = &Account{
		ID:          acc.ID,
		Name:        acc.Name,
		Type:        acc.Type,
		Currency:    acc.Currency,
		TotalDebit:  acc.TotalDebit,
		TotalCredit: acc.TotalCredit,
	}
	return nil
}

// GetAccount retrieves an account by ID.
func (l *Ledger) GetAccount(id string) (*Account, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	acc, exists := l.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Return copy
	return &Account{
		ID:          acc.ID,
		Name:        acc.Name,
		Type:        acc.Type,
		Currency:    acc.Currency,
		TotalDebit:  acc.TotalDebit,
		TotalCredit: acc.TotalCredit,
	}, nil
}

// Post atomically validates and records a balanced journal entry, updating respective account balances.
func (l *Ledger) Post(entry *JournalEntry) error {
	if entry == nil {
		return ErrInsufficientPostingLines
	}
	if err := entry.Validate(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.IsPosted {
		return ErrEntryAlreadyPosted
	}
	if _, exists := l.entries[entry.ID]; exists {
		return ErrEntryAlreadyPosted
	}

	// Verify all accounts exist before mutating state
	for _, line := range entry.Lines {
		if _, exists := l.accounts[line.AccountID]; !exists {
			return ErrAccountNotFound
		}
	}

	// Apply debit and credit legs
	for _, line := range entry.Lines {
		acc := l.accounts[line.AccountID]
		if line.Direction == Debit {
			acc.TotalDebit += line.AmountSubunits
		} else {
			acc.TotalCredit += line.AmountSubunits
		}
	}

	now := time.Now().UTC()
	entry.IsPosted = true
	entry.PostedAt = &now
	l.entries[entry.ID] = entry

	return nil
}

// TrialBalance generates the comprehensive trial balance report across all ledger accounts.
func (l *Ledger) TrialBalance() TrialBalanceReport {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var report TrialBalanceReport
	report.Rows = make([]TrialBalanceRow, 0, len(l.accounts))

	for _, acc := range l.accounts {
		bal := acc.BalanceSubunits()
		var dr, cr int64

		switch acc.Type {
		case TypeAsset, TypeExpense:
			if bal >= 0 {
				dr = bal
			} else {
				cr = -bal
			}
		case TypeLiability, TypeEquity, TypeRevenue:
			if bal >= 0 {
				cr = bal
			} else {
				dr = -bal
			}
		default:
			if bal >= 0 {
				dr = bal
			} else {
				cr = -bal
			}
		}

		report.TotalDebitSubunits += dr
		report.TotalCreditSubunits += cr

		report.Rows = append(report.Rows, TrialBalanceRow{
			AccountID:      acc.ID,
			AccountName:    acc.Name,
			Type:           acc.Type,
			DebitSubunits:  dr,
			CreditSubunits: cr,
		})
	}

	report.IsBalanced = (report.TotalDebitSubunits == report.TotalCreditSubunits)
	return report
}

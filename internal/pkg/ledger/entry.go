package goledger

import (
	"math"
	"time"
)

// Direction represents debit or credit movement.
type Direction string

const (
	Debit  Direction = "DEBIT"
	Credit Direction = "CREDIT"
)

// PostingLine represents a single leg within a journal entry.
type PostingLine struct {
	AccountID      string    `json:"account_id"`
	Direction      Direction `json:"direction"`
	AmountSubunits int64     `json:"amount_subunits"` // In minor subunits (paisa/cents)
	Description    string    `json:"description,omitempty"`
}

// JournalEntry represents a double-entry transaction containing balanced debits and credits.
type JournalEntry struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Reference   string        `json:"reference,omitempty"` // External invoice number, bill ID, payment txn
	Date        time.Time     `json:"date"`
	Lines       []PostingLine `json:"lines"`
	IsPosted    bool          `json:"is_posted"`
	PostedAt    *time.Time    `json:"posted_at,omitempty"`
}

// NewJournalEntry creates a new pending journal entry.
func NewJournalEntry(id, description, reference string, date time.Time) *JournalEntry {
	if date.IsZero() {
		date = time.Now().UTC()
	}
	return &JournalEntry{
		ID:          id,
		Description: description,
		Reference:   reference,
		Date:        date,
		Lines:       make([]PostingLine, 0, 2),
	}
}

// AddDebit adds a debit leg to the entry in minor subunits.
func (j *JournalEntry) AddDebit(accountID string, amountSubunits int64, description string) *JournalEntry {
	j.Lines = append(j.Lines, PostingLine{
		AccountID:      accountID,
		Direction:      Debit,
		AmountSubunits: amountSubunits,
		Description:    description,
	})
	return j
}

// AddCredit adds a credit leg to the entry in minor subunits.
func (j *JournalEntry) AddCredit(accountID string, amountSubunits int64, description string) *JournalEntry {
	j.Lines = append(j.Lines, PostingLine{
		AccountID:      accountID,
		Direction:      Credit,
		AmountSubunits: amountSubunits,
		Description:    description,
	})
	return j
}

// AddDebitFloat adds a debit leg converting major currency units (e.g. Rupees) to subunits with rounding.
func (j *JournalEntry) AddDebitFloat(accountID string, amountFloat float64, description string) *JournalEntry {
	subunits := int64(math.Round(amountFloat * 100.0))
	return j.AddDebit(accountID, subunits, description)
}

// AddCreditFloat adds a credit leg converting major currency units (e.g. Rupees) to subunits with rounding.
func (j *JournalEntry) AddCreditFloat(accountID string, amountFloat float64, description string) *JournalEntry {
	subunits := int64(math.Round(amountFloat * 100.0))
	return j.AddCredit(accountID, subunits, description)
}

// TotalDebit returns the sum of all debit legs in minor subunits.
func (j *JournalEntry) TotalDebit() int64 {
	var total int64
	for _, l := range j.Lines {
		if l.Direction == Debit {
			total += l.AmountSubunits
		}
	}
	return total
}

// TotalCredit returns the sum of all credit legs in minor subunits.
func (j *JournalEntry) TotalCredit() int64 {
	var total int64
	for _, l := range j.Lines {
		if l.Direction == Credit {
			total += l.AmountSubunits
		}
	}
	return total
}

// Validate verifies accounting invariance:
// 1. Minimum 2 legs.
// 2. All legs have valid accounts and positive amounts.
// 3. Sum of debits strictly equals sum of credits.
func (j *JournalEntry) Validate() error {
	if len(j.Lines) < 2 {
		return ErrInsufficientPostingLines
	}

	var totalDebit, totalCredit int64
	for _, l := range j.Lines {
		if l.AccountID == "" {
			return ErrEmptyAccountID
		}
		if l.AmountSubunits <= 0 {
			return ErrZeroOrNegativeAmount
		}
		if l.Direction == Debit {
			totalDebit += l.AmountSubunits
		} else if l.Direction == Credit {
			totalCredit += l.AmountSubunits
		}
	}

	if totalDebit != totalCredit {
		return ErrUnbalancedEntry
	}

	return nil
}

// CreateReversal creates a mirror contra-entry swapping debits and credits to cleanly void this transaction.
func (j *JournalEntry) CreateReversal(reversalID, reason string) (*JournalEntry, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}

	desc := "REVERSAL: " + j.Description
	if reason != "" {
		desc += " (" + reason + ")"
	}

	rev := NewJournalEntry(reversalID, desc, j.ID, time.Now().UTC())
	for _, line := range j.Lines {
		if line.Direction == Debit {
			rev.AddCredit(line.AccountID, line.AmountSubunits, line.Description)
		} else {
			rev.AddDebit(line.AccountID, line.AmountSubunits, line.Description)
		}
	}

	return rev, nil
}

package goledger

import "errors"

var (
	// ErrUnbalancedEntry is returned when total debits do not equal total credits.
	ErrUnbalancedEntry = errors.New("goledger: journal entry is unbalanced; sum of debits must equal sum of credits")

	// ErrZeroOrNegativeAmount is returned when a line has an amount <= 0.
	ErrZeroOrNegativeAmount = errors.New("goledger: posting line amount must be strictly positive")

	// ErrEmptyAccountID is returned when an account identifier is missing.
	ErrEmptyAccountID = errors.New("goledger: account ID cannot be empty")

	// ErrAccountNotFound is returned when operating on a non-existent account.
	ErrAccountNotFound = errors.New("goledger: account not found")

	// ErrAccountAlreadyExists is returned when registering a duplicate account.
	ErrAccountAlreadyExists = errors.New("goledger: account already exists")

	// ErrInsufficientPostingLines is returned when a journal entry has fewer than 2 legs.
	ErrInsufficientPostingLines = errors.New("goledger: journal entry must contain at least 2 posting lines")

	// ErrEntryAlreadyPosted is returned when attempting to modify or re-post a posted journal entry.
	ErrEntryAlreadyPosted = errors.New("goledger: journal entry has already been posted and is immutable")
)

package gomoney

import "errors"

var (
	// ErrCurrencyMismatch occurs when performing binary operations on different currencies.
	ErrCurrencyMismatch = errors.New("gomoney: currency mismatch")

	// ErrInvalidAmount occurs when parsing or allocating an invalid amount.
	ErrInvalidAmount = errors.New("gomoney: invalid monetary amount")

	// ErrInvalidCurrency occurs when currency code is unsupported or empty.
	ErrInvalidCurrency = errors.New("gomoney: invalid or unknown currency code")

	// ErrInvalidSplit occurs when splitting into non-positive number of shares.
	ErrInvalidSplit = errors.New("gomoney: parts count must be greater than zero")

	// ErrInvalidAllocation occurs when sum of allocation ratios is non-positive.
	ErrInvalidAllocation = errors.New("gomoney: allocation ratios sum must be greater than zero")
)

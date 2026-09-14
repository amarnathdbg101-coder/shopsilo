// Package utils contains shared helper functions.
package utils

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidPhoneNumber = errors.New("invalid Indian mobile number: must be 10 digits starting with 6, 7, 8, or 9")
	nonDigitRegex         = regexp.MustCompile(`[^\d]`)
)

// NormalizeIndianPhone extracts and validates a standard 10-digit Indian mobile number.
// Handles inputs like "+91 98765 43210", "09876543210", "919876543210", or "9876543210".
func NormalizeIndianPhone(raw string) (string, error) {
	cleaned := nonDigitRegex.ReplaceAllString(strings.TrimSpace(raw), "")

	// If prefixed with country code 91 (12 digits)
	if len(cleaned) == 12 && strings.HasPrefix(cleaned, "91") {
		cleaned = cleaned[2:]
	} else if len(cleaned) == 11 && strings.HasPrefix(cleaned, "0") {
		// If prefixed with trunk prefix 0 (11 digits)
		cleaned = cleaned[1:]
	}

	// Must be exactly 10 digits
	if len(cleaned) != 10 {
		return "", ErrInvalidPhoneNumber
	}

	// First digit must be 6, 7, 8, or 9
	first := cleaned[0]
	if first < '6' || first > '9' {
		return "", ErrInvalidPhoneNumber
	}

	return cleaned, nil
}

// Package utils contains shared helper functions.
package utils

import (
	"errors"

	"github.com/amarnathdbg101-coder/go/go-phone"
)

var (
	// ErrInvalidPhoneNumber defines standard validation rejection for Indian cellular numbers.
	ErrInvalidPhoneNumber = errors.New("invalid Indian mobile number: must be 10 digits starting with 6, 7, 8, or 9")
)

// NormalizeIndianPhone extracts and validates a standard 10-digit Indian mobile number using zero-dep go-phone.
// Handles inputs like "+91 98765 43210", "09876543210", "919876543210", or "9876543210".
func NormalizeIndianPhone(raw string) (string, error) {
	pn, err := phone.Parse(raw, "+91")
	if err != nil {
		return "", ErrInvalidPhoneNumber
	}

	// Must be Indian country code (+91) with exactly 10 national digits
	if pn.CountryCode != "+91" || len(pn.NationalNumber) != 10 {
		return "", ErrInvalidPhoneNumber
	}

	// In India, mobile numbers strictly start with 6, 7, 8, or 9 (DoT numbering plan)
	first := pn.NationalNumber[0]
	if first < '6' || first > '9' {
		return "", ErrInvalidPhoneNumber
	}

	return pn.NationalNumber, nil
}

// MaskIndianPhone provides privacy-compliant phone masking (e.g., "+91 98****3210") for OTP/login screens.
func MaskIndianPhone(raw string) string {
	return phone.Mask(raw, "+91")
}

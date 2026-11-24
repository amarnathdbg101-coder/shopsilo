package utils

import (
	"strings"
	"unicode"
)

// CleanBarcode removes non-printable characters, whitespace, scanner suffixes (\r, \n, \t).
func CleanBarcode(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if !unicode.IsControl(r) && !unicode.IsSpace(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// IsNumericBarcode returns true if the string consists strictly of ASCII digits.
func IsNumericBarcode(code string) bool {
	if len(code) == 0 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ValidateGS1Checksum verifies the standard Modulo-10 checksum for numeric barcodes (EAN-13, EAN-8, UPC-A, GTIN-14).
// If the barcode is alphanumeric (custom merchant SKU like "NIHAR-AMLA-200ML"), it returns true (valid SKU).
func ValidateGS1Checksum(barcode string) bool {
	clean := CleanBarcode(barcode)
	n := len(clean)

	// Non-numeric custom SKUs (e.g. NIHAR-AMLA-200ML) skip GS1 check
	if !IsNumericBarcode(clean) {
		return n >= 2 // Basic minimum length check
	}

	// Standard GS1 lengths: 8 (EAN-8), 12 (UPC-A), 13 (EAN-13), 14 (GTIN-14)
	if n != 8 && n != 12 && n != 13 && n != 14 {
		// Variable length barcode without standard checksum
		return n >= 3
	}

	// GS1 Modulo-10 Calculation:
	// From right to left (excluding check digit at index n-1):
	// odd positions from right get multiplier 3, even positions get multiplier 1.
	checkDigit := int(clean[n-1] - '0')
	sum := 0
	multiplier := 3

	for i := n - 2; i >= 0; i-- {
		digit := int(clean[i] - '0')
		sum += digit * multiplier
		if multiplier == 3 {
			multiplier = 1
		} else {
			multiplier = 3
		}
	}

	calculatedCheck := (10 - (sum % 10)) % 10
	return calculatedCheck == checkDigit
}

// GenerateBarcodeVariants generates canonical variations for resilient matching.
// Handles UPC-A (12 digits) <-> EAN-13 (13 digits with leading zero) interoperability,
// as well as lowercase/uppercase matching for alphanumeric SKUs.
func GenerateBarcodeVariants(raw string) []string {
	clean := CleanBarcode(raw)
	if clean == "" {
		return nil
	}

	seen := make(map[string]bool)
	var variants []string

	add := func(v string) {
		t := strings.TrimSpace(v)
		if t != "" && !seen[t] {
			seen[t] = true
			variants = append(variants, t)
		}
	}

	// 1. Exact cleaned input
	add(clean)

	// 2. Lowercase and Uppercase
	add(strings.ToLower(clean))
	add(strings.ToUpper(clean))

	// 3. Numeric conversions: EAN-13 vs UPC-A
	if IsNumericBarcode(clean) {
		if len(clean) == 12 {
			// UPC-A -> EAN-13 by prepending '0'
			add("0" + clean)
		} else if len(clean) == 13 && strings.HasPrefix(clean, "0") {
			// EAN-13 with leading zero -> UPC-A (12 digits)
			add(clean[1:])
		} else if len(clean) == 14 && strings.HasPrefix(clean, "00") {
			// GTIN-14 with double leading zero -> UPC-A
			add(clean[2:])
			add(clean[1:])
		}
	}

	return variants
}

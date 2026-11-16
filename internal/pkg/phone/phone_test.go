package phone

import (
	"errors"
	"testing"
)

func TestParse_InternationalFormat(t *testing.T) {
	tests := []struct {
		input       string
		expectedCC  string
		expectedNat string
		expectedE164 string
	}{
		{"+91 98765-43210", "+91", "9876543210", "+919876543210"},
		{"+1 (415) 555-2671", "+1", "4155552671", "+14155552671"},
		{"+44 20 7946 0991", "+44", "2079460991", "+442079460991"},
		{"+971 50 123 4567", "+971", "501234567", "+971501234567"},
	}

	for _, tt := range tests {
		pn, err := Parse(tt.input, "")
		if err != nil {
			t.Fatalf("failed to parse %s: %v", tt.input, err)
		}
		if pn.CountryCode != tt.expectedCC {
			t.Errorf("[%s] expected CC %s, got %s", tt.input, tt.expectedCC, pn.CountryCode)
		}
		if pn.NationalNumber != tt.expectedNat {
			t.Errorf("[%s] expected Nat %s, got %s", tt.input, tt.expectedNat, pn.NationalNumber)
		}
		if pn.E164 != tt.expectedE164 {
			t.Errorf("[%s] expected E164 %s, got %s", tt.input, tt.expectedE164, pn.E164)
		}
	}
}

func TestParse_DefaultCountryCodeAndTrunkPrefix(t *testing.T) {
	// Standard 10-digit without +
	pn1, err := Parse("9876543210", "91")
	if err != nil || pn1.E164 != "+919876543210" {
		t.Fatalf("expected +919876543210, got %+v (err: %v)", pn1, err)
	}

	// With leading 0 (trunk prefix)
	pn2, err := Parse("09876543210", "+91")
	if err != nil || pn2.E164 != "+919876543210" {
		t.Fatalf("expected +919876543210, got %+v (err: %v)", pn2, err)
	}

	// Already with CC digits included but no plus
	pn3, err := Parse("919876543210", "+91")
	if err != nil || pn3.E164 != "+919876543210" {
		t.Fatalf("expected +919876543210, got %+v (err: %v)", pn3, err)
	}

	// 10-digit number starting with 91 (e.g. 9111111111) should NOT have 91 stripped
	pn4, err := Parse("9111111111", "+91")
	if err != nil || pn4.NationalNumber != "9111111111" || pn4.E164 != "+919111111111" {
		t.Fatalf("expected 10-digit national number 9111111111, got %+v (err: %v)", pn4, err)
	}
}

func TestIsValidIndianMobile(t *testing.T) {
	validNumbers := []string{
		"+91 9876543210",
		"9876543210",
		"08765432109",
		"7001234567",
		"6234567890",
		"+91-88888-99999",
	}

	for _, v := range validNumbers {
		if !IsValidIndianMobile(v) {
			t.Errorf("expected %s to be valid Indian mobile", v)
		}
	}

	invalidNumbers := []string{
		"5123456789",       // Starts with 5
		"1234567890",       // Starts with 1
		"987654321",        // 9 digits
		"987654321000",     // 12 digits
		"+1 415 555 2671",  // US number
		"",                 // Blank
		"abcdefghij",       // Non-digits
	}

	for _, inv := range invalidNumbers {
		if IsValidIndianMobile(inv) {
			t.Errorf("expected %s to be invalid Indian mobile", inv)
		}
	}
}

func TestMask(t *testing.T) {
	masked := Mask("+91 9876543210", "+91")
	if masked != "+91 ******3210" {
		t.Errorf("expected '+91 ******3210', got '%s'", masked)
	}

	masked2 := Mask("4155552671", "+1")
	if masked2 != "+1 ******2671" {
		t.Errorf("expected '+1 ******2671', got '%s'", masked2)
	}
}

func TestFormatE164(t *testing.T) {
	formatted, err := FormatE164(" (987) 654-3210 ", "+91")
	if err != nil || formatted != "+919876543210" {
		t.Fatalf("expected +919876543210, got %s (err: %v)", formatted, err)
	}
}

func TestErrors(t *testing.T) {
	_, err := Parse("", "+91")
	if !errors.Is(err, ErrEmptyPhone) {
		t.Errorf("expected ErrEmptyPhone, got %v", err)
	}

	// Missing CC
	_, err = Parse("9876543210", "")
	if !errors.Is(err, ErrInvalidCountryCode) {
		t.Errorf("expected ErrInvalidCountryCode, got %v", err)
	}

	// Too short
	_, err = Parse("123", "+91")
	if !errors.Is(err, ErrInvalidLength) {
		t.Errorf("expected ErrInvalidLength, got %v", err)
	}
}

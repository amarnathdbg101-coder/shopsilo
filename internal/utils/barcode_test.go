package utils

import (
	"reflect"
	"testing"
)

func TestCleanBarcode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{" 8901058852233 \r\n", "8901058852233"},
		{"\tNIHAR-AMLA-200ML\n", "NIHAR-AMLA-200ML"},
		{"  012345678905  ", "012345678905"},
		{"", ""},
	}

	for _, tc := range tests {
		actual := CleanBarcode(tc.input)
		if actual != tc.expected {
			t.Errorf("CleanBarcode(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestValidateGS1Checksum(t *testing.T) {
	tests := []struct {
		barcode string
		valid   bool
	}{
		// Valid EAN-13
		{"8901058852233", true},  // Maggi
		{"8901030383786", true},  // Surf Excel
		{"8901030018800", true},  // Lifebuoy
		// Corrupted / Tampered EAN-13 (wrong last digit)
		{"8901058852238", false},
		{"8901030383780", false},
		// Valid UPC-A (12 digits)
		{"012345678905", true},
		{"012345678904", false},
		// Valid EAN-8 (8 digits)
		{"96385074", true},
		{"96385075", false},
		// Custom alphanumeric merchant SKUs (should pass)
		{"NIHAR-AMLA-200ML", true},
		{"WHITETONE-70G", true},
		{"DABUR-AMLA-180ML", true},
	}

	for _, tc := range tests {
		actual := ValidateGS1Checksum(tc.barcode)
		if actual != tc.valid {
			t.Errorf("ValidateGS1Checksum(%q) = %v; want %v", tc.barcode, actual, tc.valid)
		}
	}
}

func TestGenerateBarcodeVariants(t *testing.T) {
	// Test UPC-A -> EAN-13
	v1 := GenerateBarcodeVariants("012345678905")
	contains := func(list []string, item string) bool {
		for _, x := range list {
			if x == item {
				return true
			}
		}
		return false
	}

	if !contains(v1, "0012345678905") {
		t.Errorf("expected 012345678905 to have variant 0012345678905, got %v", v1)
	}

	// Test EAN-13 with leading zero -> UPC-A
	v2 := GenerateBarcodeVariants("0890105885223")
	if !contains(v2, "890105885223") {
		t.Errorf("expected 0890105885223 to have variant 890105885223, got %v", v2)
	}

	// Test Alphanumeric case folding
	v3 := GenerateBarcodeVariants("nihar-amla-200ml")
	if !contains(v3, "NIHAR-AMLA-200ML") {
		t.Errorf("expected uppercase variant for nihar-amla-200ml, got %v", v3)
	}

	// Empty
	if GenerateBarcodeVariants("") != nil {
		t.Errorf("expected nil for empty input")
	}
	_ = reflect.TypeOf(v1)
}

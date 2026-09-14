package utils

import (
	"testing"
)

func TestNormalizeIndianPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"+91 98765 43210", "9876543210", false},
		{"919876543210", "9876543210", false},
		{"09876543210", "9876543210", false},
		{"9876543210", "9876543210", false},
		{"7000000001", "7000000001", false},
		{"6200000002", "6200000002", false},
		{"8300000003", "8300000003", false},
		{"1234567890", "", true},     // starts with 1
		{"98765", "", true},          // too short
		{"91987654321000", "", true}, // too long
		{"abcdefghij", "", true},     // non-numeric
	}

	for _, tc := range tests {
		got, err := NormalizeIndianPhone(tc.input)
		if tc.hasErr && err == nil {
			t.Errorf("NormalizeIndianPhone(%q) expected error, got nil", tc.input)
		}
		if !tc.hasErr && err != nil {
			t.Errorf("NormalizeIndianPhone(%q) unexpected error: %v", tc.input, err)
		}
		if !tc.hasErr && got != tc.expected {
			t.Errorf("NormalizeIndianPhone(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

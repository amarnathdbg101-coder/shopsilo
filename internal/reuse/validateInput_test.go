package reuse

import (
	"testing"
)

func TestIsValidIndiaCoordinates(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lng      float64
		expected bool
	}{
		{"New Delhi", 28.6139, 77.2090, true},
		{"Mumbai", 19.0760, 72.8777, true},
		{"Bengaluru", 12.9716, 77.5946, true},
		{"Kolkata", 22.5726, 88.3639, true},
		{"Null Island (0,0)", 0.0, 0.0, false},
		{"New York (US)", 40.7128, -74.0060, false},
		{"London (UK)", 51.5074, -0.1278, false},
		{"Sydney (Australia)", -33.8688, 151.2093, false},
		{"Slightly South of Kanyakumari", 7.5, 77.5, false},
		{"Slightly North of Kashmir", 38.0, 75.0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsValidIndiaCoordinates(tc.lat, tc.lng)
			if result != tc.expected {
				t.Errorf("expected %v, got %v for coords (%f, %f)", tc.expected, result, tc.lat, tc.lng)
			}
		})
	}
}

func TestSanitizeReviewText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean review",
			input:    "Great product and quick service!",
			expected: "Great product and quick service!",
		},
		{
			name:     "Spam http link",
			input:    "Check out http://scam-store.xyz for discounts!",
			expected: "Check out [link removed] for discounts!",
		},
		{
			name:     "Spam telegram link",
			input:    "Join t.me/free_crypto now!",
			expected: "Join [link removed] now!",
		},
		{
			name:     "Spam www link",
			input:    "Visit www.fakecheapsunglasses.com today",
			expected: "Visit [link removed] today",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := SanitizeReviewText(tc.input)
			if res != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, res)
			}
		})
	}
}

func TestMaskPhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"9876543210", "98*****210"},
		{"+919876543210", "+9198*****210"},
		{"123", "123"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			res := MaskPhoneNumber(tc.input)
			if res != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, res)
			}
		})
	}
}

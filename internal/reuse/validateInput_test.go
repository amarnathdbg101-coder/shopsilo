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
		{
			name:     "XSS Script Tag Stripped",
			input:    "Loved the store! <script>alert('xss')</script> Will come again.",
			expected: "Loved the store! Will come again.",
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
		{"9876543210", "98****3210"},
		{"+919876543210", "+91-98****3210"},
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

	t.Run("MaskEmail", func(t *testing.T) {
		masked := MaskEmail("rahul.sharma@gmail.com")
		if masked != "r**********a@gmail.com" {
			t.Errorf("expected r***a@gmail.com, got %q", masked)
		}
	})
}
func TestParseAndValidateGeoParams(t *testing.T) {
	t.Run("Clean non-geo query (both empty)", func(t *testing.T) {
		res, err := ParseAndValidateGeoParams("", "", "", 10.0, 0.5, 30.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.HasGeo {
			t.Errorf("expected HasGeo=false, got true")
		}
	})

	t.Run("Valid coordinates inside India (Noida)", func(t *testing.T) {
		res, err := ParseAndValidateGeoParams("28.58284729", "77.3110291", "15.0", 10.0, 0.5, 30.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.HasGeo {
			t.Fatalf("expected HasGeo=true")
		}
		// Coordinate truncation to 5 decimal places
		if *res.Lat != 28.58285 {
			t.Errorf("expected rounded lat 28.58285, got %f", *res.Lat)
		}
		if *res.Lng != 77.31103 {
			t.Errorf("expected rounded lng 77.31103, got %f", *res.Lng)
		}
		if *res.RadiusKm != 15.0 {
			t.Errorf("expected radius 15.0, got %f", *res.RadiusKm)
		}
	})

	t.Run("Partial coordinates rejected: lat without lng", func(t *testing.T) {
		_, err := ParseAndValidateGeoParams("28.58", "", "", 10.0, 0.5, 30.0)
		if err == nil {
			t.Fatalf("expected error for partial coordinates, got nil")
		}
	})

	t.Run("Partial coordinates rejected: lng without lat", func(t *testing.T) {
		_, err := ParseAndValidateGeoParams("", "77.31", "", 10.0, 0.5, 30.0)
		if err == nil {
			t.Fatalf("expected error for partial coordinates, got nil")
		}
	})

	t.Run("Latitude out of bounds (> 90)", func(t *testing.T) {
		_, err := ParseAndValidateGeoParams("95.0", "77.31", "", 10.0, 0.5, 30.0)
		if err == nil {
			t.Fatalf("expected error for lat > 90, got nil")
		}
	})

	t.Run("Latitude out of bounds (< -90)", func(t *testing.T) {
		_, err := ParseAndValidateGeoParams("-91.0", "77.31", "", 10.0, 0.5, 30.0)
		if err == nil {
			t.Fatalf("expected error for lat < -90, got nil")
		}
	})

	t.Run("Longitude out of bounds (> 180)", func(t *testing.T) {
		_, err := ParseAndValidateGeoParams("28.58", "190.0", "", 10.0, 0.5, 30.0)
		if err == nil {
			t.Fatalf("expected error for lng > 180, got nil")
		}
	})

	t.Run("Radius clamped to maximum (DoS prevention)", func(t *testing.T) {
		res, err := ParseAndValidateGeoParams("28.58", "77.31", "5000.0", 10.0, 0.5, 30.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *res.RadiusKm != 30.0 {
			t.Errorf("expected radius clamped to 30.0, got %f", *res.RadiusKm)
		}
	})

	t.Run("Radius clamped to minimum", func(t *testing.T) {
		res, err := ParseAndValidateGeoParams("28.58", "77.31", "0.1", 10.0, 0.5, 30.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *res.RadiusKm != 0.5 {
			t.Errorf("expected radius clamped to 0.5, got %f", *res.RadiusKm)
		}
	})

	t.Run("Default radius used when empty", func(t *testing.T) {
		res, err := ParseAndValidateGeoParams("28.58", "77.31", "", 10.0, 0.5, 30.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *res.RadiusKm != 10.0 {
			t.Errorf("expected default radius 10.0, got %f", *res.RadiusKm)
		}
	})
}

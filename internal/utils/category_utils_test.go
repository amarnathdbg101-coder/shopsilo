package utils

import "testing"

func TestResolveCanonicalCategorySlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"Grocery & Staples", "kirana-grocery", true},
		{"kirana-grocery", "kirana-grocery", true},
		{"Kirana & Grocery", "kirana-grocery", true},
		{"dal-pulses", "kirana-grocery", true},
		{"Dairy, Bakery & Frozen", "dairy-breakfast", true},
		{"dairy-breakfast", "dairy-breakfast", true},
		{"Snacks & Munchies", "snacks-beverages", true},
		{"beverages-and-drinks", "snacks-beverages", true},
		{"Household & Cleaning", "household-cleaning", true},
		{"Clothing & Fashion", "clothing-apparel", true},
		{"Footwear & Leather", "footwear", true},
		{"Medicine & Healthcare", "pharmacy-healthcare", true},
		{"Hardware & Building Material", "hardware-tools", true},
		{"Toys & Baby Products", "baby-care", true},
		{"General Store", "general-store", true},
	}

	for _, tt := range tests {
		got, ok := ResolveCanonicalCategorySlug(tt.input)
		if ok != tt.found {
			t.Errorf("ResolveCanonicalCategorySlug(%q) found = %v, want %v", tt.input, ok, tt.found)
		}
		if got != tt.expected {
			t.Errorf("ResolveCanonicalCategorySlug(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatCategoryTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"spices", "Spices"},
		{"pooja samagri", "Pooja Samagri"},
		{"DAIRY PRODUCTS", "Dairy Products"},
		{"atta-dal", "Atta-Dal"},
		{"dry fruits & nuts", "Dry Fruits & Nuts"},
		{"oil/ghee", "Oil/Ghee"},
		{"  organic   pulses  ", "Organic Pulses"},
		{"kirana (ration)", "Kirana (Ration)"},
		{"", ""},
		{"a", "A"},
	}

	for _, tt := range tests {
		got := FormatCategoryTitle(tt.input)
		if got != tt.expected {
			t.Errorf("FormatCategoryTitle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}


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

func TestResolveFuzzyCategorySlug_TypoTolerance(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		found    bool
		minScore float64
	}{
		{"Typo in kirana (kirna)", "kirna", "kirana-grocery", true, 0.70},
		{"Typo in snacks (snaks)", "snaks", "snacks-beverages", true, 0.70},
		{"Typo in dairy (dary)", "dary", "dairy-breakfast", true, 0.70},
		{"Typo in bakery (bakry)", "bakry", "bakery-cakes", true, 0.70},
		{"Typo in vegetables (vegitables)", "vegitables", "fruits-vegetables", true, 0.70},
		{"Typo in footwear (footwer)", "footwer", "footwear", true, 0.70},
		{"Completely unmatched gibberish", "xyz123abc99", "", false, 0.70},
		{"Empty string", "", "", false, 0.70},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, score, ok := ResolveFuzzyCategorySlug(tt.input, tt.minScore)
			if ok != tt.found {
				t.Fatalf("ResolveFuzzyCategorySlug(%q) found = %v, want %v (score: %.3f)", tt.input, ok, tt.found, score)
			}
			if ok && got != tt.expected {
				t.Errorf("ResolveFuzzyCategorySlug(%q) = %q, want %q (score: %.3f)", tt.input, got, tt.expected, score)
			}
		})
	}
}

package retailtaxo_test

import (
	"testing"

	"shopMe/internal/pkg/retailtaxo"
)

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
		got, ok := retailtaxo.ResolveCanonicalCategorySlug(tt.input)
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
		got := retailtaxo.FormatCategoryTitle(tt.input)
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
			got, score, ok := retailtaxo.ResolveFuzzyCategorySlug(tt.input, tt.minScore)
			if ok != tt.found {
				t.Fatalf("ResolveFuzzyCategorySlug(%q) found = %v, want %v (score: %.3f)", tt.input, ok, tt.found, score)
			}
			if ok && got != tt.expected {
				t.Errorf("ResolveFuzzyCategorySlug(%q) = %q, want %q (score: %.3f)", tt.input, got, tt.expected, score)
			}
		})
	}
}

func TestGetCategorySynonyms(t *testing.T) {
	slugs := retailtaxo.GetCategorySynonymSlugs("kirana-grocery")
	if len(slugs) == 0 {
		t.Fatalf("expected synonym slugs for kirana-grocery")
	}

	names := retailtaxo.GetCategorySynonymNames("kirana-grocery")
	if len(names) == 0 {
		t.Fatalf("expected synonym names for kirana-grocery")
	}
}

func TestFindBySlugAndAll(t *testing.T) {
	cat, ok := retailtaxo.FindBySlug("kirana-grocery")
	if !ok {
		t.Fatalf("expected kirana-grocery to be found")
	}
	if cat.NameHi != "किराना व राशन" {
		t.Errorf("expected Hindi name 'किराना व राशन', got %q", cat.NameHi)
	}

	all := retailtaxo.AllCategories()
	if len(all) != 21 {
		t.Errorf("expected 21 categories, got %d", len(all))
	}
}

func BenchmarkResolveSlug_Exact(b *testing.B) {
	input := "Dairy, Bakery & Frozen"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retailtaxo.ResolveSlug(input)
	}
}

func BenchmarkResolveSlug_Fuzzy(b *testing.B) {
	input := "vegitables"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = retailtaxo.ResolveSlug(input)
	}
}

func BenchmarkFormatTitle(b *testing.B) {
	input := "organic dry fruits & nuts"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = retailtaxo.FormatTitle(input)
	}
}

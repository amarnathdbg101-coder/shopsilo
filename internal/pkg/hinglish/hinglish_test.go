package hinglish_test

import (
	"sync"
	"testing"

	hinglish "shopMe/internal/pkg/hinglish"
)

func TestCleanQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Punctuation Stripped", "Amul, Milk! 500ml?", "amul milk 500ml"},
		{"Null Bytes Removed", "tata\x00salt", "tata salt"},
		{"Excess Spaces Normalized", "   fortune    kachi   ghani   ", "fortune kachi ghani"},
		{"Special Symbols", "50% off on 1kg rice @ shop", "50 off on 1kg rice shop"},
		{"Empty String", "   ", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hinglish.CleanQuery(tc.input)
			if got != tc.expected {
				t.Fatalf("expected '%s', got '%s'", tc.expected, got)
			}
		})
	}
}

func TestNormalizePhonetic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cheeni", "chini"},
		{"doodh", "dudh"},
		{"zeera", "jira"},
		{"shakkar", "sakkar"},
		{"sooji", "suji"},
		{"paneer", "panir"},
		{"ab", "ab"}, // length < 3
	}

	for _, tc := range tests {
		got := hinglish.NormalizePhonetic(tc.input)
		if got != tc.expected {
			t.Fatalf("for '%s': expected '%s', got '%s'", tc.input, tc.expected, got)
		}
	}
}

func TestIsStopword(t *testing.T) {
	stopwords := []string{"ka", "ki", "ke", "chahiye", "bhaiya", "bhai", "wala"}
	for _, w := range stopwords {
		if !hinglish.IsStopword(w) {
			t.Fatalf("expected '%s' to be identified as stopword", w)
		}
	}

	keywords := []string{"atta", "doodh", "sugar", "milk", "oil"}
	for _, w := range keywords {
		if hinglish.IsStopword(w) {
			t.Fatalf("expected '%s' to NOT be a stopword", w)
		}
	}
}

func TestExpand(t *testing.T) {
	t.Run("Stopword filtering and synonym group expansion", func(t *testing.T) {
		groups := hinglish.Expand("amul ka doodh")
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups (amul + doodh, skipping 'ka'), got %d", len(groups))
		}

		if groups[0].Terms[0] != "amul" {
			t.Fatalf("expected group 1 to be 'amul', got %v", groups[0].Terms)
		}

		foundMilk := false
		for _, term := range groups[1].Terms {
			if term == "milk" {
				foundMilk = true
				break
			}
		}
		if !foundMilk {
			t.Fatalf("expected 'milk' in synonyms of doodh, got %v", groups[1].Terms)
		}
	})

	t.Run("Multi-word dictionary match", func(t *testing.T) {
		groups := hinglish.Expand("cold drink")
		if len(groups) != 1 {
			t.Fatalf("expected 1 group for multi-word phrase, got %d", len(groups))
		}
	})

	t.Run("Empty and whitespace queries", func(t *testing.T) {
		if groups := hinglish.Expand(""); groups != nil {
			t.Fatalf("expected nil for empty query, got %v", groups)
		}
	})
}

func TestExpandFlat(t *testing.T) {
	flat := hinglish.ExpandFlat("fortune tel")
	if len(flat) < 3 {
		t.Fatalf("expected at least 3 flat expanded terms, got %d", len(flat))
	}

	// Verify deduplication
	seen := make(map[string]bool)
	for _, term := range flat {
		if seen[term] {
			t.Fatalf("found duplicate term in flat expansion: %s", term)
		}
		seen[term] = true
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		query     string
		candidate string
		expected  bool
	}{
		{"doodh", "Amul Taaza Milk 500ml Pouch", true},
		{"cheeni", "Madhur Pure Crystal Sugar 1kg", true},
		{"tel", "Fortune Kachi Ghani Pure Mustard Oil 1L", true},
		{"chawal", "Daawat Rozana Basmati Rice 5kg", true},
		{"aloo", "Fresh Farm Aloo 1kg", true},
		{"sabun", "Dettol Original Bath Soap 75g", true},
		{"surf", "Tide Plus Double Power Detergent 1kg", true},
		{"doodh", "Surf Excel Easy Wash Detergent", false},
		{"cheeni", "Aashirvaad Shudh Chakki Atta 10kg", false},
	}

	for _, tc := range tests {
		got := hinglish.Matches(tc.query, tc.candidate)
		if got != tc.expected {
			t.Fatalf("Matches('%s', '%s'): expected %v, got %v", tc.query, tc.candidate, tc.expected, got)
		}
	}
}

func TestSuggest(t *testing.T) {
	suggestions := hinglish.Suggest("dood", 2)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for 'dood', got none")
	}

	foundExpected := false
	for _, s := range suggestions {
		if s == "doodh" || s == "dudh" {
			foundExpected = true
			break
		}
	}
	if !foundExpected {
		t.Fatalf("expected 'doodh' or 'dudh' in suggestions, got %v", suggestions)
	}
}

func TestLevenshteinDistanceAndSimilarity(t *testing.T) {
	if dist := hinglish.LevenshteinDistance("kitten", "sitting"); dist != 3 {
		t.Fatalf("expected distance 3, got %d", dist)
	}

	if sim := hinglish.Similarity("same", "same"); sim != 1.0 {
		t.Fatalf("expected similarity 1.0, got %f", sim)
	}

	if sim := hinglish.Similarity("", ""); sim != 1.0 {
		t.Fatalf("expected similarity 1.0 for empty strings, got %f", sim)
	}
}

func TestConcurrencyAndCustomRegistration(t *testing.T) {
	eng := hinglish.New()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			eng.RegisterSynonyms("mithai", "peda", "barfi", "gulab jamun")
			eng.RegisterStopword("ji")
		}(i)

		go func() {
			defer wg.Done()
			_ = eng.Expand("mithai ji")
			_ = eng.Matches("mithai", "Bikanervala Peda Box")
			_ = eng.IsStopword("ji")
		}()
	}
	wg.Wait()

	if !eng.Matches("mithai", "Bikanervala Peda Box") {
		t.Fatal("expected custom registered synonym 'peda' to match")
	}
}

func BenchmarkExpand(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hinglish.Expand("amul ka doodh aur cheeni")
	}
}

func BenchmarkMatches(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hinglish.Matches("tel", "Fortune Kachi Ghani Pure Mustard Oil 1L")
	}
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = hinglish.LevenshteinDistance("cheeni", "chini")
	}
}

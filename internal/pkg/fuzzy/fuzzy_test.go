package fuzzy

import (
	"math"
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"kitten", "sitting", 3},
		{"book", "back", 2},
		{"same", "same", 0},
		{"", "hello", 5},
		{"world", "", 5},
		{"नमस्ते", "नमस्ते!", 1}, // Unicode test
	}

	for _, tt := range tests {
		dist := Levenshtein(tt.s1, tt.s2)
		if dist != tt.expected {
			t.Errorf("Levenshtein(%q, %q) = %d; expected %d", tt.s1, tt.s2, dist, tt.expected)
		}
	}
}

func TestDamerauLevenshtein(t *testing.T) {
	// "teh" -> "the" is a single adjacent transposition
	dist := DamerauLevenshtein("teh", "the")
	if dist != 1 {
		t.Errorf("expected Damerau distance 1 for 'teh'/'the', got %d", dist)
	}

	// Normal Levenshtein would treat 'teh' -> 'the' as 2 substitutions/insertions
	levDist := Levenshtein("teh", "the")
	if levDist != 2 {
		t.Errorf("expected standard Levenshtein distance 2, got %d", levDist)
	}
}

func TestSimilarityRatio(t *testing.T) {
	// Exact
	if SimilarityRatio("golang", "golang", false) != 1.0 {
		t.Errorf("expected ratio 1.0 for exact strings")
	}

	// Case insensitive
	score := SimilarityRatio("iPhone", "iphone", true)
	if score != 1.0 {
		t.Errorf("expected ratio 1.0 with ignoreCase=true, got %f", score)
	}

	// Partial
	score2 := SimilarityRatio("kitten", "sitting", false)
	// maxLen = 7, dist = 3 -> 1 - 3/7 = 4/7 ~= 0.5714
	if math.Abs(score2-(4.0/7.0)) > 0.01 {
		t.Errorf("unexpected score for kitten/sitting: %f", score2)
	}
}

func TestJaroWinkler(t *testing.T) {
	score := JaroWinkler("martha", "marhta")
	if score < 0.94 {
		t.Errorf("expected high JaroWinkler score for martha/marhta, got %f", score)
	}

	score2 := JaroWinkler("dwayne", "duane")
	if score2 < 0.82 {
		t.Errorf("expected high score for dwayne/duane, got %f", score2)
	}

	if JaroWinkler("", "") != 1.0 {
		t.Errorf("expected 1.0 for two empty strings")
	}
}

func TestRankAndFindBestMatch(t *testing.T) {
	catalog := []string{
		"Apple iPhone 15 Pro",
		"Samsung Galaxy S24 Ultra",
		"Google Pixel 8 Pro",
		"OnePlus 12",
		"Apple iPad Air",
	}

	// Typo query: "iphne"
	best, found := FindBestMatch("iphne", catalog)
	if !found {
		t.Fatalf("expected to find a best match")
	}

	if best.Target != "Apple iPhone 15 Pro" {
		t.Errorf("expected best match 'Apple iPhone 15 Pro', got '%s'", best.Target)
	}

	// Rank with threshold
	matches := Rank("Pixel", catalog, 0.2)
	if len(matches) == 0 || matches[0].Target != "Google Pixel 8 Pro" {
		t.Errorf("expected top match to be 'Google Pixel 8 Pro', got %+v", matches)
	}
}

package services

import (
	"os"
	"testing"
	"time"
)

func init() {
	_ = os.Setenv("PORT", "8080")
}

func TestAICache(t *testing.T) {
	key := "test_cache_key"
	resp := "{\"name\":\"Sample Biscuit\",\"price\":10.0}"

	setAICache(key, resp, 100*time.Millisecond)

	val, found := getAICache(key)
	if !found {
		t.Fatalf("expected cache entry to be found")
	}
	if val != resp {
		t.Fatalf("expected %q, got %q", resp, val)
	}

	time.Sleep(150 * time.Millisecond)
	_, foundAfterExpiry := getAICache(key)
	if foundAfterExpiry {
		t.Fatalf("expected cache entry to expire")
	}
}

func TestExtractCleanJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wrapped with markdown code block",
			input:    "```json\n{\"name\": \"Parle-G\"}\n```",
			expected: "{\"name\": \"Parle-G\"}",
		},
		{
			name:     "raw json string",
			input:    "{\"price\": 20}",
			expected: "{\"price\": 20}",
		},
		{
			name:     "json with conversational surrounding text",
			input:    "Here is the product result:\n{\"name\": \"Britannia Good Day\", \"mrp\": 30.0}\nHope this helps!",
			expected: "{\"name\": \"Britannia Good Day\", \"mrp\": 30.0}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCleanJSON(tt.input)
			if got != tt.expected {
				t.Errorf("extractCleanJSON(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCleanBase64String(t *testing.T) {
	inputWithPrefix := "data:image/jpeg;base64,aGVsbG8gd29ybGQ="
	cleaned := cleanBase64String(inputWithPrefix)
	if cleaned != "aGVsbG8gd29ybGQ=" {
		t.Errorf("expected clean base64 data, got %q", cleaned)
	}

	inputRaw := "aGVsbG8gd29ybGQ="
	cleanedRaw := cleanBase64String(inputRaw)
	if cleanedRaw != "aGVsbG8gd29ybGQ=" {
		t.Errorf("expected untouched raw base64, got %q", cleanedRaw)
	}
}

func TestHashString(t *testing.T) {
	hash1 := hashString("test_query_one")
	hash2 := hashString("test_query_one")
	hash3 := hashString("test_query_two")

	if hash1 != hash2 {
		t.Errorf("expected identical hashes for identical inputs")
	}
	if hash1 == hash3 {
		t.Errorf("expected different hashes for different inputs")
	}
}

func TestIsUsefulProductAttribute(t *testing.T) {
	// Should reject useless boilerplate
	if isUsefulProductAttribute("Storage Instructions", "Store in a cool and dry place") {
		t.Errorf("expected storage instructions to be rejected")
	}
	if isUsefulProductAttribute("Usage", "Apply as directed") {
		t.Errorf("expected usage advice to be rejected")
	}
	if isUsefulProductAttribute("Warranty", "N/A") {
		t.Errorf("expected dummy placeholder 'N/A' to be rejected")
	}
	if isUsefulProductAttribute("Category", "Grocery & Staples") {
		t.Errorf("expected redundant category to be rejected")
	}

	// Should accept high-value specs
	if !isUsefulProductAttribute("Dietary Preference", "Vegetarian") {
		t.Errorf("expected Dietary Preference to be accepted")
	}
	if !isUsefulProductAttribute("Packaging Type", "Pouch") {
		t.Errorf("expected Packaging Type to be accepted")
	}
	if !isUsefulProductAttribute("Item Form", "Liquid") {
		t.Errorf("expected Item Form to be accepted")
	}
}

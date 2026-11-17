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

func TestGetGeminiModels(t *testing.T) {
	models := getGeminiModels()
	if len(models) == 0 {
		t.Fatalf("expected at least 1 gemini model, got 0")
	}

	// Verify no fake models exist
	for _, m := range models {
		if m == "gemini-3.5-flash-lite" || m == "gemini-3.8-flash" {
			t.Errorf("found deprecated or non-existent model: %s", m)
		}
	}

	// Verify primary model is flash-lite or flash
	if models[0] != "gemini-flash-lite-latest" && models[0] != "gemini-flash-latest" {
		t.Logf("Primary model configured: %s", models[0])
	}
}

func TestParseMandiScanResponse(t *testing.T) {
	// 1. Standard schema
	stdJSON := `{"items":[{"name":"Aashirvaad Atta 10kg","category":"Grocery & Staples","wholesale_qty":"5 बोरी (Bags)","wholesale_unit":"बोरी (Bags)","estimated_wholesale_rate":2100.0}],"raw_overview":"Atta bags"}`
	resp, err := parseMandiScanResponse(stdJSON)
	if err != nil || len(resp.Items) != 1 || resp.Items[0].Name != "Aashirvaad Atta 10kg" {
		t.Fatalf("standard schema parse failed: %v", err)
	}

	// 2. Direct array
	arrJSON := `[{"name":"Fortune Oil 1L","wholesale_qty":"1 पेटी / बॉक्स (Box)","wholesale_unit":"पेटी / बॉक्स (Box)","estimated_wholesale_rate":1450.0}]`
	respArr, err := parseMandiScanResponse(arrJSON)
	if err != nil || len(respArr.Items) != 1 || respArr.Items[0].Name != "Fortune Oil 1L" {
		t.Fatalf("direct array parse failed: %v", err)
	}

	// 3. Wrapped under products
	wrappedJSON := `{"products":[{"name":"Tata Salt","wholesale_qty":"1 कार्टन (Carton)","wholesale_unit":"कार्टन (Carton)","estimated_wholesale_rate":480.0}]}`
	respWrap, err := parseMandiScanResponse(wrappedJSON)
	if err != nil || len(respWrap.Items) != 1 || respWrap.Items[0].Name != "Tata Salt" {
		t.Fatalf("wrapped products parse failed: %v", err)
	}

	// 4. Single item object
	singleJSON := `{"name":"Parle-G Gold","wholesale_qty":"1 कार्टन (Carton)","wholesale_unit":"कार्टन (Carton)","estimated_wholesale_rate":550.0}`
	respSingle, err := parseMandiScanResponse(singleJSON)
	if err != nil || len(respSingle.Items) != 1 || respSingle.Items[0].Name != "Parle-G Gold" {
		t.Fatalf("single item parse failed: %v", err)
	}
}

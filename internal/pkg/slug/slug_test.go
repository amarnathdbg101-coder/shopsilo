package goslug

import (
	"testing"
)

func TestSlug_Make_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello-world"},
		{"  Go  1.23  Programming  Language--- ", "go-1-23-programming-language"},
		{"Apple iPhone 16 Pro (256 GB) - Natural Titanium", "apple-iphone-16-pro-256-gb-natural-titanium"},
		{"Special #@!% Characters & Symbols", "special-hash-at-percent-characters-and-symbols"},
		{"", ""},
		{"---", ""},
	}

	for _, tt := range tests {
		got := Make(tt.input)
		if got != tt.expected {
			t.Errorf("Make(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestSlug_Accents(t *testing.T) {
	input := "Café & Crème Brûlée in München"
	expected := "cafe-and-creme-brulee-in-munchen"
	got := Make(input)
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSlug_Devanagari(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"भारत का नया स्मार्टफोन ₹14999", "bhart-ka-nya-smartphon-inr-14999"},
		{"नमस्ते दुनिया", "nmste-duniya"},
	}

	for _, tt := range tests {
		got := Make(tt.input)
		if got != tt.expected {
			t.Errorf("Make(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestSlug_MaxLength_WordBoundary(t *testing.T) {
	input := "The quick brown fox jumps over the lazy dog"
	// Truncate at 20 chars:
	// "the-quick-brown-fox-" -> should cut cleanly at "the-quick-brown-fox"
	cfg := DefaultConfig
	cfg.MaxLength = 20
	got := MakeWithConfig(input, cfg)
	expected := "the-quick-brown-fox"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSlug_CustomSeparator(t *testing.T) {
	cfg := DefaultConfig
	cfg.Separator = "_"
	got := MakeWithConfig("Database Migration Script 001", cfg)
	expected := "database_migration_script_001"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestIsValidSlug(t *testing.T) {
	valid := []string{"hello-world", "go-lang-123", "a-b-c"}
	for _, s := range valid {
		if !IsValidSlug(s, "-") {
			t.Errorf("expected %q to be valid slug", s)
		}
	}

	invalid := []string{
		"-starts-with-dash",
		"ends-with-dash-",
		"has--double-dash",
		"Has-Upper-Case",
		"has spaces",
		"",
	}
	for _, s := range invalid {
		if IsValidSlug(s, "-") {
			t.Errorf("expected %q to be invalid slug", s)
		}
	}
}

func BenchmarkMake(b *testing.B) {
	input := "Apple iPhone 16 Pro Max 256GB - Desert Titanium at ₹1,44,900"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Make(input)
	}
}

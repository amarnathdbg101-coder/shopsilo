package gosanitize

import (
	"strings"
	"testing"
)

func TestStripHTML(t *testing.T) {
	input := `<p>Hello <b>World</b>! <script>alert("hack");</script>Visit <a href="https://example.com">here</a> &amp; enjoy.</p>`
	expected := `Hello World! Visit here & enjoy.`

	got := StripHTML(input)
	if got != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, got)
	}
}

func TestSanitizeHTML_XSS_Neutralization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Removes script tags and contents",
			input:    `<p>Normal text <script>alert('xss')</script> continues</p>`,
			expected: `<p>Normal text  continues</p>`,
		},
		{
			name:     "Removes iframe and contents",
			input:    `<div>Check this <iframe src="http://evil.com">evil</iframe> out</div>`,
			expected: `Check this  out`,
		},
		{
			name:     "Neutralizes javascript: URL in href",
			input:    `<a href="javascript:alert('pwned')">Click here</a>`,
			expected: `<a>Click here</a>`,
		},
		{
			name:     "Preserves valid https: URL in href",
			input:    `<a href="https://shopsilo.com">ShopSilo</a>`,
			expected: `<a href="https://shopsilo.com">ShopSilo</a>`,
		},
		{
			name:     "Drops unauthorized onerror attributes",
			input:    `<p>Image coming: <img src="x" onerror="alert(1)"></p>`,
			expected: `<p>Image coming: </p>`, // img not in UGCPolicy
		},
		{
			name:     "Keeps safe formatting",
			input:    `<b>Bold</b> and <i>Italic</i>`,
			expected: `<b>Bold</b> and <i>Italic</i>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeHTML(tt.input, UGCPolicy)
			if got != tt.expected {
				t.Fatalf("SanitizeHTML failed:\nExpected: %s\nGot:      %s", tt.expected, got)
			}
		})
	}
}

func TestEscapeSQLWildcards(t *testing.T) {
	input := "50% discount on item_10"
	expected := `50\% discount on item\_10`
	got := EscapeSQLWildcards(input)
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestCleanUnicode(t *testing.T) {
	// Zero-width space (\u200B) between letters: "A\u200Bd\u200Bi\u200Bt\u200By\u200Ba"
	input := "A\u200Bd\u200Bi\u200Bt\u200By\u200Ba"
	got := CleanUnicode(input)
	if got != "Aditya" {
		t.Fatalf("expected 'Aditya' after stripping zero-width runes, got %q (len %d)", got, len(got))
	}

	// Bidi override rune \u202E
	bidiInput := "admin\u202Egnp.exe"
	cleanBidi := CleanUnicode(bidiInput)
	if strings.ContainsRune(cleanBidi, '\u202E') {
		t.Fatal("bidi override character should have been dropped")
	}
}

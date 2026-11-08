package reuse

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ramesh Electronics", "ramesh-electronics"},
		{"Ramesh & Sons Co., Ltd.!", "ramesh-and-sons-co-ltd"},
		{"   Multiple   Spaces   ", "multiple-spaces"},
		{"---Leading-and-trailing---", "leading-and-trailing"},
		{"Special @#$ Characters", "special-at-hash-usd-characters"},
		{"श्री राम किराना स्टोर", "shri-ram-kirana-stor"},
		{"₹500 Discount Hub", "inr-500-discount-hub"},
		{"", "shop"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

package reuse

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ramesh Electronics", "ramesh-electronics"},
		{"Ramesh & Sons Co., Ltd.!", "ramesh-sons-co-ltd"},
		{"   Multiple   Spaces   ", "multiple-spaces"},
		{"---Leading-and-trailing---", "leading-and-trailing"},
		{"Special @#$ Characters", "special-characters"},
		{"", "shop"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

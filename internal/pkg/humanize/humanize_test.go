package humanize

import (
	"testing"
	"time"
)

func TestFormatComma(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1000, "1,000"},
		{1250000, "1,250,000"},
		{-9876543, "-9,876,543"},
		{0, "0"},
	}

	for _, tt := range tests {
		res := FormatComma(tt.input)
		if res != tt.expected {
			t.Errorf("FormatComma(%d) = %s; expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestFormatIndian(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1000, "1,000"},
		{10000, "10,000"},
		{100000, "1,00,000"},       // 1 Lakh
		{1250000, "12,50,000"},     // 12.5 Lakhs
		{10000000, "1,00,00,000"},  // 1 Crore
		{123456789, "12,34,56,789"},
		{-1500000, "-15,00,000"},
	}

	for _, tt := range tests {
		res := FormatIndian(tt.input)
		if res != tt.expected {
			t.Errorf("FormatIndian(%d) = %s; expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestCompact(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1000, "1K"},
		{1500, "1.5K"},
		{2000000, "2M"},
		{2500000, "2.5M"},
		{3800000000, "3.8B"},
		{-1500, "-1.5K"},
	}

	for _, tt := range tests {
		res := Compact(tt.input)
		if res != tt.expected {
			t.Errorf("Compact(%d) = %s; expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestCompactIndian(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{2500, "2.5 K"},
		{150000, "1.5 Lakh"},
		{20000000, "2 Cr"},
		{35000000, "3.5 Cr"},
	}

	for _, tt := range tests {
		res := CompactIndian(tt.input)
		if res != tt.expected {
			t.Errorf("CompactIndian(%d) = %s; expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestOrdinal(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{1, "1st"},
		{2, "2nd"},
		{3, "3rd"},
		{4, "4th"},
		{11, "11th"},
		{12, "12th"},
		{13, "13th"},
		{21, "21st"},
		{22, "22nd"},
		{23, "23rd"},
		{101, "101st"},
		{111, "111th"},
	}

	for _, tt := range tests {
		res := Ordinal(tt.input)
		if res != tt.expected {
			t.Errorf("Ordinal(%d) = %s; expected %s", tt.input, res, tt.expected)
		}
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	if TimeAgo(now.Add(-5*time.Second)) != "just now" {
		t.Errorf("expected 'just now'")
	}
	if TimeAgo(now.Add(-5*time.Minute)) != "5 minutes ago" {
		t.Errorf("expected '5 minutes ago'")
	}
	if TimeAgo(now.Add(-3*time.Hour)) != "3 hours ago" {
		t.Errorf("expected '3 hours ago'")
	}
	if TimeAgo(now.Add(-30*time.Hour)) != "yesterday" {
		t.Errorf("expected 'yesterday'")
	}
}

func TestPlural(t *testing.T) {
	if Plural(1, "order", "orders") != "1 order" {
		t.Errorf("expected '1 order'")
	}
	if Plural(5, "order", "orders") != "5 orders" {
		t.Errorf("expected '5 orders'")
	}
	if Plural(0, "item", "items") != "0 items" {
		t.Errorf("expected '0 items'")
	}
}

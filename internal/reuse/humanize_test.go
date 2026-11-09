package reuse_test

import (
	"strings"
	"testing"
	"time"

	"shopMe/internal/reuse"
)

func TestHumanizeIndianRupees(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "\u20b90"},
		{500, "\u20b9500"},
		{1500, "\u20b91,500"},
		{100000, "\u20b91,00,000"},
		{1250000.50, "\u20b912,50,000.50"},
		{10000000, "\u20b91,00,00,000"},
		{-25000, "-\u20b925,000"},
	}

	for _, tt := range tests {
		got := reuse.HumanizeIndianRupees(tt.input)
		if got != tt.expected {
			t.Errorf("HumanizeIndianRupees(%.2f) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestHumanizeIndianCompact(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{50000, "\u20b950 K"},
		{150000, "\u20b91.5 Lakh"},
		{25000000, "\u20b92.5 Cr"},
	}

	for _, tt := range tests {
		got := reuse.HumanizeIndianCompact(tt.input)
		if got != tt.expected {
			t.Errorf("HumanizeIndianCompact(%.2f) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestHumanizeTimeAgo(t *testing.T) {
	now := time.Now()
	justNow := reuse.HumanizeTimeAgo(now.Add(-2 * time.Second))
	if !strings.Contains(justNow, "now") {
		t.Errorf("expected relative time 'just now', got %q", justNow)
	}

	minsAgo := reuse.HumanizeTimeAgo(now.Add(-10 * time.Minute))
	if !strings.Contains(minsAgo, "minute") {
		t.Errorf("expected relative minutes ago, got %q", minsAgo)
	}
}

func TestHumanizeItemCount(t *testing.T) {
	if got := reuse.HumanizeItemCount(1, "item", "items"); got != "1 item" {
		t.Errorf("expected '1 item', got %q", got)
	}
	if got := reuse.HumanizeItemCount(5, "item", "items"); got != "5 items" {
		t.Errorf("expected '5 items', got %q", got)
	}
}

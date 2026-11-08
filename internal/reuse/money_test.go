package reuse_test

import (
	"strings"
	"testing"

	"shopMe/internal/reuse"
)

func TestMoneyUtilities(t *testing.T) {
	t.Run("FormatINR formats correctly", func(t *testing.T) {
		formatted := reuse.FormatINR(1499.50)
		if !strings.Contains(formatted, "1,499.50") && !strings.Contains(formatted, "1499.50") {
			t.Fatalf("unexpected formatted string: %s", formatted)
		}
	})

	t.Run("CalculateGST with exact paisa rounding", func(t *testing.T) {
		// ₹100.00 with 18% GST = ₹18.00 tax, ₹118.00 total
		tax, total := reuse.CalculateGST(100.0, 18.0)
		if tax != 18.0 {
			t.Errorf("expected 18.0 tax, got %.2f", tax)
		}
		if total != 118.0 {
			t.Errorf("expected 118.0 total, got %.2f", total)
		}
	})

	t.Run("AllocateBillSplits preserves every paisa without rounding loss", func(t *testing.T) {
		// ₹100.00 split 3 ways must total exactly ₹100.00
		splits, err := reuse.AllocateBillSplits(100.0, 3)
		if err != nil {
			t.Fatalf("unexpected error allocating splits: %v", err)
		}

		if len(splits) != 3 {
			t.Fatalf("expected 3 splits, got %d", len(splits))
		}

		var sum float64
		for _, s := range splits {
			sum += s
		}

		if sum != 100.0 {
			t.Errorf("splits sum mismatch: expected 100.0, got %.4f (splits: %v)", sum, splits)
		}
		// First split gets the remainder paisa: ₹33.34
		if splits[0] != 33.34 || splits[1] != 33.33 || splits[2] != 33.33 {
			t.Errorf("expected [33.34, 33.33, 33.33], got %v", splits)
		}
	})
}

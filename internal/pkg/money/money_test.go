package gomoney

import (
	"encoding/json"
	"testing"
)

func TestMoney_Creation(t *testing.T) {
	// From Subunits
	m1, err := NewFromSubunits(149950, "INR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m1.Amount() != 149950 || m1.Float64() != 1499.50 {
		t.Errorf("expected 1499.50, got %f", m1.Float64())
	}

	// From Float
	m2, err := NewFromFloat(19.99, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2.Amount() != 1999 {
		t.Errorf("expected 1999 cents, got %d", m2.Amount())
	}

	// From String
	m3, err := NewFromString("500.25", "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m3.Amount() != 50025 {
		t.Errorf("expected 50025 cents, got %d", m3.Amount())
	}

	// Zero scale currency (JPY)
	mJpy, err := NewFromFloat(500.0, "JPY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mJpy.Amount() != 500 {
		t.Errorf("expected 500 yen, got %d", mJpy.Amount())
	}
}

func TestMoney_Arithmetic(t *testing.T) {
	m1, _ := NewFromFloat(100.50, "INR")
	m2, _ := NewFromFloat(49.50, "INR")

	sum, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if sum.Amount() != 15000 {
		t.Errorf("expected 15000, got %d", sum.Amount())
	}

	diff, err := m1.Sub(m2)
	if err != nil {
		t.Fatalf("Sub failed: %v", err)
	}
	if diff.Amount() != 5100 {
		t.Errorf("expected 5100, got %d", diff.Amount())
	}

	mul := m1.Mul(2)
	if mul.Amount() != 20100 {
		t.Errorf("expected 20100, got %d", mul.Amount())
	}

	mulFloat := m1.MulFloat(1.18) // 18% GST on 100.50 = 118.59
	if mulFloat.Amount() != 11859 {
		t.Errorf("expected 11859, got %d", mulFloat.Amount())
	}

	// Mismatched currency error check
	usd, _ := NewFromFloat(10.0, "USD")
	if _, err := m1.Add(usd); err == nil {
		t.Errorf("expected error on currency mismatch")
	}
}

func TestMoney_SplitNoLoss(t *testing.T) {
	// Splitting 10000 paise (₹100) into 3 parts:
	// Should be 3334, 3333, 3333 (Sum = 10000 exactly)
	m, _ := NewFromSubunits(10000, "INR")
	splits, err := m.Split(3)
	if err != nil {
		t.Fatalf("Split failed: %v", err)
	}

	if len(splits) != 3 {
		t.Fatalf("expected 3 splits, got %d", len(splits))
	}

	total := int64(0)
	for _, s := range splits {
		total += s.Amount()
	}
	if total != 10000 {
		t.Errorf("split lost or gained money! expected 10000, got %d", total)
	}
	if splits[0].Amount() != 3334 || splits[1].Amount() != 3333 || splits[2].Amount() != 3333 {
		t.Errorf("unexpected split distribution: %v", splits)
	}
}

func TestMoney_Allocate(t *testing.T) {
	// Allocate 100 cents according to ratios 1 : 2 : 1
	// 25, 50, 25 (Sum = 100)
	m, _ := NewFromSubunits(100, "USD")
	allocated, err := m.Allocate(1, 2, 1)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	total := int64(0)
	for _, a := range allocated {
		total += a.Amount()
	}
	if total != 100 {
		t.Errorf("allocation lost money! expected 100, got %d", total)
	}
	if allocated[0].Amount() != 25 || allocated[1].Amount() != 50 || allocated[2].Amount() != 25 {
		t.Errorf("unexpected allocation: %v", allocated)
	}
}

func TestMoney_Format(t *testing.T) {
	inr, _ := NewFromFloat(1499.50, "INR")
	if formatted := inr.Format(); formatted != "₹1,499.50" {
		t.Errorf("expected '₹1,499.50', got %q", formatted)
	}

	usd, _ := NewFromFloat(1234567.89, "USD")
	if formatted := usd.Format(); formatted != "$1,234,567.89" {
		t.Errorf("expected '$1,234,567.89', got %q", formatted)
	}

	jpy, _ := NewFromSubunits(5000, "JPY")
	if formatted := jpy.Format(); formatted != "¥5,000" {
		t.Errorf("expected '¥5,000', got %q", formatted)
	}
}

func TestMoney_JSON(t *testing.T) {
	m, _ := NewFromFloat(99.99, "USD")
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var parsed Money
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if parsed.Amount() != 9999 || parsed.CurrencyCode() != "USD" {
		t.Errorf("unmarshaled mismatch: amount %d, curr %s", parsed.Amount(), parsed.CurrencyCode())
	}
}

func TestMoney_SQLDriver(t *testing.T) {
	m, _ := NewFromSubunits(45000, "INR")
	val, err := m.Value()
	if err != nil {
		t.Fatalf("Value failed: %v", err)
	}
	if val.(int64) != 45000 {
		t.Errorf("expected 45000, got %v", val)
	}

	var scanned Money
	if err := scanned.Scan(int64(75000)); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if scanned.Amount() != 75000 {
		t.Errorf("expected 75000, got %d", scanned.Amount())
	}
}

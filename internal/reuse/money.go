package reuse

import (
	"fmt"
	"math"

	gomoney "github.com/amarnathdbg101-coder/go/go-money"
)

// INR creates a Money instance in Indian Rupees from a standard decimal float.
func INR(rupees float64) gomoney.Money {
	m, err := gomoney.NewFromFloat(rupees, "INR")
	if err != nil {
		// Fallback safe zero
		inr, _ := gomoney.GetCurrency("INR")
		return gomoney.New(0, inr)
	}
	return m
}

// FormatINR formats a decimal rupee amount into a clean currency string (e.g. "₹1,499.50").
func FormatINR(rupees float64) string {
	return INR(rupees).Format()
}

// CalculateGST computes exact GST tax and total using integer paisa math to avoid 1-paisa rounding bugs.
func CalculateGST(subtotal float64, gstPercentage float64) (gstAmount float64, totalAmount float64) {
	sub := INR(subtotal)
	rate := gstPercentage / 100.0

	// Compute tax rounded to nearest paisa
	taxSubunits := int64(math.Round(float64(sub.Amount()) * rate))
	inr, _ := gomoney.GetCurrency("INR")
	tax := gomoney.New(taxSubunits, inr)

	total, _ := sub.Add(tax)
	return tax.Float64(), total.Float64()
}

// AllocateBillSplits divides total money across multiple parts without losing or gaining a single paisa.
// Solves the classic split payment problem: ₹100 split 3 ways becomes [₹33.34, ₹33.33, ₹33.33], sum is strictly ₹100.00!
func AllocateBillSplits(total float64, parts int) ([]float64, error) {
	if parts <= 0 {
		return nil, fmt.Errorf("parts must be greater than 0")
	}

	m := INR(total)
	ratios := make([]int, parts)
	for i := 0; i < parts; i++ {
		ratios[i] = 1
	}

	splits, err := m.Allocate(ratios...)
	if err != nil {
		return nil, err
	}

	result := make([]float64, len(splits))
	for i, s := range splits {
		result[i] = s.Float64()
	}
	return result, nil
}

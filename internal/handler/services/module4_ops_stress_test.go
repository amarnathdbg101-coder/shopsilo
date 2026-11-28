package services

import (
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"strings"
	"testing"
)

func TestModule4OpsValidation_2000StressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(2026112804))

	// 1. 700 Expense validation cases
	payMethods := []string{"", "cash", "UPI", "bank", "card", "online", "other", "crypto", "barter"}
	for i := 0; i < 700; i++ {
		req := dto.CreateExpenseRequest{
			Category:      "Rent",
			Amount:        float64(rng.Intn(5000)-200) + 0.5,
			PaymentMethod: payMethods[i%len(payMethods)],
			ExpenseDate:   "2026-11-28",
		}
		switch i % 7 {
		case 0:
			req.Category = "   "
		case 1:
			req.Amount = 0
		case 2:
			req.Amount = math.NaN()
		case 3:
			req.ExpenseDate = "28-11-2026"
		case 4:
			req.ExpenseDate = ""
		}

		err := validateCreateExpenseInput(&req)
		pm := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
		validPM := pm == "" || pm == "cash" || pm == "upi" || pm == "bank" || pm == "card" || pm == "online" || pm == "other"
		validDate := req.ExpenseDate == "" || req.ExpenseDate == "2026-11-28"
		expectErr := strings.TrimSpace(req.Category) == "" ||
			math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) || req.Amount <= 0 ||
			!validPM || !validDate

		if expectErr && err == nil {
			t.Fatalf("expense case %d: expected error for req=%+v", i, req)
		}
		if !expectErr && err != nil {
			t.Fatalf("expense case %d: unexpected error: %v", i, err)
		}
	}

	// 2. 700 Cash Drawer & Procurement validation cases
	for i := 0; i < 700; i++ {
		opening := float64(rng.Intn(10000) - 500)
		var thr *float64
		if i%5 == 0 {
			v := float64(rng.Intn(5000) - 500)
			thr = &v
		}
		if i%11 == 0 {
			opening = math.NaN()
		}
		err := validateOpenDrawerInput(dto.OpenDrawerSessionRequest{
			OpeningCash:       opening,
			SafeDropThreshold: thr,
		})
		badOpen := math.IsNaN(opening) || math.IsInf(opening, 0) || opening < 0 ||
			(thr != nil && (math.IsNaN(*thr) || math.IsInf(*thr, 0) || *thr <= 0))
		if badOpen && err == nil {
			t.Fatalf("drawer open case %d: expected error", i)
		}
		if !badOpen && err != nil {
			t.Fatalf("drawer open case %d: unexpected error: %v", i, err)
		}

		amt := float64(rng.Intn(5000) - 200)
		if i%9 == 0 {
			amt = math.Inf(-1)
		}
		amtErr := validateDrawerAmount(amt, "test")
		if (math.IsNaN(amt) || math.IsInf(amt, 0) || amt <= 0) != (amtErr != nil) {
			t.Fatalf("drawer amount case %d: mismatch for %v", i, amt)
		}

		est := float64(rng.Intn(1000) - 50)
		act := float64(rng.Intn(1000) - 50)
		procErr := validateProcurementPrices(est, act)
		if (est < 0 || act < 0) != (procErr != nil) {
			t.Fatalf("procurement price case %d: mismatch est=%v act=%v", i, est, act)
		}
	}

	// 3. 600 Damage Claim validation cases
	for i := 0; i < 600; i++ {
		prodID := fmt.Sprintf("p-%d", i)
		var expPtr *string
		if i%6 == 0 {
			badDate := "not-a-date"
			expPtr = &badDate
		} else if i%6 == 1 {
			goodDate := "2026-12-31"
			expPtr = &goodDate
		}

		req := dto.CreateDamageClaimRequest{
			DistributorName: "Metro Wholesale",
			ProductID:       &prodID,
			ProductName:     "Milk",
			Quantity:        rng.Intn(20) - 2,
			UnitCost:        float64(rng.Intn(500) - 20),
			ExpiryDate:      expPtr,
		}
		if i%8 == 0 {
			req.DistributorName = ""
		}
		if i%10 == 0 {
			req.ProductName = ""
			req.ProductID = nil
		}

		_, err := validateCreateDamageClaimInput(req)
		badDate := expPtr != nil && *expPtr == "not-a-date"
		expectErr := strings.TrimSpace(req.DistributorName) == "" ||
			(strings.TrimSpace(req.ProductName) == "" && req.ProductID == nil) ||
			req.Quantity <= 0 ||
			req.UnitCost < 0 ||
			badDate

		if expectErr && err == nil {
			t.Fatalf("damage claim case %d: expected error for req=%+v", i, req)
		}
		if !expectErr && err != nil {
			t.Fatalf("damage claim case %d: unexpected error: %v", i, err)
		}
	}
}

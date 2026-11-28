package services

import (
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"strings"
	"testing"
)

func TestModule5AuthStaffLoyalty_2000StressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(2026112805))

	// 1. 700 Staff PIN & CreateStaff validation cases (catches non-numeric 4-char PINs like "abcd"!)
	badPINs := []string{"", "123", "12345", "abcd", "12a4", "----", " 12 "}
	for i := 0; i < 700; i++ {
		pin := fmt.Sprintf("%04d", rng.Intn(10000))
		isBadPIN := false
		if i%4 == 0 {
			pin = badPINs[i%len(badPINs)]
			isBadPIN = true
		}

		req := dto.CreateStaffRequest{
			FullName: "Ramesh Cashier",
			Phone:    "9876543210",
			PIN:      pin,
			Role:     "",
		}
		if i%9 == 0 {
			req.FullName = "   "
		}
		if i%11 == 0 {
			req.Phone = ""
		}

		_, err := validateCreateStaffInput(&req)
		expectErr := isBadPIN || strings.TrimSpace(req.FullName) == "" || strings.TrimSpace(req.Phone) == ""
		if expectErr && err == nil {
			t.Fatalf("staff case %d: expected error for req=%+v", i, req)
		}
		if !expectErr && err != nil {
			t.Fatalf("staff case %d: unexpected error: %v", i, err)
		}
		if err == nil && req.Role == "" {
			t.Fatalf("staff case %d: expected default role", i)
		}
	}

	// 2. 650 Return input validation cases
	for i := 0; i < 650; i++ {
		in := dto.CreateReturnRequest{
			ProductID:    fmt.Sprintf("prod-%d", i),
			Quantity:     rng.Intn(20) - 3,
			RefundAmount: float64(rng.Intn(500) - 25),
		}
		if i%7 == 0 {
			in.ProductID = "  "
		}
		if i%13 == 0 {
			in.RefundAmount = math.NaN()
		}
		err := validateReturnInput(in)
		expectErr := strings.TrimSpace(in.ProductID) == "" ||
			in.Quantity <= 0 ||
			math.IsNaN(in.RefundAmount) || math.IsInf(in.RefundAmount, 0) || in.RefundAmount < 0
		if expectErr != (err != nil) {
			t.Fatalf("return case %d: mismatch for %+v (err=%v)", i, in, err)
		}
	}

	// 3. 650 Offer terms validation cases
	for i := 0; i < 650; i++ {
		title := "Festive Deal"
		discType := "percentage"
		if i%2 == 0 {
			discType = "flat"
		}
		discVal := float64(rng.Intn(130) - 10)
		minOrder := float64(rng.Intn(500) - 20)
		maxDisc := float64(rng.Intn(300) - 10)
		minPts := rng.Intn(100) - 10
		if i%8 == 0 {
			title = "  "
		}
		if i%15 == 0 {
			discVal = math.Inf(1)
		}

		err := validateOfferTerms(title, discType, discVal, minOrder, maxDisc, minPts)
		expectErr := strings.TrimSpace(title) == "" ||
			math.IsNaN(discVal) || math.IsInf(discVal, 0) || discVal < 0 ||
			(discType == "percentage" && discVal > 100) ||
			minOrder < 0 || maxDisc < 0 || minPts < 0
		if expectErr != (err != nil) {
			t.Fatalf("offer case %d: mismatch (type=%s val=%v minO=%v maxD=%v pts=%d err=%v)",
				i, discType, discVal, minOrder, maxDisc, minPts, err)
		}
	}
}

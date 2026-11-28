package services

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"shopMe/internal/handler/dto"
)

func TestKhataValidation_2000PredictableAndUnpredictableInputs(t *testing.T) {
	svc := &KhataService{}
	ctx := context.Background()
	rng := rand.New(rand.NewSource(777))

	badAmounts := []float64{0, -0.01, -100, -999999.99, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, amt := range badAmounts {
		if err := validatePositiveAmount(amt); err == nil {
			t.Fatalf("expected error for invalid amount %v", amt)
		}
		if _, err := svc.RecordCredit(ctx, "owner-1", dto.RecordCreditTransactionRequest{
			CustomerMobile: "9876543210",
			Amount:         amt,
		}); err == nil {
			t.Fatalf("RecordCredit should reject amount %v", amt)
		}
		if _, err := svc.RecordPayment(ctx, "owner-1", "9876543210", dto.RecordPaymentRequest{
			Amount: amt,
		}); err == nil {
			t.Fatalf("RecordPayment should reject amount %v", amt)
		}
	}

	// Empty mobile rejection
	if _, err := svc.RecordCredit(ctx, "owner-1", dto.RecordCreditTransactionRequest{
		CustomerMobile: "   ",
		Amount:         100,
	}); err == nil {
		t.Fatal("RecordCredit should reject empty mobile")
	}
	if _, err := svc.RecordPayment(ctx, "owner-1", "  ", dto.RecordPaymentRequest{
		Amount: 100,
	}); err == nil {
		t.Fatal("RecordPayment should reject empty mobile")
	}

	// 2,000 randomized credit limit & positive amount checks
	for i := 0; i < 2000; i++ {
		val := (rng.Float64() - 0.5) * 100000
		if i%50 == 0 {
			val = math.NaN()
		} else if i%75 == 0 {
			val = math.Inf(1)
		}

		amtErr := validatePositiveAmount(val)
		if (math.IsNaN(val) || math.IsInf(val, 0) || val <= 0) && amtErr == nil {
			t.Fatalf("iter %d: expected validation error for amount %v", i, val)
		}
		if !math.IsNaN(val) && !math.IsInf(val, 0) && val > 0 && amtErr != nil {
			t.Fatalf("iter %d: unexpected validation error for valid amount %v: %v", i, val, amtErr)
		}

		if math.IsNaN(val) || math.IsInf(val, 0) || val < 0 {
			if err := svc.UpdateCreditLimit(ctx, "owner-1", "9876543210", val); err == nil {
				t.Fatalf("iter %d: UpdateCreditLimit should reject %v", i, val)
			}
		}
	}
}

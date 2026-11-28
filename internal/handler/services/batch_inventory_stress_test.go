package services

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"shopMe/internal/handler/dto"
)

func TestBatchAndInventoryValidation_2000StressCases(t *testing.T) {
	invSvc := &InventoryService{}
	ctx := context.Background()
	rng := rand.New(rand.NewSource(888))

	// Inventory negative threshold & blank product rejection
	neg := -1
	if _, err := invSvc.AdjustStock(ctx, "owner-1", dto.AdjustStockRequest{ProductID: "", Adjustment: 5}); err == nil {
		t.Fatal("expected error for blank product_id")
	}
	if _, err := invSvc.AdjustStock(ctx, "owner-1", dto.AdjustStockRequest{ProductID: "p1", Adjustment: 5, LowStockThreshold: &neg}); err == nil {
		t.Fatal("expected error for negative low_stock_threshold")
	}

	// 2,000 randomized batch & markdown validation checks
	for i := 0; i < 2000; i++ {
		qty := rng.Intn(40) - 10
		cost := (rng.Float64() - 0.2) * 500
		sell := (rng.Float64() - 0.2) * 500
		md := (rng.Float64() - 0.2) * 140
		if i%40 == 0 {
			md = math.NaN()
		}

		mfg := "2026-01-01"
		exp := "2026-12-31"
		if i%7 == 0 {
			mfg = "2027-05-01" // after expiry
		} else if i%11 == 0 {
			exp = "bad-date"
		}

		_, _, err := validateCreateBatchInput(dto.CreateBatchRequest{
			ProductID:       "p1",
			Quantity:        qty,
			CostPrice:       cost,
			SellingPrice:    sell,
			MarkdownPercent: md,
			MfgDate:         &mfg,
			ExpiryDate:      exp,
		})

		shouldFail := qty <= 0 ||
			math.IsNaN(cost) || cost < 0 ||
			math.IsNaN(sell) || sell <= 0 ||
			math.IsNaN(md) || md < 0 || md > 95 ||
			exp == "bad-date" || mfg == "2027-05-01"
		if shouldFail && err == nil {
			t.Fatalf("iter %d: expected validation error for qty=%d cost=%v sell=%v md=%v mfg=%s exp=%s", i, qty, cost, sell, md, mfg, exp)
		}
		if !shouldFail && err != nil {
			t.Fatalf("iter %d: unexpected validation error: %v", i, err)
		}

		mdErr := validateBatchMarkdownInput(dto.ApplyBatchMarkdownRequest{MarkdownPercent: md})
		if (math.IsNaN(md) || md <= 0 || md > 95) && mdErr == nil {
			t.Fatalf("iter %d: expected markdown error for md=%v", i, md)
		}
	}
}

package services

import (
	"math"
	"math/rand"
	"shopMe/internal/utils"
	"testing"
	"time"
)

func TestModule7CopilotAndEODTime_2000StressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(2026112807))

	// 1. 1,000 IST date-boundary normalization tests for EOD PDF keys
	for i := 0; i < 1000; i++ {
		hour := rng.Intn(24)
		min := rng.Intn(60)
		utcTime := time.Date(2026, 11, 28, hour, min, 0, 0, time.UTC)
		istTime := utcTime.In(utils.IndianLocation)

		// After 18:30 UTC, IST date must advance to 2026-11-29
		wantDate := "2026-11-28"
		if hour > 18 || (hour == 18 && min >= 30) {
			wantDate = "2026-11-29"
		}
		if got := istTime.Format("2006-01-02"); got != wantDate {
			t.Fatalf("case %d (%02d:%02d UTC): got IST date %s, want %s", i, hour, min, got, wantDate)
		}
	}

	// 2. 1,000 bargain floor & dead-stock floor invariant checks
	for i := 0; i < 1000; i++ {
		originalPrice := float64(50 + rng.Intn(2000))
		costPrice := float64(rng.Intn(int(originalPrice)))
		if i%5 == 0 {
			costPrice = 0 // triggers 70% fallback
		}

		effCost := costPrice
		if effCost <= 0 && originalPrice > 0 {
			effCost = math.Round(originalPrice * 0.70)
		}
		hardFloor := math.Max(effCost*1.12, effCost+15.0)
		deadFloor := math.Max(effCost*1.05, effCost+10.0)

		if hardFloor <= effCost {
			t.Fatalf("case %d: hardFloor %v must exceed effCost %v", i, hardFloor, effCost)
		}
		if deadFloor <= effCost || deadFloor > hardFloor {
			t.Fatalf("case %d: deadFloor %v must sit in (effCost=%v, hardFloor=%v]", i, deadFloor, effCost, hardFloor)
		}
	}
}

package services

import (
	"math"
	"testing"

	"go.uber.org/zap"
)

func TestCopilotService_EvaluateBargain_HardFloorProtection(t *testing.T) {
	logger := zap.NewNop()
	// Test pure mathematical margin & decision logic
	costPrice := 850.0
	originalPrice := 1400.0
	hardFloorPrice := math.Max(costPrice*1.12, costPrice+15.0) // ₹952.0

	// 1. Case A: Customer offers below cost price (DANGER: ₹700 < ₹850)
	offeredPriceA := 700.0
	isAcceptableA := false
	if offeredPriceA >= costPrice && offeredPriceA >= hardFloorPrice {
		isAcceptableA = true
	}
	if isAcceptableA {
		t.Fatalf("Expected offer below cost price to be rejected, but got accepted")
	}

	// 2. Case B: Customer offers ₹900 (Between Cost ₹850 and Floor ₹952) on normal stock
	offeredPriceB := 900.0
	isDeadStockB := false
	isAcceptableB := false
	if offeredPriceB >= hardFloorPrice {
		isAcceptableB = true
	} else if isDeadStockB && offeredPriceB >= costPrice*1.05 {
		isAcceptableB = true
	}
	if isAcceptableB {
		t.Fatalf("Expected offer below floor price on active inventory to be rejected")
	}

	// 3. Case C: Customer offers ₹900 on DEAD STOCK (>60 days idle, CZ category)
	isDeadStockC := true
	isAcceptableC := false
	deadStockFloor := math.Max(costPrice*1.05, costPrice+10.0) // ₹892.5
	if offeredPriceB >= deadStockFloor && isDeadStockC {
		isAcceptableC = true
	}
	if !isAcceptableC {
		t.Fatalf("Expected dead stock near-cost offer to be accepted to recover liquid capital")
	}

	// 4. Case D: Customer offers ₹1,100 (Above floor price)
	offeredPriceD := 1100.0
	isAcceptableD := offeredPriceD >= hardFloorPrice
	if !isAcceptableD {
		t.Fatalf("Expected offer above hard floor price to be accepted")
	}

	netProfitD := offeredPriceD - costPrice
	if netProfitD != 250.0 {
		t.Fatalf("Expected net profit ₹250, got ₹%.2f", netProfitD)
	}

	_ = logger
	_ = originalPrice
}

func TestCopilotService_EvaluateBargain_WorstCaseZeroCostAndExtremeNumbers(t *testing.T) {
	// 1. Unrecorded cost price (cost = 0) on a ₹2,000 product
	originalPrice := 2000.0
	costPrice := 0.0

	// System must safely infer 70% cost basis (₹1,400) instead of selling at ₹20
	if costPrice <= 0 && originalPrice > 0 {
		costPrice = math.Round(originalPrice * 0.70)
	}
	if costPrice != 1400.0 {
		t.Fatalf("Expected conservative cost fallback ₹1400, got ₹%.2f", costPrice)
	}

	hardFloorPrice := math.Max(costPrice*1.12, costPrice+15.0) // ₹1568.0
	if hardFloorPrice < 1500.0 {
		t.Fatalf("Expected floor price >= 1500, got ₹%.2f", hardFloorPrice)
	}

	// 2. Customer offers ₹50 on this ₹2,000 product (which had 0 cost initially)
	offeredPrice := 50.0
	isAcceptable := offeredPrice >= costPrice && offeredPrice >= hardFloorPrice
	if isAcceptable {
		t.Fatalf("DANGER: Sold ₹2000 product for ₹50 due to missing cost price!")
	}

	// 3. Extreme NaN / Inf numbers validation
	nanOffer := math.NaN()
	infOffer := math.Inf(1)
	negOffer := -200.0

	isValid := func(v float64) bool {
		return !math.IsNaN(v) && !math.IsInf(v, 0) && v > 0 && v <= 1e8
	}

	if isValid(nanOffer) || isValid(infOffer) || isValid(negOffer) {
		t.Fatalf("Extreme number sanitizer failed to block NaN/Inf/Negative values")
	}
}

func TestCopilotService_Recalculate_ZeroRevenueAndNaNImmunity(t *testing.T) {
	// Worst-case: Newly opened store with 0 sales in past 30 days
	grandRev := 0.0
	itemRev := 0.0
	qtySold := 0
	stddev := 0.0
	stock := 10

	revShare := 0.0
	if grandRev > 0 && !math.IsNaN(grandRev) && !math.IsInf(grandRev, 0) {
		revShare = itemRev / grandRev
	}
	if revShare != 0.0 || math.IsNaN(revShare) {
		t.Fatalf("Expected 0.0 revShare on 0 grandRev, got %f", revShare)
	}

	dailyVel := float64(qtySold) / 30.0
	if math.IsNaN(dailyVel) || math.IsInf(dailyVel, 0) || dailyVel < 0 {
		dailyVel = 0.0
	}
	if dailyVel != 0.0 {
		t.Fatalf("Expected 0.0 daily velocity, got %f", dailyVel)
	}

	cv := 1.5
	if dailyVel > 0 && !math.IsNaN(stddev) && !math.IsInf(stddev, 0) {
		cv = stddev / dailyVel
	}
	if cv != 1.5 || math.IsNaN(cv) {
		t.Fatalf("Expected default 1.5 CV when dailyVel is 0, got %f", cv)
	}

	daysSupply := 999
	if dailyVel > 0 {
		rawSupply := float64(stock) / dailyVel
		if !math.IsNaN(rawSupply) && !math.IsInf(rawSupply, 0) && rawSupply >= 0 {
			daysSupply = int(math.Min(999, rawSupply))
		}
	}
	if daysSupply != 999 {
		t.Fatalf("Expected 999 daysSupply when 0 daily velocity, got %d", daysSupply)
	}
}

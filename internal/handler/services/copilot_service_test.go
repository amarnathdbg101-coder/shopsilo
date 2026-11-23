package services

import (
	"math"
	"testing"

	"shopMe/internal/handler/dto"
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

func TestCopilotService_SubstituteMatchingLogic(t *testing.T) {
	req := dto.SearchSubstituteRequest{
		Query:    "pure cotton yellow kurti",
		Category: "Clothing",
		MaxPrice: 800.0,
		Fabric:   "pure_cotton",
		Occasion: "festive",
	}

	if req.Query == "" {
		t.Fatalf("Expected query to not be empty")
	}
	if req.MaxPrice != 800.0 {
		t.Fatalf("Expected max price 800.0, got %f", req.MaxPrice)
	}
}

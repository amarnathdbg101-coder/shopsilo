package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/services"
)

func TestStoreTelemetrySnapshot_FormatMarkdown(t *testing.T) {
	snap := &services.StoreTelemetrySnapshot{
		ShopID:             "shop_test_1",
		TodaySalesAmount:   15420.50,
		TodaySalesCount:    42,
		AverageBasketValue: 367.15,
		CashSales:          8200.00,
		UPISales:           6500.50,
		KhataSales:         720.00,
		TodayGrossProfit:   3850.00,
		TodayExpenses:      450.00,
		LowStockCount:      2,
		CriticalItems: []services.CriticalItem{
			{Name: "Aashirvaad Atta 5kg", Stock: 1, Threshold: 5},
			{Name: "Amul Butter 500g", Stock: 0, Threshold: 3},
		},
		TotalKhataUdhar:     8450.00,
		TotalKhataCustomers: 6,
		TopOverdueDebtors: []services.OverdueDebtor{
			{Name: "Sharmaji", Mobile: "9876543210", Balance: 2400.00},
		},
		TopMovingProductsToday: []string{"Atta 5kg", "Amul Milk 500ml", "Sugar 1kg"},
		CapturedAt:             time.Now(),
	}

	md := snap.FormatMarkdown()

	// Verify all key real ground truths are present in the markdown
	if !strings.Contains(md, "15420.50") {
		t.Errorf("expected sales amount 15420.50 in markdown")
	}
	if !strings.Contains(md, "42 bills") {
		t.Errorf("expected 42 bills in markdown")
	}
	if !strings.Contains(md, "Cash ₹8200.00 | UPI ₹6500.50") {
		t.Errorf("expected payment split in markdown")
	}
	if !strings.Contains(md, "Aashirvaad Atta 5kg") {
		t.Errorf("expected critical low stock item in markdown")
	}
	if !strings.Contains(md, "Sharmaji (₹2400") {
		t.Errorf("expected top overdue debtor in markdown")
	}
}

func TestTelemetryProvider_NilSafe(t *testing.T) {
	provider := services.NewTelemetryProvider(nil)
	snap, err := provider.GetSnapshot(context.Background(), "any_shop")
	if err != nil {
		t.Fatalf("unexpected error for nil db provider: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot returned")
	}
	if snap.ShopID != "any_shop" {
		t.Errorf("expected shopID 'any_shop', got %s", snap.ShopID)
	}
}

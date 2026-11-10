package services_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/migration"
	"shopMe/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func setupMarginTest(t *testing.T) (*pgxpool.Pool, *repository.ShopRepo, *repository.HotkeyRepo, *services.MarginService) {
	for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					_ = os.Setenv("DATABASE_URL", strings.TrimPrefix(line, "DATABASE_URL="))
				}
			}
			break
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	if testing.Short() {
		t.Skip("skipping remote db test in short mode")
	}
	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("cannot connect to test db (%v); skipping integration test", err)
		return nil, nil, nil, nil
	}

	logger, _ := zap.NewDevelopment()
	shopRepo := repository.NewShopRepo(pool, logger)
	hotkeyRepo := repository.NewHotkeyRepo(pool, logger)
	marginService := services.NewMarginService(shopRepo, hotkeyRepo, logger)

	return pool, shopRepo, hotkeyRepo, marginService
}

func TestMarginShieldService(t *testing.T) {
	pool, shopRepo, _, marginService := setupMarginTest(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	migration.EnsureSchemaColumns(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Fetch an existing shop
	var shopID, ownerUserID string
	err := pool.QueryRow(ctx, "SELECT id, user_id FROM shops WHERE is_active = true LIMIT 1;").Scan(&shopID, &ownerUserID)
	if err != nil {
		t.Skip("No active shop found for test")
	}

	// 2. Fetch or create a test product with explicit cost and floor price
	var prodID string
	err = pool.QueryRow(ctx, "SELECT id FROM products WHERE shop_id = $1 AND is_active = true LIMIT 1;", shopID).Scan(&prodID)
	if err != nil {
		t.Skip("No active product found for test")
	}

	// Set test product: Price = 1200, Cost = 800, Floor = 1000, MinMargin = 15%
	_, err = pool.Exec(ctx, `
		UPDATE products 
		SET price = 1200.00, cost_price = 800.00, floor_price = 1000.00, min_margin_percent = 15.00 
		WHERE id = $1;
	`, prodID)
	if err != nil {
		t.Fatalf("Failed to update test product prices: %v", err)
	}

	// Case A: Loss Alert (RED) - Proposed price below floor (e.g. ₹900)
	t.Run("Loss Alert Below Floor - RED", func(t *testing.T) {
		req := dto.MarginCheckRequest{
			ProductID:     prodID,
			ProposedPrice: 900.00,
			Quantity:      1,
		}
		res, err := marginService.CheckMargin(ctx, ownerUserID, req)
		if err != nil {
			t.Fatalf("CheckMargin failed: %v", err)
		}
		if res.IsAllowed {
			t.Errorf("Expected IsAllowed=false for price 900 below floor 1000")
		}
		if res.IndicatorColor != "RED" {
			t.Errorf("Expected IndicatorColor=RED, got %s", res.IndicatorColor)
		}
		if res.StealthBadge != "🔴 LOSS ALERT" {
			t.Errorf("Expected StealthBadge=🔴 LOSS ALERT, got %s", res.StealthBadge)
		}
		t.Logf("Red Check Passed: %s", res.WarningMessage)
	})

	// Case B: Tight Margin (ORANGE) - Proposed price is at floor 1020 (Profit = 220, Margin = 21.5% -> Wait, let's test e.g. 11% margin)
	t.Run("Tight Margin - ORANGE", func(t *testing.T) {
		// Update floor to 880, cost 800 -> at 890, profit is 90 (10.1% margin)
		_, _ = pool.Exec(ctx, "UPDATE products SET floor_price = 850.00, cost_price = 800.00, min_margin_percent = 5.00 WHERE id = $1;", prodID)
		req := dto.MarginCheckRequest{
			ProductID:     prodID,
			ProposedPrice: 890.00,
			Quantity:      2,
		}
		res, err := marginService.CheckMargin(ctx, ownerUserID, req)
		if err != nil {
			t.Fatalf("CheckMargin failed: %v", err)
		}
		if !res.IsAllowed {
			t.Errorf("Expected IsAllowed=true for price 890 above floor 850")
		}
		if res.IndicatorColor != "ORANGE" {
			t.Errorf("Expected IndicatorColor=ORANGE, got %s (margin: %.2f%%)", res.IndicatorColor, res.ProfitMarginPct)
		}
		t.Logf("Orange Check Passed: %s (Total Profit: ₹%.2f)", res.WarningMessage, res.TotalProfit)
	})

	// Case C: Safe Healthy Margin (GREEN) - Proposed price 1150 (Cost 800 -> Profit 350, 30.4% margin)
	t.Run("Safe Margin - GREEN", func(t *testing.T) {
		req := dto.MarginCheckRequest{
			ProductID:     prodID,
			ProposedPrice: 1150.00,
			Quantity:      3,
		}
		res, err := marginService.CheckMargin(ctx, ownerUserID, req)
		if err != nil {
			t.Fatalf("CheckMargin failed: %v", err)
		}
		if !res.IsAllowed {
			t.Errorf("Expected IsAllowed=true for price 1150")
		}
		if res.IndicatorColor != "GREEN" {
			t.Errorf("Expected IndicatorColor=GREEN, got %s", res.IndicatorColor)
		}
		if res.TotalProfit != 350.00*3 {
			t.Errorf("Expected TotalProfit=1050, got %.2f", res.TotalProfit)
		}
		t.Logf("Green Check Passed: %s", res.WarningMessage)
	})

	// 3. Test Hotkey Fast-Grid APIs
	t.Run("Fast-Grid Hotkeys CRUD", func(t *testing.T) {
		createReq := dto.CreateShopHotkeyRequest{
			ProductID:   prodID,
			DisplayName: "Special Kurti",
			ColorHex:    "#2563EB",
			SortOrder:   1,
			TimeSlot:    "ALL_DAY",
		}
		created, err := marginService.CreateHotkey(ctx, ownerUserID, createReq)
		if err != nil {
			t.Fatalf("CreateHotkey failed: %v", err)
		}
		t.Logf("Created Hotkey: ID=%s, Name=%s", created.ID, created.DisplayName)

		listRes, err := marginService.ListHotkeys(ctx, ownerUserID, "ALL_DAY")
		if err != nil {
			t.Fatalf("ListHotkeys failed: %v", err)
		}
		if listRes.Count == 0 {
			t.Errorf("Expected at least 1 hotkey in list")
		}
		t.Logf("ListHotkeys returned %d hotkeys", listRes.Count)

		// Delete hotkey
		err = marginService.DeleteHotkey(ctx, ownerUserID, created.ID)
		if err != nil {
			t.Fatalf("DeleteHotkey failed: %v", err)
		}
		t.Logf("DeleteHotkey succeeded for ID: %s", created.ID)
	})
	_ = shopRepo
}


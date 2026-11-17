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


package services_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/migration"
	"shopMe/internal/utils"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *repository.LocationRepo, *repository.ShopRepo, *repository.ProductRepo, *services.LocationService) {
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
		t.Skip("DATABASE_URL not set; skipping database integration test")
	}

	if testing.Short() {
		t.Skip("skipping remote db test in short mode")
	}
	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("cannot connect to test db (%v); skipping integration test", err)
	}

	if err := migration.RunAutoMigrations(pool); err != nil {
		t.Fatalf("RunAutoMigrations failed: %v", err)
	}
	migration.EnsureSchemaColumns(pool)

	logger := zap.NewNop()
	locRepo := repository.NewLocationRepo(pool, logger)
	shopRepo := repository.NewShopRepo(pool, logger)
	prodRepo := repository.NewProductRepo(pool, logger)
	locService := services.NewLocationService(locRepo, shopRepo, prodRepo)

	return pool, locRepo, shopRepo, prodRepo, locService
}

func TestGenerateVariantMatrix_Calculations(t *testing.T) {
	pool, _, _, prodRepo, locService := setupTestDB(t)

	ctx := context.Background()
	// Find any active shop owner user
	var userID, shopID string
	err := pool.QueryRow(ctx, "SELECT user_id, id FROM shops WHERE is_active = true LIMIT 1;").Scan(&userID, &shopID)
	if err != nil {
		t.Skip("no active shop found for integration test")
	}

	// 1. Create a parent apparel product
	prod, err := prodRepo.Create(ctx, &model.Product{
		ShopID:       shopID,
		Name:         "Festive Rayon Anarkali Kurti",
		Slug:         "festive-rayon-anarkali-kurti-test",
		SKU:          "KURTI-FESTIVE-01",
		Price:        899.0,
		CostPrice:    450.0,
		ComparePrice: 1299.0,
		FloorPrice:   750.0,
		IsActive:     true,
	}, 0)
	if err != nil {
		t.Fatalf("failed to create parent product: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1;", prod.ID)
	}()

	// 2. Generate Size x Color Matrix (4 sizes x 2 colors = 8 variants)
	matrixReq := dto.GenerateVariantMatrixRequest{
		Sizes: []string{"S", "M", "L", "XL"},
		Colors: []dto.ColorOption{
			{Name: "Navy Blue", Hex: "#000080"},
			{Name: "Maroon", Hex: "#800000"},
		},
		Price:            899.0,
		CostPrice:        450.0,
		ComparePrice:     1299.0,
		FloorPrice:       750.0,
		InitialStockEach: 5,
	}

	variants, err := locService.GenerateVariantMatrix(ctx, userID, prod.ID, matrixReq)
	if err != nil {
		t.Fatalf("GenerateVariantMatrix failed: %v", err)
	}

	if len(variants) != 8 {
		t.Errorf("expected 8 variants, got %d", len(variants))
	}

	// Verify generated SKU pattern
	expectedSKU := "KURTI-FESTIVE-01-NAVYBLUE-S"
	var found bool
	for _, v := range variants {
		if v.SKU == expectedSKU {
			found = true
			if v.Price != 899.0 || v.FloorPrice != 750.0 {
				t.Errorf("variant pricing mismatch: %+v", v)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected variant SKU %s not found in generated results", expectedSKU)
	}
}

func TestStockTransfer_Workflow(t *testing.T) {
	pool, locRepo, _, _, locService := setupTestDB(t)

	ctx := context.Background()
	var userID, shopID string
	err := pool.QueryRow(ctx, "SELECT user_id, id FROM shops WHERE is_active = true LIMIT 1;").Scan(&userID, &shopID)
	if err != nil {
		t.Skip("no active shop found")
	}

	// 1. Ensure at least 2 locations exist (Showroom & Secondary Godown)
	showroom, err := locRepo.GetDefaultLocation(ctx, shopID)
	if err != nil {
		t.Fatalf("failed to get default showroom: %v", err)
	}

	godown, err := locRepo.CreateLocation(ctx, shopID, dto.CreateLocationRequest{
		Name: "Basement Godown B-2",
		Code: "GDN-B2",
		Type: model.LocationTypeGodown,
	})
	if err != nil {
		// Code might exist in re-runs, fetch it
		locs, _ := locRepo.ListLocationsByShop(ctx, shopID)
		for _, l := range locs {
			if l.Code == "GDN-B2" {
				godown = l
				break
			}
		}
	}

	if godown == nil {
		t.Fatal("failed to resolve secondary godown location")
	}

	// 2. Create a test product with 20 items in Godown
	var prodID string
	err = pool.QueryRow(ctx, `
		INSERT INTO products (shop_id, name, slug, sku, price, cost_price, is_active)
		VALUES ($1, 'Silk Brocade Dupatta', 'silk-brocade-dupatta-test', 'DUP-SILK-TEST', 499, 200, true)
		RETURNING id;
	`, shopID).Scan(&prodID)
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM products WHERE id = $1;", prodID)
	}()

	// Deposit 20 units in Godown
	err = locRepo.AdjustLocationStock(ctx, shopID, godown.ID, prodID, nil, 20)
	if err != nil {
		t.Fatalf("failed to deposit stock in godown: %v", err)
	}

	// 3. Create Transfer Requisition (Move 6 units from Godown to Showroom)
	transfer, err := locService.CreateStockTransfer(ctx, userID, dto.CreateStockTransferRequest{
		FromLocationID: godown.ID,
		ToLocationID:   showroom.ID,
		Notes:          "Weekend evening rush stock replenishment",
		Items: []dto.StockTransferItemRequest{
			{
				ProductID:    prodID,
				RequestedQty: 6,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateStockTransfer failed: %v", err)
	}

	if transfer.Status != model.TransferStatusPending {
		t.Errorf("expected initial status PENDING, got %s", transfer.Status)
	}

	// 4. Dispatch Transfer (Godown stock should decrease from 20 -> 14)
	transfer, err = locService.UpdateTransferStatus(ctx, userID, transfer.ID, dto.UpdateTransferStatusRequest{
		Status: model.TransferStatusDispatched,
	})
	if err != nil {
		t.Fatalf("UpdateTransferStatus(DISPATCHED) failed: %v", err)
	}

	// Verify godown stock decreased
	invGodown, err := locRepo.GetLocationInventory(ctx, shopID, godown.ID)
	if err != nil {
		t.Fatalf("GetLocationInventory godown failed: %v", err)
	}
	for _, it := range invGodown {
		if it.ProductID == prodID {
			if it.Quantity != 14 {
				t.Errorf("expected 14 units in godown after dispatch, got %d", it.Quantity)
			}
		}
	}

	// 5. Progress to RECEIVED then RECONCILED (Showroom stock should increase by 6)
	transfer, err = locService.UpdateTransferStatus(ctx, userID, transfer.ID, dto.UpdateTransferStatusRequest{
		Status: model.TransferStatusReceived,
	})
	if err != nil {
		t.Fatalf("UpdateTransferStatus(RECEIVED) failed: %v", err)
	}

	transfer, err = locService.UpdateTransferStatus(ctx, userID, transfer.ID, dto.UpdateTransferStatusRequest{
		Status: model.TransferStatusReconciled,
	})
	if err != nil {
		t.Fatalf("UpdateTransferStatus(RECONCILED) failed: %v", err)
	}

	if transfer.Status != model.TransferStatusReconciled {
		t.Errorf("expected RECONCILED status, got %s", transfer.Status)
	}

	// Verify showroom stock increased by 6
	invShowroom, err := locRepo.GetLocationInventory(ctx, shopID, showroom.ID)
	if err != nil {
		t.Fatalf("GetLocationInventory showroom failed: %v", err)
	}
	var showroomFound bool
	for _, it := range invShowroom {
		if it.ProductID == prodID {
			showroomFound = true
			if it.Quantity < 6 {
				t.Errorf("expected at least 6 units in showroom after reconciliation, got %d", it.Quantity)
			}
		}
	}
	if !showroomFound {
		t.Error("reconciled stock item not found in showroom location inventory")
	}
}


package services_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestProductServiceFindNearbyProducts(t *testing.T) {
	for _, envPath := range []string{"../../.env", "../../../.env", "../.env", ".env"} {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					realURL := strings.TrimPrefix(line, "DATABASE_URL=")
					_ = os.Setenv("DATABASE_URL", realURL)
					break
				}
			}
			break
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || strings.Contains(dbURL, "localhost:5432") {
		t.Skip("Skipping live database test: valid remote DATABASE_URL not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if testing.Short() {
		t.Skip("skipping remote db test in short mode")
	}
	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	productRepo := repository.NewProductRepo(pool, logger)
	shopRepo := repository.NewShopRepo(pool, logger)
	categoryRepo := repository.NewCategoryRepo(pool, logger)
	productService := services.NewProductService(productRepo, shopRepo, categoryRepo)

	// Coordinates of Darbhanga (around active registered shops)
	lat := 26.1512
	lng := 85.9664
	radiusKm := 15.0

	res, err := productService.FindNearbyProducts(ctx, lat, lng, radiusKm, "", "", false, 1, 10)
	if err != nil {
		t.Fatalf("FindNearbyProducts service call failed: %v", err)
	}

	t.Logf("FindNearbyProducts returned %d in-stock items (Total: %d, TotalPages: %d)", len(res.Products), res.TotalCount, res.TotalPages)
	if res.TotalCount == 0 {
		t.Logf("Warning: 0 nearby items found in test DB; skipping individual product checks")
		return
	}

	if len(res.Products) == 0 {
		t.Fatalf("Expected at least 1 product item in first page, got 0")
	}

	// Verify all returned products satisfy the Decision A enterprise guarantees
	for i, item := range res.Products {
		if item.ProductID == "" {
			t.Errorf("[%d] item ProductID is empty", i)
		}
		if item.ID != item.ProductID {
			t.Errorf("[%d] item.ID (%s) != item.ProductID (%s)", i, item.ID, item.ProductID)
		}
		if item.ShopID == "" {
			t.Errorf("[%d] item ShopID is empty", i)
		}
		// Strict in-stock guarantee: (quantity - reserved_quantity) must be > 0
		if !item.IsInStock {
			t.Errorf("[%d] product %s has IsInStock = false, but was returned in nearby search", i, item.Name)
		}
		if item.AvailableQuantity <= 0 {
			t.Errorf("[%d] product %s has AvailableQuantity = %d <= 0, violating in-stock filter", i, item.Name, item.AvailableQuantity)
		}
		// Distance must be non-negative and within the requested radius
		if item.DistanceKm < 0 || item.DistanceKm > radiusKm+0.1 {
			t.Errorf("[%d] product %s distance %.2f km is outside radius %.1f km", i, item.Name, item.DistanceKm, radiusKm)
		}

		if i < 3 {
			t.Logf("Verified Item [%d]: %s | Shop: %s (%s) | Dist: %.2f km | Stock: %d | Open: %v",
				i+1, item.Name, item.ShopName, item.ShopCity, item.DistanceKm, item.AvailableQuantity, item.IsCurrentlyOpen)
		}
	}
}


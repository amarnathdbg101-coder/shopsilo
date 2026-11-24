package repository_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestBulkImportProducts_HighSpeedBatch(t *testing.T) {
	// Try loading .env from parent directories
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
		t.Skip("Skipping live database test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	repo := repository.NewProductRepo(pool, logger)

	// Fetch an active shop from the database
	var shopID string
	err = pool.QueryRow(ctx, "SELECT id FROM shops LIMIT 1").Scan(&shopID)
	if err != nil || shopID == "" {
		t.Skip("No shop found in database to run bulk import test")
	}

	testBatchID := fmt.Sprintf("BATCH%d", time.Now().UnixNano()%100000)

	// Generate 120 test products to test multi-chunk batching (chunk size = 50)
	totalTestItems := 120
	var importItems []dto.BulkImportProductItem
	for i := 1; i <= totalTestItems; i++ {
		sku := fmt.Sprintf("%s-SKU-%03d", testBatchID, i)
		importItems = append(importItems, dto.BulkImportProductItem{
			Name:          fmt.Sprintf("Bulk Test Product %s %d", testBatchID, i),
			SKU:           sku,
			Price:         100.0 + float64(i),
			CostPrice:     80.0 + float64(i),
			ComparePrice:  120.0 + float64(i),
			StockQuantity: 50 + i,
			MinStock:      5,
			CategoryName:  "General Store",
			Description:   fmt.Sprintf("High speed test item %d", i),
		})
	}

	// Clean up after test
	defer func() {
		cleanupCtx, cCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cCancel()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM products WHERE shop_id = $1 AND sku LIKE $2", shopID, testBatchID+"%")
	}()

	startTime := time.Now()
	res, err := repo.BulkImportProducts(ctx, shopID, importItems, false)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("BulkImportProducts failed: %v", err)
	}

	if res.ImportedCount != totalTestItems {
		t.Fatalf("Expected %d imported products, got %d (Errors: %v)", totalTestItems, res.ImportedCount, res.Errors)
	}

	t.Logf("SUCCESS: Bulk imported %d items in %v (Average: %v per item)", res.ImportedCount, duration, duration/time.Duration(totalTestItems))

	// Verify inventory records were also created for these products
	var inventoryCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM inventory inv 
		JOIN products p ON inv.product_id = p.id 
		WHERE p.shop_id = $1 AND p.sku LIKE $2
	`, shopID, testBatchID+"%").Scan(&inventoryCount)

	if err != nil {
		t.Fatalf("Failed to verify inventory count: %v", err)
	}

	if inventoryCount != totalTestItems {
		t.Errorf("Expected %d inventory rows, found %d", totalTestItems, inventoryCount)
	}

	// Test Update of Existing Products
	updateItems := make([]dto.BulkImportProductItem, len(importItems))
	copy(updateItems, importItems)
	for i := range updateItems {
		updateItems[i].Price = 999.0 // Updated price
		updateItems[i].StockQuantity = 100
	}

	updateStart := time.Now()
	updateRes, err := repo.BulkImportProducts(ctx, shopID, updateItems, true)
	updateDuration := time.Since(updateStart)

	if err != nil {
		t.Fatalf("Bulk update failed: %v", err)
	}

	if updateRes.UpdatedCount != totalTestItems {
		t.Errorf("Expected %d updated products, got %d (Errors: %v)", totalTestItems, updateRes.UpdatedCount, updateRes.Errors)
	}

	t.Logf("SUCCESS: Bulk updated %d items in %v", updateRes.UpdatedCount, updateDuration)
}

func TestBulkImportProducts_MultiChunkScale(t *testing.T) {
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
		t.Skip("Skipping live database test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	repo := repository.NewProductRepo(pool, logger)

	var shopID string
	err = pool.QueryRow(ctx, "SELECT id FROM shops LIMIT 1").Scan(&shopID)
	if err != nil || shopID == "" {
		t.Skip("No shop found in database to run scale test")
	}

	testBatchID := fmt.Sprintf("SCALE%d", time.Now().UnixNano()%100000)

	// 2,500 items to cross multiple 1,000-item micro-batches (1,000 + 1,000 + 500)
	totalScaleItems := 2500
	var scaleItems []dto.BulkImportProductItem
	for i := 1; i <= totalScaleItems; i++ {
		sku := fmt.Sprintf("%s-SKU-%04d", testBatchID, i)
		scaleItems = append(scaleItems, dto.BulkImportProductItem{
			Name:          fmt.Sprintf("Scale Test Product %s %d", testBatchID, i),
			SKU:           sku,
			Price:         50.0 + float64(i%500),
			CostPrice:     40.0 + float64(i%500),
			ComparePrice:  60.0 + float64(i%500),
			StockQuantity: 100,
			MinStock:      10,
			CategoryName:  "General Store",
			Description:   fmt.Sprintf("Multi-chunk stress item %d", i),
		})
	}

	defer func() {
		cleanupCtx, cCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cCancel()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM products WHERE shop_id = $1 AND sku LIKE $2", shopID, testBatchID+"%")
	}()

	start := time.Now()
	res, err := repo.BulkImportProducts(ctx, shopID, scaleItems, false)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Multi-chunk bulk import failed: %v", err)
	}

	if res.ImportedCount != totalScaleItems {
		t.Fatalf("Expected %d imported products, got %d (Errors: %v)", totalScaleItems, res.ImportedCount, res.Errors)
	}

	throughput := float64(res.ImportedCount) / elapsed.Seconds()
	t.Logf("SCALE PASS: Bulk imported %d items in %v (Throughput: %.1f products/sec)", res.ImportedCount, elapsed, throughput)
}

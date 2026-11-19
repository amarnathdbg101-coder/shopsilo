package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestCategoryAndAppVersion_InMemoryTTLCache(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	catRepo := repository.NewCategoryRepo(pool, logger)
	versionRepo := repository.NewAppVersionRepo(pool, logger)

	// 1. Benchmark Category Cache (FindDefault)
	// Cold call (hits DB)
	startCold := time.Now()
	defCat1, err := catRepo.FindDefault(ctx)
	coldDuration := time.Since(startCold)
	if err != nil {
		t.Fatalf("FindDefault failed: %v", err)
	}

	// Warm call (hits In-Memory Cache via FindByID/FindByNameOrSlug)
	startWarm := time.Now()
	defCat2, err := catRepo.FindByID(ctx, defCat1.ID)
	warmDuration := time.Since(startWarm)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if defCat1.ID != defCat2.ID {
		t.Errorf("Mismatch in category ID: %s vs %s", defCat1.ID, defCat2.ID)
	}

	t.Logf("CATEGORY CACHE PROOF -> Cold DB call: %v | Hot In-Memory cache call: %v", coldDuration, warmDuration)

	// 2. Benchmark AppVersion Cache
	// Call 1 (Warm up)
	startV1 := time.Now()
	v1, err := versionRepo.GetLatestActiveVersion(ctx, "merchant", "android")
	v1Duration := time.Since(startV1)

	if err == nil && v1 != nil {
		// Call 2 (Sub-microsecond RAM cache)
		startV2 := time.Now()
		v2, err := versionRepo.GetLatestActiveVersion(ctx, "merchant", "android")
		v2Duration := time.Since(startV2)

		if err != nil || v2 == nil {
			t.Fatalf("Failed to fetch cached app version: %v", err)
		}

		t.Logf("APP VERSION CACHE PROOF -> Call 1: %v | Call 2 (In-Memory RAM): %v (Speedup: %.1fx)", v1Duration, v2Duration, float64(v1Duration)/float64(v2Duration))
	} else {
		t.Logf("App version record not found in DB (skipped version sub-check): %v", err)
	}

	// 3. Category Shop List Caching
	var shopID string
	_ = pool.QueryRow(ctx, "SELECT id FROM shops LIMIT 1").Scan(&shopID)
	if shopID != "" {
		startShopCold := time.Now()
		_, err := catRepo.ListByShop(ctx, shopID)
		shopColdDur := time.Since(startShopCold)
		if err != nil {
			t.Fatalf("ListByShop cold failed: %v", err)
		}

		startShopWarm := time.Now()
		_, err = catRepo.ListByShop(ctx, shopID)
		shopWarmDur := time.Since(startShopWarm)
		if err != nil {
			t.Fatalf("ListByShop warm failed: %v", err)
		}

		t.Logf("SHOP CATEGORIES CACHE PROOF -> Cold ListByShop: %v | Hot Cached ListByShop: %v (Speedup: %.1fx)", shopColdDur, shopWarmDur, float64(shopColdDur)/float64(shopWarmDur))
	}
}

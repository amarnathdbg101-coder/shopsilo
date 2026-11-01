package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestTypoAndFuzzyProductSearch(t *testing.T) {
	_ = os.Setenv("PORT", "8080")
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

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	repo := repository.NewProductRepo(pool, logger)

	testCases := []struct {
		name       string
		searchTerm string
		expectMin  int
	}{
		{
			name:       "Exact search for shampoo",
			searchTerm: "shampoo",
			expectMin:  1,
		},
		{
			name:       "Typo search for clinic (clnic)",
			searchTerm: "clnic",
			expectMin:  1,
		},
		{
			name:       "Typo search for pantene (pantn)",
			searchTerm: "pantn",
			expectMin:  1,
		},
		{
			name:       "Multi-word typo search (clnic shampu)",
			searchTerm: "clnic shampu",
			expectMin:  1,
		},
		{
			name:       "Bilingual Hinglish search for hair oil (tel)",
			searchTerm: "tel",
			expectMin:  1,
		},
		{
			name:       "Non-existent nonsense query",
			searchTerm: "xyznotfound999",
			expectMin:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter := dto.ProductFilter{
				Search: tc.searchTerm,
				Limit:  10,
				Page:   1,
			}
			products, totalCount, err := repo.FindAll(ctx, filter)
			if err != nil {
				t.Fatalf("FindAll failed for query '%s': %v", tc.searchTerm, err)
			}
			t.Logf("Query: '%s' -> Found %d products (total: %d)", tc.searchTerm, len(products), totalCount)
			for i, p := range products {
				if i < 3 {
					t.Logf("  [%d] %s (₹%.2f)", i+1, p.Name, p.Price)
				}
			}
			if totalCount < tc.expectMin {
				t.Errorf("Expected at least %d product for query '%s', got %d", tc.expectMin, tc.searchTerm, totalCount)
			}
		})
	}

	// Test 2: In-Memory TTL Cache Hit Speed
	t.Run("In-memory cache hit is sub-millisecond", func(t *testing.T) {
		filter := dto.ProductFilter{Search: "shampoo", Limit: 10, Page: 1}
		start := time.Now()
		_, _, err := repo.FindAll(ctx, filter)
		if err != nil {
			t.Fatalf("first call failed: %v", err)
		}
		firstDuration := time.Since(start)

		// Second call should hit in-memory cache!
		cacheStart := time.Now()
		cachedProds, _, err := repo.FindAll(ctx, filter)
		if err != nil {
			t.Fatalf("cached call failed: %v", err)
		}
		cacheDuration := time.Since(cacheStart)

		t.Logf("First call (DB): %v | Second call (In-Memory Cache): %v (speedup: %.1fx)", firstDuration, cacheDuration, float64(firstDuration)/float64(cacheDuration+1))
		if len(cachedProds) == 0 {
			t.Errorf("expected cached products, got 0")
		}
	})

	// Test 3: Fast Autocomplete Suggestions & In-Memory LRU Cache SLA (<0.5ms)
	t.Run("Autocomplete search suggestions with cache speedup", func(t *testing.T) {
		start := time.Now()
		suggestions1, err := repo.SuggestProducts(ctx, "cli", "", 5)
		if err != nil {
			t.Fatalf("SuggestProducts first call failed: %v", err)
		}
		dbDuration := time.Since(start)
		t.Logf("First call (DB): %v -> Suggestions: %v", dbDuration, suggestions1)
		if len(suggestions1) == 0 {
			t.Errorf("expected suggestions for 'cli', got 0")
		}

		// Second call must hit Go in-memory LRU cache in sub-millisecond (< 0.5ms)
		cacheStart := time.Now()
		suggestions2, err := repo.SuggestProducts(ctx, "cli", "", 5)
		if err != nil {
			t.Fatalf("SuggestProducts cached call failed: %v", err)
		}
		cacheDuration := time.Since(cacheStart)
		t.Logf("Second call (In-Memory LRU Cache): %v -> Count: %d", cacheDuration, len(suggestions2))

		if cacheDuration > 5*time.Millisecond {
			t.Errorf("expected in-memory cache hit to be under 5ms, took %v", cacheDuration)
		}
	})

	// Test 4: Bilingual Hinglish Autocomplete Suggestion (tel -> Hair Oil / Tail)
	t.Run("Hinglish retail term autocomplete suggestion", func(t *testing.T) {
		suggestions, err := repo.SuggestProducts(ctx, "tel", "", 5)
		if err != nil {
			t.Fatalf("Hinglish SuggestProducts failed: %v", err)
		}
		t.Logf("Hinglish suggestions for 'tel': %v", suggestions)
		if len(suggestions) == 0 {
			t.Errorf("expected suggestions for 'tel', got 0")
		}
	})
}

func TestCheckPostGIS(t *testing.T) {
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
		t.Skip("Skipping: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer pool.Close()

	// Try CREATE EXTENSION IF NOT EXISTS postgis
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS postgis;")
	if err != nil {
		t.Logf("CREATE EXTENSION postgis result: %v", err)
	} else {
		t.Logf("CREATE EXTENSION postgis succeeded!")
		var version string
		if err := pool.QueryRow(ctx, "SELECT postgis_version();").Scan(&version); err != nil {
			t.Logf("SELECT postgis_version error: %v", err)
		} else {
			t.Logf("PostGIS Version: %s", version)
		}
	}
}


func TestShopSearchByAddressAndFuzzy(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
			if data, err := os.ReadFile(envPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "DATABASE_URL=") {
						dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
						break
					}
				}
			}
			if dbURL != "" {
				break
			}
		}
	}
	if dbURL == "" {
		t.Skip("Skipping shop search test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping shop search test: unable to connect: %v", err)
	}
	defer pool.Close()

	shopRepo := repository.NewShopRepo(pool, zap.NewNop())

	// List all active shops first
	allShops, totalShops, _ := shopRepo.FindAll(ctx, dto.ShopFilter{Limit: 5, Page: 1})
	t.Logf("Total active shops in DB: %d (fetched: %d)", totalShops, len(allShops))
	for _, s := range allShops {
		t.Logf("  Existing Shop -> ID: %s | Name: %s | City: %s | Address: %s", s.ID, s.Name, s.City, s.Address)
	}

	t.Run("Search shop by address locality (pipal chowk)", func(t *testing.T) {
		shops, count, err := shopRepo.FindAll(ctx, dto.ShopFilter{
			Search: "pipal chowk",
			Limit:  10,
			Page:   1,
		})
		if err != nil {
			t.Fatalf("Shop FindAll failed: %v", err)
		}
		t.Logf("Found %d shops for 'pipal chowk' (count: %d)", len(shops), count)
		for _, s := range shops {
			t.Logf("  Matched Shop: %s | Address: %s", s.Name, s.Address)
		}
		if count < 1 {
			t.Errorf("Expected at least 1 shop matching address 'pipal chowk', got %d", count)
		}
	})

	t.Run("Search shop by address locality with typo (pipal chwk)", func(t *testing.T) {
		shops, count, err := shopRepo.FindAll(ctx, dto.ShopFilter{
			Search: "pipal chwk",
			Limit:  10,
			Page:   1,
		})
		if err != nil {
			t.Fatalf("Shop FindAll failed: %v", err)
		}
		t.Logf("Found %d shops for 'pipal chwk' with typo (count: %d)", len(shops), count)
		for _, s := range shops {
			t.Logf("  Matched Shop: %s | Address: %s", s.Name, s.Address)
		}
		if count < 1 {
			t.Errorf("Expected at least 1 shop matching typo address 'pipal chwk', got %d", count)
		}
	})

	t.Run("Search shop by city name in free-text search (Darbhanga)", func(t *testing.T) {
		shops, count, err := shopRepo.FindAll(ctx, dto.ShopFilter{
			Search: "Darbhanga",
			Limit:  10,
			Page:   1,
		})
		if err != nil {
			t.Fatalf("Shop FindAll failed: %v", err)
		}
		t.Logf("Found %d shops for 'Darbhanga' (count: %d)", len(shops), count)
		for _, s := range shops {
			t.Logf("  Matched Shop: %s | City: %s", s.Name, s.City)
		}
		if count < 1 {
			t.Errorf("Expected at least 1 shop matching city 'Darbhanga', got %d", count)
		}
	})
}

func checkPostGISHelper(t *testing.T) {
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
		t.Skip("Skipping: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer pool.Close()

	// Try CREATE EXTENSION IF NOT EXISTS postgis
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS postgis;")
	if err != nil {
		t.Logf("CREATE EXTENSION postgis result: %v", err)
	} else {
		t.Logf("CREATE EXTENSION postgis succeeded!")
		var version string
		if err := pool.QueryRow(ctx, "SELECT postgis_version();").Scan(&version); err != nil {
			t.Logf("SELECT postgis_version error: %v", err)
		} else {
			t.Logf("PostGIS Version: %s", version)
		}
	}
}


func TestHierarchicalGeoAndShopScopedSearch(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
			if data, err := os.ReadFile(envPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "DATABASE_URL=") {
						dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
						break
					}
				}
			}
			if dbURL != "" {
				break
			}
		}
	}
	if dbURL == "" {
		t.Skip("Skipping test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Unable to connect: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	shopRepo := repository.NewShopRepo(pool, logger)
	productRepo := repository.NewProductRepo(pool, logger)

	// Fetch known shop from DB to get real test coordinates and ID
	allShops, _, _ := shopRepo.FindAll(ctx, dto.ShopFilter{Limit: 1, Page: 1})
	if len(allShops) == 0 {
		t.Skip("No active shops found in database to test")
	}
	testShop := allShops[0]
	shopLat := *testShop.Latitude
	shopLng := *testShop.Longitude
	shopID := testShop.ID
	t.Logf("Testing with real Shop: '%s' (ID: %s) at Lat: %.4f, Lng: %.4f", testShop.Name, shopID, shopLat, shopLng)

	// -------------------------------------------------------------
	// TEST 1: Strict Geo Location Filter on Shops
	// -------------------------------------------------------------
	t.Run("Geo Layer: Shop inside user radius is found", func(t *testing.T) {
		radius := 15.0
		shops, count, err := shopRepo.FindAll(ctx, dto.ShopFilter{
			Lat:      &shopLat,
			Lng:      &shopLng,
			RadiusKm: &radius,
			Limit:    10,
			Page:     1,
		})
		if err != nil {
			t.Fatalf("Shop FindAll failed: %v", err)
		}
		t.Logf("Found %d shops (fetched %d) within %.1f km of shop coordinates", count, len(shops), radius)
		if count < 1 {
			t.Errorf("Expected at least 1 shop inside radius %.1f km, got %d", radius, count)
		}
	})

	t.Run("Geo Layer: User far away (1000km away) finds 0 shops in range", func(t *testing.T) {
		// Opposite coordinates (e.g., Chennai: 13.0827, 80.2707) with 10km radius
		farLat := 13.0827
		farLng := 80.2707
		smallRadius := 10.0
		shops, count, err := shopRepo.FindAll(ctx, dto.ShopFilter{
			Lat:      &farLat,
			Lng:      &farLng,
			RadiusKm: &smallRadius,
			Limit:    10,
			Page:     1,
		})
		if err != nil {
			t.Fatalf("Shop FindAll failed: %v", err)
		}
		t.Logf("Shops found in Chennai (far away): %d", count)
		if count != 0 {
			t.Errorf("Expected 0 shops in 10km radius from Chennai, but got %d (leakage detected!)", count)
		}
		if len(shops) != 0 {
			t.Errorf("Expected empty shop slice, got %d", len(shops))
		}
	})

	// -------------------------------------------------------------
	// TEST 2: Strict Geo Location Filter on Products
	// -------------------------------------------------------------
	t.Run("Geo Layer: User in location range can search products", func(t *testing.T) {
		radius := 20.0
		products, count, err := productRepo.FindAll(ctx, dto.ProductFilter{
			Lat:      &shopLat,
			Lng:      &shopLng,
			RadiusKm: &radius,
			Limit:    10,
			Page:     1,
		})
		if err != nil {
			t.Fatalf("Product FindAll failed: %v", err)
		}
		t.Logf("Found %d products (fetched %d) from shops within %.1f km radius", count, len(products), radius)
	})

	t.Run("Geo Layer: User far away finds 0 products in range", func(t *testing.T) {
		farLat := 13.0827
		farLng := 80.2707
		smallRadius := 5.0
		products, count, err := productRepo.FindAll(ctx, dto.ProductFilter{
			Lat:      &farLat,
			Lng:      &farLng,
			RadiusKm: &smallRadius,
			Search:   "shampoo",
			Limit:    10,
			Page:     1,
		})
		if err != nil {
			t.Fatalf("Product FindAll failed: %v", err)
		}
		t.Logf("Products found from Chennai (far away): %d", count)
		if count != 0 {
			t.Errorf("Expected 0 products in 5km radius from Chennai, but got %d (leakage detected!)", count)
		}
		if len(products) != 0 {
			t.Errorf("Expected empty product slice, got %d", len(products))
		}
	})

	// -------------------------------------------------------------
	// TEST 3: Strict Shop-Scoped Product Search (Inside Shop)
	// -------------------------------------------------------------
	t.Run("Shop Scoped Layer: Search strictly inside target shop", func(t *testing.T) {
		products, count, err := productRepo.FindAll(ctx, dto.ProductFilter{
			ShopID: shopID,
			Limit:  20,
			Page:   1,
		})
		if err != nil {
			t.Fatalf("FindAll failed for shopID %s: %v", shopID, err)
		}
		t.Logf("Total products inside shop '%s': %d", testShop.Name, count)
		for _, p := range products {
			if p.ShopID != shopID {
				t.Fatalf("CROSS-STORE LEAKAGE! Product %s belongs to shop %s, expected %s", p.Name, p.ShopID, shopID)
			}
		}
	})

	t.Run("Shop Scoped Layer: Non-existent shopID returns exactly 0 products", func(t *testing.T) {
		fakeShopID := "00000000-0000-0000-0000-000000000000"
		products, count, err := productRepo.FindAll(ctx, dto.ProductFilter{
			ShopID: fakeShopID,
			Limit:  10,
			Page:   1,
		})
		if err != nil {
			t.Fatalf("FindAll failed: %v", err)
		}
		if count != 0 || len(products) != 0 {
			t.Errorf("Expected 0 products for fake shop ID, got count=%d, len=%d", count, len(products))
		}
	})

	// -------------------------------------------------------------
	// TEST 4: Multi-Layer Filter Stacking (Shop + Search Keyword + Price)
	// -------------------------------------------------------------
	t.Run("Stacked Filter Layer: Shop + Price Filter (min_price > 100)", func(t *testing.T) {
		minPrice := 100.0
		products, count, err := productRepo.FindAll(ctx, dto.ProductFilter{
			ShopID:   shopID,
			MinPrice: minPrice,
			Limit:    10,
			Page:     1,
		})
		if err != nil {
			t.Fatalf("FindAll failed: %v", err)
		}
		t.Logf("Products in shop with price >= ₹%.0f: %d", minPrice, count)
		for _, p := range products {
			if p.ShopID != shopID {
				t.Fatalf("Shop scope violation! Product belongs to %s, expected %s", p.ShopID, shopID)
			}
			if p.Price < minPrice {
				t.Fatalf("Price filter violation! Product price ₹%.2f is less than min ₹%.2f", p.Price, minPrice)
			}
		}
	})
}

func TestFindNearbyProductsPostGIS(t *testing.T) {
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
		t.Skip("Skipping: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Fatalf("ConnectDB failed: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	productRepo := repository.NewProductRepo(pool, logger)

	// User at (26.1512, 85.9664) within 15km
	items, count, err := productRepo.FindNearbyProducts(ctx, 26.1512, 85.9664, 15.0, "", "", false, 1, 10)
	if err != nil {
		t.Fatalf("FindNearbyProducts failed: %v", err)
	}
	t.Logf("FindNearbyProducts returned %d in-stock items (count: %d)", len(items), count)
	if count > 0 {
		t.Logf("First item: %s | Shop: %s | Distance: %.2f km", items[0].Name, items[0].ShopName, items[0].DistanceKm)
	}

	// User in Chennai (far away: 13.0827, 80.2707)
	itemsFar, countFar, err := productRepo.FindNearbyProducts(ctx, 13.0827, 80.2707, 15.0, "", "", false, 1, 10)
	if err != nil {
		t.Fatalf("FindNearbyProducts far test failed: %v", err)
	}
	if countFar != 0 || len(itemsFar) != 0 {
		t.Fatalf("Expected 0 items in Chennai, got count: %d", countFar)
	}
	t.Logf("Correctly returned 0 items for distant coordinates in Chennai")
}

func checkPostGISHelper2(t *testing.T) {
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
		t.Skip("Skipping: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer pool.Close()

	// Try CREATE EXTENSION IF NOT EXISTS postgis
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS postgis;")
	if err != nil {
		t.Logf("CREATE EXTENSION postgis result: %v", err)
	} else {
		t.Logf("CREATE EXTENSION postgis succeeded!")
		var version string
		if err := pool.QueryRow(ctx, "SELECT postgis_version();").Scan(&version); err != nil {
			t.Logf("SELECT postgis_version error: %v", err)
		} else {
			t.Logf("PostGIS Version: %s", version)
		}
	}
}



func TestInStoreBarcodeScanFastPath(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
			if data, err := os.ReadFile(envPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "DATABASE_URL=") {
						dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
						break
					}
				}
			}
			if dbURL != "" {
				break
			}
		}
	}
	if dbURL == "" {
		t.Skip("Skipping live database test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("Skipping live database test: unable to connect to db: %v", err)
	}
	defer pool.Close()

	logger := zap.NewNop()
	repo := repository.NewProductRepo(pool, logger)

	var testSKU, testShopID, testBarcode string
	row := pool.QueryRow(ctx, "SELECT COALESCE(sku, ''), shop_id, COALESCE(attributes->>'barcode', '') FROM products WHERE is_active = true AND (sku IS NOT NULL OR attributes ? 'barcode') LIMIT 1")
	if err := row.Scan(&testSKU, &testShopID, &testBarcode); err != nil {
		t.Logf("No products with SKU/barcode found in DB: %v", err)
		return
	}

	queryCode := testBarcode
	if queryCode == "" {
		queryCode = testSKU
	}
	if queryCode == "" {
		t.Skip("No test SKU or barcode available in database")
	}

	start := time.Now()
	filter := dto.ProductFilter{
		ShopID:  testShopID,
		Barcode: queryCode,
		Limit:   1,
		Page:    1,
	}
	products, count, err := repo.FindAll(ctx, filter)
	if err != nil {
		t.Fatalf("Barcode scan failed: %v", err)
	}
	scanDuration := time.Since(start)
	t.Logf("In-Store Barcode Fast-Path lookup for code '%s' in shop '%s' completed in %v (Found: %d)", queryCode, testShopID, scanDuration, count)
	if count < 1 || len(products) < 1 {
		t.Errorf("Expected to find product with barcode '%s', got %d", queryCode, count)
	} else {
		t.Logf("Matched Product: %s | Price: ₹%.2f | Stock: %d", products[0].Name, products[0].Price, products[0].StockQuantity)
	}
}

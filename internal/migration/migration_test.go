package migration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/migration"
	"shopMe/internal/utils"
)

func TestPostGISMigrationAndSchema(t *testing.T) {
	for _, envPath := range []string{"../../.env", "../.env", ".env"} {
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

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Fatalf("ConnectDB failed: %v", err)
	}
	defer pool.Close()

	// 1. Run migrations
	if err := migration.RunAutoMigrations(pool); err != nil {
		t.Fatalf("RunAutoMigrations failed: %v", err)
	}

	// 2. Run EnsureSchemaColumns
	migration.EnsureSchemaColumns(pool)

	// 3. Verify location column exists on shops
	var colType string
	err = pool.QueryRow(ctx, `
		SELECT udt_name 
		FROM information_schema.columns 
		WHERE table_name = 'shops' AND column_name = 'location';
	`).Scan(&colType)
	if err != nil {
		t.Fatalf("Failed to verify location column on shops: %v", err)
	}
	t.Logf("shops.location column verified with udt_name: %s", colType)

	// 4. Verify GIST index exists
	var indexName string
	err = pool.QueryRow(ctx, `
		SELECT indexname 
		FROM pg_indexes 
		WHERE tablename = 'shops' AND indexname = 'idx_shops_location_gist';
	`).Scan(&indexName)
	if err != nil {
		t.Fatalf("Failed to verify idx_shops_location_gist index: %v", err)
	}
	t.Logf("Index verified: %s", indexName)

	// 5. Verify trigger exists
	var triggerName string
	err = pool.QueryRow(ctx, `
		SELECT tgname 
		FROM pg_trigger 
		WHERE tgname = 'trg_sync_shop_location';
	`).Scan(&triggerName)
	if err != nil {
		t.Fatalf("Failed to verify trg_sync_shop_location trigger: %v", err)
	}
	t.Logf("Trigger verified: %s", triggerName)

	// 6. Test ST_DWithin and ST_Distance query directly
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM shops 
		WHERE location IS NOT NULL 
		  AND ST_DWithin(location, ST_SetSRID(ST_MakePoint(85.9664, 26.1512), 4326)::geography, 15000);
	`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to execute ST_DWithin test query: %v", err)
	}
	t.Logf("ST_DWithin returned %d shops within 15km of (26.1512, 85.9664)", count)
}


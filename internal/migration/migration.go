package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunAutoMigrations(dbURL string) error {
	m, err := migrate.New("file://internal/migration", dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

// EnsureSchemaColumns runs fast, idempotent ALTER TABLE statements to guarantee all critical columns exist.
func EnsureSchemaColumns(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queries := []string{
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS floor_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS allow_bargain BOOLEAN DEFAULT false;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS is_price_public BOOLEAN DEFAULT true;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS compare_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;",
		"UPDATE users SET phone = NULL WHERE phone = '';",
		"ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_key;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users(phone) WHERE phone IS NOT NULL AND phone != '';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS upi_id VARCHAR(100) DEFAULT '';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS opening_time VARCHAR(10) DEFAULT '09:00';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS closing_time VARCHAR(10) DEFAULT '21:00';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS weekly_off VARCHAR(20) DEFAULT '';",
		"ALTER TABLE shops ALTER COLUMN phone DROP NOT NULL;",
		"ALTER TABLE shops ALTER COLUMN pincode DROP NOT NULL;",
		"ALTER TABLE shops ALTER COLUMN city DROP NOT NULL;",
	}

	for _, q := range queries {
		_, _ = pool.Exec(ctx, q)
	}
}

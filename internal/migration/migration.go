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
	}

	for _, q := range queries {
		_, _ = pool.Exec(ctx, q)
	}
}

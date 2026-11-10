package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(database string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(database)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	log.Printf("attempting database connection: host=%s port=%d user=%s dbname=%s",
		config.ConnConfig.Host,
		config.ConnConfig.Port,
		config.ConnConfig.User,
		config.ConnConfig.Database,
	)

	// Supabase Pooler (Port 6543 / PgBouncer) Compatibility:
	// PgBouncer in transaction mode does not support session-level prepared statements.
	// Using QueryExecModeExec prevents "prepared statement already exists" errors under high load.
	if strings.Contains(database, "6543") || strings.Contains(database, "pooler") {
		config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	// Production pool tuning for low latency & concurrency (Tuned for 1GB RAM VPS)
	config.MaxConns = 18
	config.MinConns = 2
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 15 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	var lastErr error
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		p, err := pgxpool.NewWithConfig(ctx, config)
		if err == nil {
			if pingErr := p.Ping(ctx); pingErr == nil {
				log.Println("database connection pool initialized successfully!")
				return p, nil
			} else {
				lastErr = pingErr
				p.Close()
			}
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			log.Printf("database connection attempt %d/%d failed (%v), retrying in 1s...", attempt, maxAttempts, lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}
	}

	return nil, fmt.Errorf("unable to connect to database at host=%s after %d attempts: %w", config.ConnConfig.Host, maxAttempts, lastErr)
}



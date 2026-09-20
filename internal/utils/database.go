package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(database string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

	// Production pool tuning for low latency & concurrency
	config.MaxConns = 30
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create database connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database at host=%s: %w", config.ConnConfig.Host, err)
	}

	log.Println("database connection pool initialized successfully!")
	return pool, nil
}

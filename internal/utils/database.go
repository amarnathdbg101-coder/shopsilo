package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(database string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(database)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Serverless & Neon-friendly pool tuning:
	// - MinConns = 0 allows pool to scale down to 0 connections when idle (Neon scale-to-zero)
	// - MaxConnIdleTime = 1m closes unused connections promptly
	// - HealthCheckPeriod = 0 disables background ping queries that keep database awake
	config.MaxConns = 15
	config.MinConns = 0
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute
	config.HealthCheckPeriod = 0

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create database connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("database connection pool initialized successfully (serverless scale-to-zero optimized)!")
	return pool, nil
}

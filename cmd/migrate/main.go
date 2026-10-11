package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"shopMe/internal/migration"
	"shopMe/internal/utils"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("use: migrate up | version")
		return
	}

	cfg := utils.MustLoad()
	db, err := utils.ConnectDB(cfg.DBURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	switch os.Args[1] {
	case "up":
		if err := migration.RunAutoMigrations(db); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
		migration.EnsureSchemaColumns(db)
		log.Println("All migrations and schema optimizations verified successfully!")
	case "version":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var v int64
		var dirty bool
		err := db.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1;").Scan(&v, &dirty)
		if err != nil {
			log.Fatalf("failed to query version: %v", err)
		}
		fmt.Printf("Current schema_migrations version: %d, Dirty: %v\n", v, dirty)
	default:
		log.Fatalf("unknown migration command: %s (supported: up, version)", os.Args[1])
	}
}

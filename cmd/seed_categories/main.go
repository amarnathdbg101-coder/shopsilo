package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"shopMe/internal/handler/repository"
)

func main() {
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DATABASE_URL")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	logger, _ := zap.NewDevelopment()
	repo := repository.NewCategoryRepo(pool, logger)

	cats, err := repo.FindAll(ctx)
	if err != nil {
		log.Fatalf("FindAll failed: %v", err)
	}

	fmt.Printf("FindAll returned %d categories:\n", len(cats))
	for _, c := range cats {
		fmt.Printf(" - %s (%s) [ID: %s]\n", c.Name, c.Slug, c.ID)
	}
}

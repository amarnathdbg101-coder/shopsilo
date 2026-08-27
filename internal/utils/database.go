package utils

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(database string) *pgxpool.Pool{
  ctx := context.Background()
  config,err := pgxpool.ParseConfig(database)
  if err != nil {
    log.Fatalf("database url parse nhi ho raha hai:%v",err)
  }

  config.MaxConnIdleTime=30*time.Minute
  config.MaxConns = 25
  config.MinConns = 5
  config.MaxConnLifetime = time.Hour

  pool,err := pgxpool.NewWithConfig(ctx,config)
  if err != nil{
    log.Fatalf("unable to create database connection:%v",err)
  }
  if err := pool.Ping(ctx); err != nil{
    log.Fatalf("unble to ping database:%v",err)
  }

  log.Println("database connection successful !")

  return pool
}
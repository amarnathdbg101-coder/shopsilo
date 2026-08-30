package utils

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(database string) *pgxpool.Pool{

  pool,err := pgxpool.New(context.Background(),database)
  if err != nil{
    log.Fatalf("unable to create database connection:%v",err)
  }
  if err := pool.Ping(context.Background()); err != nil{
    log.Fatalf("unble to ping database:%v",err)
  }

  log.Println("database connection successful !")

  return pool
}
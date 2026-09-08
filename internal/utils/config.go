// Package utils contains common project helpers and configuration loading.
package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv" // for local .env file
)

type Config struct {
	Port      string `koanf:"PORT"`
	DBURL     string `koanf:"DATABASE_URL"`
	Jwt       string `koanf:"JWT_SECRET"`
	Bucket    string `koanf:"R2_BUCKET"`
	AccessKey string `koanf:"R2_ACCESS_KEY"`
	SecretKey string `koanf:"R2_SECRET_KEY"`
	Endpoint  string `koanf:"R2_ENDPOINT"`
	PublicURL string `koanf:"R2_PUBLIC_URL"`
}

func MustLoad() Config {
	// Load .env file only in local dev (ignore error if file not found)
	_ = godotenv.Load(".env")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalf("port is required")
	}

	database := os.Getenv("DATABASE_URL")
	if database == "" {
		log.Fatal("database url is require")
	}
	jwt := os.Getenv("JWT_SECRET")
	if jwt == "" {
		log.Fatal("jwt is require")
	}
	bucket := os.Getenv("R2_BUCKET")
	if bucket == "" {
		log.Fatalf("R2 BUCKET is required")
	}

	accessKey := os.Getenv("R2_ACCESS_KEY")
	if accessKey == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	secertKey := os.Getenv("R2_SECRET_KEY")
	if secertKey == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	r2Endpoint := os.Getenv("R2_ENDPOINT")
	if r2Endpoint == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	return Config{
		Port:      port,
		DBURL:     database,
		Jwt:       jwt,
		AccessKey: accessKey,
		Endpoint:  r2Endpoint,
		SecretKey: secertKey,
		Bucket:    bucket,
		PublicURL: os.Getenv("R2_PUBLIC_URL"),
	}
}

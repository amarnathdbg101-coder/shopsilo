package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv" // for local .env file
)

type Config struct {
	Port      string `koanf:"PORT"`
	DbUrl     string `koanf:"DATABASE_URL"`
	Jwt       string `koanf:"JWT_SECRET"`
	Bucket    string `koanf:"R2_BUCKET"`
	AccessKey string `koanf:"R2_ACCESS_KEY"`
	SecretKey string `koanf:"R2_SECRET_KEY"`
	Endpoint  string `koanf:"R2_ENDPOINT"`
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

	access_key := os.Getenv("R2_ACCESS_KEY")
	if access_key == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	secert_key := os.Getenv("R2_SECRET_KEY")
	if secert_key == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	r2_endpoint := os.Getenv("R2_ENDPOINT")
	if r2_endpoint == "" {
		log.Fatalf("R2 BUCKET is required")
	}
	return Config{
		Port:  port,
		DbUrl: database,
		Jwt:   jwt,
        AccessKey: access_key,
        Endpoint: r2_endpoint,
        SecretKey: secert_key,
        Bucket: bucket,
	}
}

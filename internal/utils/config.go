// Package utils contains common project helpers and configuration loading.
package utils

import (
	"log"
	"os"
	"strings"
	"sync"

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
	PublicURL     string   `koanf:"R2_PUBLIC_URL"`
	GeminiAPIKeys []string `koanf:"GEMINI_API_KEYS"`
}

var (
	configOnce   sync.Once
	cachedConfig Config
)

func MustLoad() Config {
	configOnce.Do(func() {
		// Load .env file (supports local dev and VPS deployment paths)
		_ = godotenv.Load(".env", "/root/shopsilo/.env", "/root/shopsilo/cmd/api/.env", "../.env")

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
			log.Fatal("jwt is required")
		}
		if len(jwt) < 16 {
			log.Printf("[SECURITY WARNING] JWT_SECRET is shorter than 16 characters! Use a strong secret in production.")
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

		cachedConfig = Config{
			Port:          port,
			DBURL:         database,
			Jwt:           jwt,
			AccessKey:     accessKey,
			Endpoint:      r2Endpoint,
			SecretKey:     secertKey,
			Bucket:        bucket,
			PublicURL:     os.Getenv("R2_PUBLIC_URL"),
			GeminiAPIKeys: getCombinedGeminiKeys(),
		}
	})

	return cachedConfig
}

func getCombinedGeminiKeys() []string {
	k1 := os.Getenv("GEMINI_API_KEYS")
	k2 := os.Getenv("GEMINI_API_KEY")
	var combined []string
	seen := make(map[string]bool)

	for _, raw := range []string{k1, k2} {
		for _, k := range parseCommaList(raw) {
			if k != "" && !seen[k] {
				seen[k] = true
				combined = append(combined, k)
			}
		}
	}
	return combined
}

func parseCommaList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range strings.Split(raw, ",") {
		clean := strings.TrimSpace(strings.Trim(p, `"'`))
		if clean != "" {
			parts = append(parts, clean)
		}
	}
	return parts
}

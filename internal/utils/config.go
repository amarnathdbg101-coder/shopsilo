// Package utils contains common project helpers and configuration loading.
package utils

import (
	"log"
	"strings"
	"sync"

	goenv "github.com/amarnathdbg101-coder/go/go-env"
	"github.com/amarnathdbg101-coder/go/go-envvalidator"
)

type Config struct {
	Port          string   `koanf:"PORT"`
	DBURL         string   `koanf:"DATABASE_URL"`
	Jwt           string   `koanf:"JWT_SECRET"`
	Bucket        string   `koanf:"R2_BUCKET"`
	AccessKey     string   `koanf:"R2_ACCESS_KEY"`
	SecretKey     string   `koanf:"R2_SECRET_KEY"`
	Endpoint      string   `koanf:"R2_ENDPOINT"`
	PublicURL     string   `koanf:"R2_PUBLIC_URL"`
	GeminiAPIKeys []string `koanf:"GEMINI_API_KEYS"`
}

var (
	configOnce   sync.Once
	cachedConfig Config
)

// MustLoad loads, validates, and caches application configuration using zero-dep go-env and go-envvalidator.
func MustLoad() Config {
	configOnce.Do(func() {
		// 1. Zero-dependency .env loader (searches standard and deployment locations)
		_ = goenv.Load(".env", "../.env", "../../.env", "../../../.env", "/root/shopsilo/.env", "/root/shopsilo/cmd/api/.env")

		// 2. Declarative Schema-Driven Validation
		schema := envvalidator.New()

		// Network & Database
		schema.Require("PORT").AsInt().Between(1, 65535)
		schema.Require("DATABASE_URL")

		// Security & Storage
		schema.Require("JWT_SECRET").MinLength(16).Masked()
		schema.Require("R2_BUCKET")
		schema.Require("R2_ACCESS_KEY").Masked()
		schema.Require("R2_SECRET_KEY").Masked()
		schema.Require("R2_ENDPOINT")

		// Optional Fields
		schema.Optional("R2_PUBLIC_URL", "")
		schema.Optional("GEMINI_API_KEYS", "")
		schema.Optional("GEMINI_API_KEY", "")

		env, err := schema.Validate(nil)
		if err != nil {
			log.Fatalf("[FATAL] Configuration validation failed:\n%v", err)
		}

		cachedConfig = Config{
			Port:          env.GetString("PORT"),
			DBURL:         env.GetString("DATABASE_URL"),
			Jwt:           env.GetString("JWT_SECRET"),
			AccessKey:     env.GetString("R2_ACCESS_KEY"),
			Endpoint:      env.GetString("R2_ENDPOINT"),
			SecretKey:     env.GetString("R2_SECRET_KEY"),
			Bucket:        env.GetString("R2_BUCKET"),
			PublicURL:     env.GetString("R2_PUBLIC_URL"),
			GeminiAPIKeys: getCombinedGeminiKeys(),
		}
	})

	return cachedConfig
}

func getCombinedGeminiKeys() []string {
	k1 := goenv.GetString("GEMINI_API_KEYS", "")
	k2 := goenv.GetString("GEMINI_API_KEY", "")
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
	var parts []string
	for _, p := range strings.Split(raw, ",") {
		clean := strings.TrimSpace(strings.Trim(p, `"'`))
		if clean != "" {
			parts = append(parts, clean)
		}
	}
	return parts
}

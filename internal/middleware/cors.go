package middleware

import (
	"net/http"
	"os"
	"strings"

	gocors "github.com/amarnathdbg101-coder/go/go-cors"
)

// buildCORSConfig parses ALLOWED_ORIGINS and creates an optimized pre-computed CORS handler.
func buildCORSConfig() gocors.Config {
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var origins []string

	if allowedOriginsEnv == "" || allowedOriginsEnv == "*" {
		origins = []string{"*"}
	} else {
		for _, o := range strings.Split(allowedOriginsEnv, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		// Always permit standard development localhosts & emulators
		origins = append(origins,
			"http://localhost*",
			"https://localhost*",
			"http://127.0.0.1*",
			"https://127.0.0.1*",
			"http://10.0.2.2*", // Android emulator
		)
	}

	return gocors.Config{
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Content-Type", "Authorization", "X-Requested-With",
			"Accept", "X-Device-Fingerprint", "X-Idempotency-Key", "Idempotency-Key",
		},
		ExposedHeaders:   []string{"Link", "X-Total-Count", "X-Cache"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

var defaultCORS = gocors.New(buildCORSConfig())

// CORS handles Cross-Origin Resource Sharing with zero-allocation headers and injects standard OWASP Security Headers.
// Powered by zero-dep go-cors with preflight OPTIONS short-circuiting.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Production OWASP Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; sandbox;")
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
		w.Header().Set("X-Download-Options", "noopen")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 2. Delegate to high-performance go-cors engine
		defaultCORS.Handler(next).ServeHTTP(w, r)
	})
}

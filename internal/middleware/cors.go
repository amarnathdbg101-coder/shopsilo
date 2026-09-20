package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORS handles Cross-Origin Resource Sharing and injects standard Security Headers
func CORS(next http.Handler) http.Handler {
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedList []string
	if allowedOriginsEnv != "" && allowedOriginsEnv != "*" {
		for _, o := range strings.Split(allowedOriginsEnv, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				allowedList = append(allowedList, trimmed)
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 1. CORS Configuration
		if allowedOriginsEnv == "" || allowedOriginsEnv == "*" {
			// Permissive mode for local development
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			// Production whitelist mode
			matched := false
			for _, allowed := range allowedList {
				if strings.EqualFold(origin, allowed) {
					matched = true
					break
				}
			}

			if matched {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, X-Device-Fingerprint, X-Idempotency-Key, Idempotency-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 2. Production Security Headers (Point 19)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

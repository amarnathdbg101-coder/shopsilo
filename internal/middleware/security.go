package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders injects OWASP production-grade HTTP security headers
// to protect against clickjacking, MIME-sniffing, XSS, and downgrade attacks.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// 2. Prevent Clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// 3. XSS Filter protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// 4. Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 5. Modern Permissions-Policy
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), payment=()")

		// 6. Strict-Transport-Security (HSTS) for HTTPS & SSL-terminated reverse proxies
		isTLS := r.TLS != nil ||
			strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") ||
			strings.Contains(r.Header.Get("CF-Visitor"), "https")

		if isTLS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		next.ServeHTTP(w, r)
	})
}

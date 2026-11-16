package auditlog

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ActorExtractorFunc extracts the current authenticated user's ID and Email from the request.
type ActorExtractorFunc func(r *http.Request) (actorID, actorEmail string)

// MiddlewareOption configures HTTP audit logging middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	extractor ActorExtractorFunc
	skipPaths []string
}

// WithActorExtractor sets a custom user ID / email extractor (e.g. from JWT context).
func WithActorExtractor(fn ActorExtractorFunc) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.extractor = fn
	}
}

// WithSkipPaths ignores audit logging for certain endpoints (e.g. health checks).
func WithSkipPaths(paths ...string) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.skipPaths = append(c.skipPaths, paths...)
	}
}

// statusWriter wraps http.ResponseWriter to capture response status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Middleware creates standard HTTP middleware that records incoming API requests into the audit log.
func Middleware(engine *Engine, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := middlewareConfig{
		extractor: func(r *http.Request) (string, string) {
			return "", ""
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip ignored paths
			for _, skip := range cfg.skipPaths {
				if strings.HasPrefix(r.URL.Path, skip) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(sw, r)

			// Record audit event asynchronously
			actorID, actorEmail := cfg.extractor(r)

			entry := NewEntry(
				fmt.Sprintf("http.%s", strings.ToLower(r.Method)),
				"api_route",
				r.URL.Path,
			)
			entry.ActorID = actorID
			entry.ActorEmail = actorEmail
			entry.IPAddress = extractIP(r)
			entry.UserAgent = r.UserAgent()
			entry.Metadata = map[string]any{
				"status_code":   sw.statusCode,
				"duration_ms":   time.Since(start).Milliseconds(),
				"method":        r.Method,
				"query":         r.URL.RawQuery,
			}

			_ = engine.Log(entry)
		})
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

package gocors

import (
	"net/http"
	"strconv"
	"strings"
)

// Config configures cross-origin resource sharing headers.
type Config struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // Seconds for Access-Control-Max-Age
}

// DefaultConfig provides standard web and mobile client defaults.
var DefaultConfig = Config{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "Origin"},
	ExposedHeaders:   []string{"Link", "X-Total-Count", "X-Request-ID"},
	AllowCredentials: true,
	MaxAge:           86400, // 24 hours
}

type wildcardRule struct {
	scheme string // "https://" or "http://" or ""
	suffix string // ".shopsilo.com"
}

// CORS holds pre-computed header values for zero-allocation request handling.
type CORS struct {
	cfg            Config
	allowAll       bool
	allowedOrigins map[string]bool
	wildcardRules  []wildcardRule
	methodsHeader  string
	headersHeader  string
	exposedHeader  string
	maxAgeHeader   string
}

// New initializes CORS middleware with pre-computed headers.
func New(cfg ...Config) *CORS {
	c := DefaultConfig
	if len(cfg) > 0 {
		c = cfg[0]
	}

	cors := &CORS{
		cfg:            c,
		allowedOrigins: make(map[string]bool),
		methodsHeader:  strings.Join(c.AllowedMethods, ", "),
		headersHeader:  strings.Join(c.AllowedHeaders, ", "),
		exposedHeader:  strings.Join(c.ExposedHeaders, ", "),
		maxAgeHeader:   strconv.Itoa(c.MaxAge),
	}

	for _, o := range c.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "*" {
			cors.allowAll = true
		} else if strings.Contains(o, "://*.") {
			parts := strings.Split(o, "://*.")
			cors.wildcardRules = append(cors.wildcardRules, wildcardRule{
				scheme: parts[0] + "://",
				suffix: "." + parts[1],
			})
		} else if strings.HasPrefix(o, "*.") {
			cors.wildcardRules = append(cors.wildcardRules, wildcardRule{
				scheme: "",
				suffix: strings.TrimPrefix(o, "*"),
			})
		} else {
			cors.allowedOrigins[strings.ToLower(o)] = true
		}
	}

	return cors
}

// IsOriginAllowed checks if the origin is permitted by whitelist or wildcard rules.
func (c *CORS) IsOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if c.allowAll {
		return true
	}
	lower := strings.ToLower(origin)
	if c.allowedOrigins[lower] {
		return true
	}

	for _, w := range c.wildcardRules {
		if w.scheme != "" && !strings.HasPrefix(lower, w.scheme) {
			continue
		}
		if strings.HasSuffix(lower, w.suffix) {
			return true
		}
	}

	return false
}

// Handler returns standard HTTP middleware.
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// If no Origin header, request is not CORS
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !c.IsOriginAllowed(origin) {
			// Origin disallowed; continue without CORS headers
			next.ServeHTTP(w, r)
			return
		}

		// Set Origin header
		if c.allowAll && !c.cfg.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		if c.cfg.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.exposedHeader != "" {
			w.Header().Set("Access-Control-Expose-Headers", c.exposedHeader)
		}

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")

			if c.methodsHeader != "" {
				w.Header().Set("Access-Control-Allow-Methods", c.methodsHeader)
			}
			if c.headersHeader != "" {
				w.Header().Set("Access-Control-Allow-Headers", c.headersHeader)
			}
			if c.cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", c.maxAgeHeader)
			}

			// Preflight terminates here with 204 No Content
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware creates a handler from config directly.
func Middleware(cfg ...Config) func(http.Handler) http.Handler {
	c := New(cfg...)
	return c.Handler
}

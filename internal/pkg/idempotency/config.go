package idempotency

import (
	"net/http"
	"time"
)

// Option represents a functional option for configuring the Idempotency Engine.
type Option func(*Config)

// Hooks defines zero-dependency observability callbacks for metrics, tracing, and logging.
type Hooks struct {
	// OnHit is invoked when a matching response is served directly from cache.
	OnHit func(key string, statusCode int)

	// OnConflict is invoked when a concurrent in-flight collision returns HTTP 409.
	OnConflict func(key string)

	// OnSaved is invoked when a completed response is cached.
	OnSaved func(key string, statusCode int)

	// OnError is invoked when storage driver encounters an unexpected error.
	OnError func(key string, err error)
}

// Config holds the configuration settings for the Idempotency engine and middleware.
type Config struct {
	// KeyHeader is the primary header to extract the idempotency key (Default: "X-Idempotency-Key").
	KeyHeader string

	// AltKeyHeader is the fallback header (Default: "Idempotency-Key").
	AltKeyHeader string

	// TTL specifies how long completed responses are cached (Default: 5 minutes).
	TTL time.Duration

	// LockTTL specifies how long an in-flight execution lock is held (Default: 30 seconds).
	LockTTL time.Duration

	// Storage defines the persistence and locking backend.
	Storage Storage

	// KeyGenerator customizes composite key creation (e.g., combining Method, Path, UserID, Key).
	KeyGenerator func(r *http.Request, rawKey string) string

	// ShouldCache determines whether a given response status code should be persisted (Default: statusCode < 500).
	ShouldCache func(statusCode int) bool

	// Hooks provides optional observability callbacks.
	Hooks Hooks
}

// DefaultConfig returns safe production defaults.
func DefaultConfig() *Config {
	return &Config{
		KeyHeader:    "X-Idempotency-Key",
		AltKeyHeader: "Idempotency-Key",
		TTL:          5 * time.Minute,
		LockTTL:      30 * time.Second,
		Storage:      NewMemoryStorage(1 * time.Minute),
		KeyGenerator: func(r *http.Request, rawKey string) string {
			return r.Method + ":" + r.URL.Path + ":" + rawKey
		},
		ShouldCache: func(statusCode int) bool {
			return statusCode < 500
		},
	}
}

// Validate ensures the configuration is sound.
func (c *Config) Validate() error {
	if c.KeyHeader == "" {
		return ErrInvalidConfig
	}
	if c.TTL <= 0 || c.LockTTL <= 0 {
		return ErrInvalidConfig
	}
	if c.Storage == nil {
		return ErrInvalidConfig
	}
	if c.KeyGenerator == nil {
		return ErrInvalidConfig
	}
	if c.ShouldCache == nil {
		return ErrInvalidConfig
	}
	return nil
}

// WithHeader sets a custom HTTP header for the idempotency key.
func WithHeader(header string) Option {
	return func(c *Config) {
		if header != "" {
			c.KeyHeader = header
		}
	}
}

// WithTTL sets the duration completed responses remain cached.
func WithTTL(ttl time.Duration) Option {
	return func(c *Config) {
		if ttl > 0 {
			c.TTL = ttl
		}
	}
}

// WithLockTTL sets the duration an in-flight lock is maintained.
func WithLockTTL(lockTTL time.Duration) Option {
	return func(c *Config) {
		if lockTTL > 0 {
			c.LockTTL = lockTTL
		}
	}
}

// WithStorage sets a custom storage driver (e.g., Redis, DB, or custom MemoryStorage).
func WithStorage(storage Storage) Option {
	return func(c *Config) {
		if storage != nil {
			c.Storage = storage
		}
	}
}

// WithKeyGenerator specifies a custom function to compose unique idempotency keys.
func WithKeyGenerator(fn func(r *http.Request, rawKey string) string) Option {
	return func(c *Config) {
		if fn != nil {
			c.KeyGenerator = fn
		}
	}
}

// WithStatusFilter configures which HTTP status codes qualify for caching.
func WithStatusFilter(fn func(statusCode int) bool) Option {
	return func(c *Config) {
		if fn != nil {
			c.ShouldCache = fn
		}
	}
}

// WithHooks attaches custom telemetry and observability hooks.
func WithHooks(hooks Hooks) Option {
	return func(c *Config) {
		c.Hooks = hooks
	}
}


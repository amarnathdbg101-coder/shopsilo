package idempotency

import (
	"context"
	"net/http"
	"time"
)

// Record represents a cached HTTP response associated with an idempotency key.
type Record struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
	CreatedAt  time.Time   `json:"created_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
}

// Storage defines the driver contract for idempotency persistence and distributed locking.
type Storage interface {
	// Lock attempts to acquire an in-flight lock for the given key to prevent concurrent executions.
	// Returns true if lock was acquired, false if another request is currently executing.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Unlock releases an in-flight lock after execution completes.
	Unlock(ctx context.Context, key string) error

	// Get retrieves a completed response record for the given key. Returns (nil, nil) if key does not exist.
	Get(ctx context.Context, key string) (*Record, error)

	// Set stores the completed response record with the specified TTL.
	Set(ctx context.Context, key string, record *Record, ttl time.Duration) error

	// Delete removes a record from storage.
	Delete(ctx context.Context, key string) error

	// Close releases any background resources, goroutines, or connections.
	Close() error
}

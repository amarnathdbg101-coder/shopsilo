package middleware

import (
	"net/http"
	"time"

	"github.com/amarnathdbg101-coder/go/go-idempotency"
)

var defaultIdempotencyEngine, _ = idempotency.New(
	idempotency.WithTTL(60*time.Second),
	idempotency.WithLockTTL(30*time.Second),
	idempotency.WithKeyGenerator(func(r *http.Request, rawKey string) string {
		ip := ExtractIP(r)
		return ip + ":" + r.Method + ":" + r.URL.Path + ":" + rawKey
	}),
)

// IdempotencyGuard catches X-Idempotency-Key / Idempotency-Key headers on mutating requests
// (POST, PUT, PATCH, DELETE) and guarantees exactly-once execution semantics.
// Powered by zero-dep go-idempotency with single-flight concurrency locking (409 Conflict defense)
// and thread-safe automatic TTL memory eviction.
func IdempotencyGuard(next http.Handler) http.Handler {
	if defaultIdempotencyEngine == nil {
		return next
	}
	return defaultIdempotencyEngine.Middleware(next)
}

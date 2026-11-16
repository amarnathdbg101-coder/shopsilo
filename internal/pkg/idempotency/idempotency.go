// Package idempotency provides production-grade HTTP request deduplication and response caching.
// It guards mutating endpoints (POST/PUT/PATCH/DELETE) against double-clicks, duplicate orders,
// and network retry storms with zero external dependencies.
package idempotency

import "net/http"

// New initializes an Idempotency Engine with the provided functional options.
func New(opts ...Option) (*Engine, error) {
	return NewEngine(opts...)
}

// Handler returns a standard net/http middleware wrapper from default configuration.
func Handler(opts ...Option) (func(http.Handler) http.Handler, error) {
	eng, err := NewEngine(opts...)
	if err != nil {
		return nil, err
	}
	return eng.Middleware, nil
}

package idempotency

import "errors"

var (
	// ErrKeyMissing indicates that the required idempotency key was not found in request headers.
	ErrKeyMissing = errors.New("idempotency: key is missing in request headers")

	// ErrKeyInFlight indicates that a request with the identical key is already being processed.
	ErrKeyInFlight = errors.New("idempotency: request with this key is already in-flight")

	// ErrStorageClosed indicates that operations were attempted on a closed storage driver.
	ErrStorageClosed = errors.New("idempotency: storage driver is closed")

	// ErrInvalidConfig indicates that invalid configuration parameters were provided.
	ErrInvalidConfig = errors.New("idempotency: invalid configuration parameters")
)

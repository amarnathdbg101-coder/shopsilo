package tinylfu

import "errors"

var (
	// ErrKeyNotFound is returned when the key does not exist in the cache.
	ErrKeyNotFound = errors.New("tinylfu: key not found in cache")

	// ErrInvalidCapacity is returned when initialized with a capacity <= 0.
	ErrInvalidCapacity = errors.New("tinylfu: capacity must be greater than zero")

	// ErrCacheClosed is returned when operating on a closed cache.
	ErrCacheClosed = errors.New("tinylfu: cache is closed")
)

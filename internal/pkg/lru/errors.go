package golru

import "errors"

var (
	// ErrInvalidCapacity indicates LRU cache capacity must be greater than zero.
	ErrInvalidCapacity = errors.New("golru: capacity must be greater than zero")
)

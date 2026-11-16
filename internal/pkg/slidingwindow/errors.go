package slidingwindow

import "errors"

var (
	// ErrInvalidWindow is returned when window duration is non-positive.
	ErrInvalidWindow = errors.New("slidingwindow: window duration must be positive")
	// ErrInvalidBuckets is returned when bucket count is less than 1.
	ErrInvalidBuckets = errors.New("slidingwindow: bucket count must be at least 1")
)

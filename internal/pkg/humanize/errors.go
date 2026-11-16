package humanize

import "errors"

var (
	// ErrInvalidInput is returned when input data cannot be formatted into a human-readable representation.
	ErrInvalidInput = errors.New("humanize: invalid input value")
)

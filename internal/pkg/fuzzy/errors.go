package fuzzy

import "errors"

var (
	// ErrEmptyInput is returned when computing metrics on empty input where disallowed.
	ErrEmptyInput = errors.New("fuzzy: input slice is empty")
)

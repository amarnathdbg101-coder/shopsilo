package bizhours

import "errors"

var (
	// ErrInvalidTimeFormat indicates that an operating time string was malformed (expected "HH:MM").
	ErrInvalidTimeFormat = errors.New("bizhours: invalid time format, expected 'HH:MM' or 'HH:MM AM/PM'")

	// ErrInvalidDay indicates that an unsupported weekday string or index was supplied.
	ErrInvalidDay = errors.New("bizhours: invalid weekday specification")
)

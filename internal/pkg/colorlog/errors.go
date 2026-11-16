package colorlog

import "errors"

var (
	// ErrOddKeyValues is returned or handled when an odd number of key-value parameters are passed.
	ErrOddKeyValues = errors.New("colorlog: keyvals must have an even length")
)

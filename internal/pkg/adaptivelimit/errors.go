package adaptivelimit

import "errors"

var (
	// ErrLimitExceeded is returned when the adaptive concurrency threshold is reached and incoming execution is throttled.
	ErrLimitExceeded = errors.New("adaptivelimit: adaptive concurrency limit reached (system overloaded)")

	// ErrLimiterClosed is returned when attempting to acquire a permit from a closed limiter.
	ErrLimiterClosed = errors.New("adaptivelimit: limiter is closed")

	// ErrTimeout is returned when waiting for an available permit exceeds context deadline.
	ErrTimeout = errors.New("adaptivelimit: timed out waiting for concurrency permit")
)

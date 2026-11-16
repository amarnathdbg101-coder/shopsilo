package goretry

import (
	"errors"
	"fmt"
)

var (
	// ErrMaxAttempts indicates all retry attempts failed.
	ErrMaxAttempts = errors.New("retry: max attempts reached without success")
)

// RetryError wraps the underlying error with attempt details.
type RetryError struct {
	Attempts int
	LastErr  error
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("retry: failed after %d attempts: %v", e.Attempts, e.LastErr)
}

func (e *RetryError) Unwrap() error {
	return e.LastErr
}

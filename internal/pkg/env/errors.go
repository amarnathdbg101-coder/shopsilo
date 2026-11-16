package goenv

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrMissingEnv indicates one or more required environment variables are missing.
	ErrMissingEnv = errors.New("env: required environment variable missing")

	// ErrInvalidType indicates an environment variable could not be parsed into the target type.
	ErrInvalidType = errors.New("env: invalid type conversion")
)

// MissingKeysError wraps a list of all missing required keys.
type MissingKeysError struct {
	Keys []string
}

func (e *MissingKeysError) Error() string {
	return fmt.Sprintf("env: missing required environment variables: [%s]", strings.Join(e.Keys, ", "))
}

func (e *MissingKeysError) Unwrap() error {
	return ErrMissingEnv
}

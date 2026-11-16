package envvalidator

import "errors"

var (
	// ErrValidationFailed is returned when one or more environment rules fail.
	ErrValidationFailed = errors.New("envvalidator: environment validation failed")
	// ErrVarNotFound is returned when querying an unregistered or unset variable.
	ErrVarNotFound = errors.New("envvalidator: variable not found in validated environment")
)

package saga

import "errors"

var (
	// ErrSagaFailed is returned when forward execution of a step fails.
	ErrSagaFailed = errors.New("saga: step execution failed")
	// ErrCompensationFailed is returned when one or more compensation actions fail.
	ErrCompensationFailed = errors.New("saga: compensation action failed")
	// ErrEmptySaga is returned when executing a coordinator with zero registered steps.
	ErrEmptySaga = errors.New("saga: no steps registered in coordinator")
)

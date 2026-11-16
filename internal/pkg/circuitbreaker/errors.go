package gocb

import "errors"

var (
	// ErrCircuitOpen indicates the circuit breaker is Open and fast-failing calls.
	ErrCircuitOpen = errors.New("circuitbreaker: circuit is open")

	// ErrTooManyRequests indicates HalfOpen state has exceeded allowed concurrent probes.
	ErrTooManyRequests = errors.New("circuitbreaker: too many probe requests in half-open state")
)

package auditlog

import "errors"

// Common domain errors for go-auditlog.
var (
	// ErrBufferFull is returned when non-blocking audit logging buffer is saturated.
	ErrBufferFull = errors.New("auditlog: buffer capacity reached")

	// ErrEngineClosed is returned when attempting to log to a stopped engine.
	ErrEngineClosed = errors.New("auditlog: engine has been shut down")

	// ErrStoreNil is returned when engine is initialized without a store implementation.
	ErrStoreNil = errors.New("auditlog: storage driver cannot be nil")
)

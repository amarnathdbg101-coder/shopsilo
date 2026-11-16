package w3ctrace

import "errors"

var (
	// ErrInvalidTraceparent is returned when the traceparent header violates the W3C format.
	ErrInvalidTraceparent = errors.New("w3ctrace: invalid traceparent header format")

	// ErrInvalidTraceID is returned when the trace-id is all zeros or not a 32-character hex.
	ErrInvalidTraceID = errors.New("w3ctrace: invalid trace-id (must be 32 non-zero hex chars)")

	// ErrInvalidSpanID is returned when the span-id is all zeros or not a 16-character hex.
	ErrInvalidSpanID = errors.New("w3ctrace: invalid span-id (must be 16 non-zero hex chars)")

	// ErrSpanEnded is returned when trying to mutate an already ended span.
	ErrSpanEnded = errors.New("w3ctrace: span has already ended")
)

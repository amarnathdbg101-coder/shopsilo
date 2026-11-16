package bulkhead

import "errors"

var (
	// ErrBulkheadFull is returned when max concurrent slots and waiting queue are both exhausted.
	ErrBulkheadFull = errors.New("bulkhead: capacity exceeded, execution rejected")

	// ErrQueueTimeout is returned when a task waited in the queue longer than QueueTimeout.
	ErrQueueTimeout = errors.New("bulkhead: queue wait timeout exceeded")

	// ErrBulkheadClosed is returned when attempting to submit work to a closed bulkhead.
	ErrBulkheadClosed = errors.New("bulkhead: bulkhead is closed")
)

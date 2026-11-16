package goworkerpool

import "errors"

var (
	// ErrPoolClosed occurs when attempting to submit jobs to a stopped worker pool.
	ErrPoolClosed = errors.New("workerpool: pool is closed")

	// ErrQueueFull occurs when submitting with non-blocking mode and the queue buffer is full.
	ErrQueueFull = errors.New("workerpool: job queue is full")

	// ErrJobTimeout occurs when a job execution exceeds its context deadline.
	ErrJobTimeout = errors.New("workerpool: job execution timed out")
)

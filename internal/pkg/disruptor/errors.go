package disruptor

import "errors"

var (
	// ErrBufferFull is returned when non-blocking TryPublish fails due to ring buffer saturation.
	ErrBufferFull = errors.New("disruptor: ring buffer is full")

	// ErrDisruptorClosed is returned when publishing or consuming on a stopped disruptor instance.
	ErrDisruptorClosed = errors.New("disruptor: ring buffer is closed")

	// ErrInvalidCapacity is returned when the capacity is not a positive power of two.
	ErrInvalidCapacity = errors.New("disruptor: capacity must be a positive power of two")
)

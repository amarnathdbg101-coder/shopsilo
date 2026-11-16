package gooutbox

import "errors"

var (
	// ErrStoreNil is returned when engine initialization is attempted without a store.
	ErrStoreNil = errors.New("go-outbox: store cannot be nil")

	// ErrPublisherNil is returned when engine initialization is attempted without a publisher.
	ErrPublisherNil = errors.New("go-outbox: publisher cannot be nil")

	// ErrEmptyTopic is returned when enqueuing an event with an empty topic.
	ErrEmptyTopic = errors.New("go-outbox: event topic cannot be empty")

	// ErrEmptyPayload is returned when enqueuing an event with a nil or empty payload.
	ErrEmptyPayload = errors.New("go-outbox: payload cannot be empty")

	// ErrEngineClosed is returned when attempting to enqueue or process with a stopped engine.
	ErrEngineClosed = errors.New("go-outbox: engine is closed")

	// ErrEventNotFound is returned when operating on a non-existent event ID.
	ErrEventNotFound = errors.New("go-outbox: event not found")
)

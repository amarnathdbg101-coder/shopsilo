package goetag

import "errors"

var (
	// ErrEmptyPayload is returned when attempting to generate an ETag from empty data.
	ErrEmptyPayload = errors.New("go-etag: payload cannot be empty")

	// ErrPreconditionFailed is returned when an If-Match precondition evaluates to false (HTTP 412).
	ErrPreconditionFailed = errors.New("go-etag: precondition failed (mid-air collision detected)")

	// ErrNotModified is returned when an If-None-Match header matches the current ETag (HTTP 304).
	ErrNotModified = errors.New("go-etag: resource not modified")
)

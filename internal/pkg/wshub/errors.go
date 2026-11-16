package gowshub

import "errors"

var (
	// ErrNilConn is returned when attempting to register or write to a nil connection.
	ErrNilConn = errors.New("gowshub: connection cannot be nil")

	// ErrEmptyRoom is returned when attempting to join or broadcast to an empty room name.
	ErrEmptyRoom = errors.New("gowshub: room name cannot be empty")

	// ErrHubClosed is returned when operations are attempted on a closed hub.
	ErrHubClosed = errors.New("gowshub: hub is closed")
)

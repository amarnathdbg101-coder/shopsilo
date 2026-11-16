package supervisor

import "errors"

var (
	// ErrCrashLoopDetected is returned when a worker crashes more than MaxRestarts within RestartWindow.
	ErrCrashLoopDetected = errors.New("supervisor: crash loop detected, aborting restarts")

	// ErrWorkerAlreadyRegistered is returned when a worker with the same name is added twice.
	ErrWorkerAlreadyRegistered = errors.New("supervisor: worker already registered with this name")

	// ErrWorkerNotFound is returned when attempting to stop or restart an unknown worker.
	ErrWorkerNotFound = errors.New("supervisor: worker not found")

	// ErrSupervisorClosed is returned when attempting to start a closed supervisor.
	ErrSupervisorClosed = errors.New("supervisor: supervisor is closed")
)

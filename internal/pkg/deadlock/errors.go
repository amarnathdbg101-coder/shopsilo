package deadlock

import "errors"

var (
	// ErrPotentialDeadlock is triggered when a cycle in the lock-order graph is detected.
	ErrPotentialDeadlock = errors.New("deadlock: potential lock inversion detected")

	// ErrLockTimeout is triggered when a lock is held or waited upon beyond the configured timeout.
	ErrLockTimeout = errors.New("deadlock: lock acquisition timed out (possible starvation)")

	// ErrDoubleUnlock is triggered when Unlock is called on an unheld mutex.
	ErrDoubleUnlock = errors.New("deadlock: unlock of unlocked mutex")
)

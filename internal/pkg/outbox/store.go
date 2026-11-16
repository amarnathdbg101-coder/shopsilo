package gooutbox

import (
	"context"
	"time"
)

// Store defines persistence and atomic locking contracts for outbox events.
type Store interface {
	// Save writes an event to outbox storage. tx may be nil or an active database transaction (*sql.Tx, pgx.Tx, etc.).
	Save(ctx context.Context, tx any, event *Event) error

	// FetchAndLock atomically retrieves a batch of pending or ready-to-retry events and marks them StatusProcessing.
	FetchAndLock(ctx context.Context, batchSize int) ([]*Event, error)

	// MarkCompleted transitions an event to StatusCompleted with a processed timestamp.
	MarkCompleted(ctx context.Context, id string) error

	// MarkFailed records an error, increments the retry counter, and sets the next scheduled retry time.
	MarkFailed(ctx context.Context, id string, err error, nextRetryAfter time.Duration) error

	// MarkDeadLetter moves an event to StatusDeadLetter after max retries are exhausted.
	MarkDeadLetter(ctx context.Context, id string, err error) error
}

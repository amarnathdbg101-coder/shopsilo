// Package app provides centralized application state, dependency container, feature registry, and worker coordination.
package app

import (
	"context"
	"sync"

	"shopMe/internal/worker"

	"go.uber.org/zap"
)

// StartWorkers starts all background concurrent workers centrally from the container.
// This guarantees that all background engines share the exact same singleton dependencies,
// eliminating duplicate instances and uninitialized subsystems.
func (c *Container) StartWorkers(ctx context.Context, wg *sync.WaitGroup) {
	c.Logger.Info("Starting centralized background worker pool...",
		zap.Int("total_workers", 3),
	)

	wg.Add(3)

	// 1. Reservation Cleaner: Frees unverified in-store holds after timeout
	go func() {
		defer wg.Done()
		worker.StartReservationCleaner(ctx, c.DB, c.Logger)
	}()

	// 2. Backup Worker: Database snapshot scheduler
	go func() {
		defer wg.Done()
		worker.StartBackupWorker(ctx, c.DB, c.Logger)
	}()

	// 3. Outbox Worker: Transactional outbox event publisher
	go func() {
		defer wg.Done()
		worker.StartOutboxWorker(ctx, c.DB, c.Logger)
	}()
}

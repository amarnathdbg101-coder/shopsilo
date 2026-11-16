// Package app provides centralized application state, dependency container, feature registry, and worker coordination.
package app

import (
	"context"
	"sync"
	"time"

	"shopMe/internal/worker"

	"go.uber.org/zap"
)

// StartWorkers starts all background concurrent workers centrally from the container.
// Each worker runs in its own goroutine with panic recovery and graceful shutdown coordination.
func (c *Container) StartWorkers(ctx context.Context, wg *sync.WaitGroup) {
	c.Logger.Info("Starting background worker engine...", zap.Int("total_workers", 3))

	runWorker := func(name string, fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				func() {
					defer func() {
						if r := recover(); r != nil {
							c.Logger.Error("background worker recovered from panic", zap.String("worker", name), zap.Any("panic", r))
							time.Sleep(2 * time.Second)
						}
					}()
					fn()
				}()

				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
		}()
	}

	// 1. Reservation Cleaner: Frees unverified in-store holds after timeout
	runWorker("reservation_cleaner", func() {
		worker.StartReservationCleaner(ctx, c.DB, c.Logger)
	})

	// 2. Backup Worker: Database snapshot scheduler
	runWorker("backup_worker", func() {
		worker.StartBackupWorker(ctx, c.DB, c.Logger)
	})

	// 3. Outbox Worker: Transactional outbox event publisher
	runWorker("outbox_worker", func() {
		worker.StartOutboxWorker(ctx, c.DB, c.Logger)
	})
}

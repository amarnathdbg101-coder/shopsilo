// Package app provides centralized application state, dependency container, feature registry, and worker coordination.
package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"shopMe/internal/worker"

	supervisor "github.com/amarnathdbg101-coder/go/go-supervisor"
	"go.uber.org/zap"
)

// StartWorkers starts all background concurrent workers centrally from the container
// under an Erlang/OTP-style fault-tolerant supervisor with crash-loop circuit breaker.
// If any worker experiences an unhandled panic or network glitch, the supervisor
// captures the stack trace and restarts the worker automatically with exponential backoff.
func (c *Container) StartWorkers(ctx context.Context, wg *sync.WaitGroup) {
	c.Logger.Info("Starting supervised background worker engine (go-supervisor)...",
		zap.Int("total_workers", 3),
		zap.String("strategy", "OneForOne"),
	)

	sup := supervisor.New(
		supervisor.WithStrategy(supervisor.StrategyOneForOne),
		supervisor.WithCrashLoopProtection(10, 5*time.Minute),
		supervisor.WithOnPanic(func(name string, panicVal any, stack []byte) {
			c.Logger.Error("CRITICAL: supervised background worker panicked, restarting...",
				zap.String("worker", name),
				zap.Any("panic", panicVal),
				zap.ByteString("stack", stack),
			)
		}),
	)

	// 1. Reservation Cleaner: Frees unverified in-store holds after timeout
	_ = sup.Supervise("reservation_cleaner", func(workerCtx context.Context) error {
		worker.StartReservationCleaner(workerCtx, c.DB, c.Logger)
		return nil
	})

	// 2. Backup Worker: Database snapshot scheduler
	_ = sup.Supervise("backup_worker", func(workerCtx context.Context) error {
		worker.StartBackupWorker(workerCtx, c.DB, c.Logger)
		return nil
	})

	// 3. Outbox Worker: Transactional outbox event publisher
	_ = sup.Supervise("outbox_worker", func(workerCtx context.Context) error {
		worker.StartOutboxWorker(workerCtx, c.DB, c.Logger)
		return nil
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sup.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			c.Logger.Warn("supervisor worker pool terminated with status", zap.Error(err))
		}
	}()
}

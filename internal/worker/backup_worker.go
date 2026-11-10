//Package worker
package worker

import (
	"context"
	"shopMe/internal/utils"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func StartBackupWorker(ctx context.Context, db *pgxpool.Pool, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("database backup worker recovered from panic", zap.Any("panic", r))
		}
	}()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("database backup worker stopped cleanly")
			return
		case <-ticker.C:
			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("backup worker caught panic during execution", zap.Any("panic", r))
					}
				}()
				cleanCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				defer cancel()

				// Multi-replica leader election via native PostgreSQL advisory lock (ID: 847291)
				var acquired bool
				if err := db.QueryRow(cleanCtx, "SELECT pg_try_advisory_lock()", 847291).Scan(&acquired); err != nil || !acquired {
					logger.Info("another cluster replica is running database backup; skipping to avoid duplicate dump")
					return
				}
				defer func() {
					_, _ = db.Exec(context.Background(), "SELECT pg_advisory_unlock()", 847291)
				}()

				meta, err := utils.GenerateFullDatabaseBackupDump(cleanCtx, db)
				if err != nil {
					logger.Error("nightly automated database backup failed", zap.Error(err))
				} else {
					logger.Info("nightly automated database backup completed successfully",
						zap.String("filename", meta.Filename),
						zap.Int64("size_bytes", meta.SizeBytes))
				}
			}()
		}
	}
}


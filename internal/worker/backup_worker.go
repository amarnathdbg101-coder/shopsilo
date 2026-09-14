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
			cleanCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			meta, err := utils.GenerateFullDatabaseBackupDump(cleanCtx, db)
			if err != nil {
				logger.Error("nightly automated database backup failed", zap.Error(err))
			} else {
				logger.Info("nightly automated database backup completed successfully",
					zap.String("filename", meta.Filename),
					zap.Int64("size_bytes", meta.SizeBytes))
			}
			cancel()
		}
	}
}

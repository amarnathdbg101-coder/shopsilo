package worker

import (
	"context"
	"shopMe/internal/handler/repository"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func StartReservationCleaner(ctx context.Context, db *pgxpool.Pool, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("reservation cleaner recovered from panic", zap.Any("panic", r))
		}
	}()

	// Run every 30 minutes instead of every 2 minutes so Neon compute can auto-suspend after 5 min of idle time
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	resRepo := repository.NewReservationRepo(db, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("reservation cleaner stopped cleanly")
			return
		case <-ticker.C:
			cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := resRepo.ExpireStaleReservations(cleanCtx); err != nil {
				logger.Warn("reservation cleaner encountered an issue", zap.Error(err))
			}
			cancel()
		}
	}
}

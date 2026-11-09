package worker

import (
	"context"
	"time"

	"shopMe/internal/handler/services"

	"go.uber.org/zap"
)

// StartDreamingWorker starts the background autonomous memory consolidation worker ("The Dreaming Engine").
// It periodically scans dialogue turns from active shops, distills permanent merchant rules and preferences,
// and saves high-confidence semantic memory nodes.
func StartDreamingWorker(
	ctx context.Context,
	partnerService *services.AIPartnerService,
	logger *zap.Logger,
) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("ai dreaming worker recovered from panic", zap.Any("panic", r))
		}
	}()

	logger.Info("ai dreaming worker initialized (Nightly Memory Consolidation Engine)")

	// Check periodically (e.g., every 1 hour, triggers nightly memory sweep)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Track last consolidation date to prevent multiple runs on the same calendar night
	lastRunDay := -1

	for {
		select {
		case <-ctx.Done():
			logger.Info("ai dreaming worker stopped cleanly")
			return

		case now := <-ticker.C:
			// Execute nightly memory consolidation between 2:00 AM and 4:00 AM
			currentDay := now.Day()
			currentHour := now.Hour()

			if currentHour >= 2 && currentHour < 5 && lastRunDay != currentDay {
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Error("ai dreaming worker caught panic during run", zap.Any("panic", r))
						}
					}()

					logger.Info("commencing nightly ai memory distillation pass...")
					sweepCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
					defer cancel()

					since := now.Add(-24 * time.Hour)
					nodesSaved, err := partnerService.ConsolidateAllActiveShops(sweepCtx, since)
					if err != nil {
						logger.Error("nightly ai memory consolidation encountered errors", zap.Error(err))
					} else {
						logger.Info("nightly ai memory consolidation completed successfully",
							zap.Int("new_semantic_nodes_stored", nodesSaved),
						)
						lastRunDay = currentDay
					}
				}()
			}
		}
	}
}

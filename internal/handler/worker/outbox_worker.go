package worker

import (
	"context"
	"encoding/json"
	"shopMe/internal/handler/services"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type OutboxEvent struct {
	ID        string `json:"id"`
	ShopID    string `json:"shop_id"`
	EventType string `json:"event_type"`
	Payload   []byte `json:"payload"`
}

// StartOutboxWorker runs an atomic Background Event Dispatcher (Transactional Outbox Pattern)
func StartOutboxWorker(ctx context.Context, db *pgxpool.Pool, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("outbox worker recovered from panic", zap.Any("panic", r))
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	wsHub := services.GetWebSocketHub(logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("outbox worker stopped cleanly")
			return
		case <-ticker.C:
			processOutboxEvents(ctx, db, wsHub, logger)
		}
	}
}

func processOutboxEvents(ctx context.Context, db *pgxpool.Pool, wsHub *services.WebSocketHub, logger *zap.Logger) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		SELECT id, shop_id, event_type, payload
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 100
	`

	rows, err := db.Query(queryCtx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	var events []*OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.ShopID, &e.EventType, &e.Payload); err == nil {
			events = append(events, &e)
		}
	}

	if len(events) == 0 {
		return
	}

	for _, e := range events {
		var parsedPayload interface{}
		_ = json.Unmarshal(e.Payload, &parsedPayload)

		// Broadcast real-time WebSocket alert to shopkeeper counter in 0ms!
		wsHub.BroadcastToShop(e.ShopID, e.EventType, parsedPayload)

		// Mark outbox event as processed atomically
		updateQuery := `UPDATE outbox_events SET status = 'processed', processed_at = NOW() WHERE id = $1`
		_, _ = db.Exec(queryCtx, updateQuery, e.ID)
	}
}

package worker

import (
	"context"
	"encoding/json"
	"shopMe/internal/handler/services"
	"sync"
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

var (
	outboxNotifyChan = make(chan struct{}, 100)
	outboxOnce       sync.Once
)

// TriggerOutboxDispatch notifies the background worker immediately when a new outbox event is inserted (0ms latency, 0 constant polling)
func TriggerOutboxDispatch() {
	select {
	case outboxNotifyChan <- struct{}{}:
	default:
		// Channel already has a pending signal, no need to block
	}
}

// StartOutboxWorker runs an event-driven Transactional Outbox Dispatcher.
// It eliminates continuous 1-second DB polling to preserve serverless compute hours (Neon Scale-to-Zero).
func StartOutboxWorker(ctx context.Context, db *pgxpool.Pool, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("outbox worker recovered from panic", zap.Any("panic", r))
		}
	}()

	wsHub := services.GetWebSocketHub(logger)

	// Infrequent fallback heartbeat ticker (30 minutes) instead of 1 second to allow DB sleep
	heartbeatTicker := time.NewTicker(30 * time.Minute)
	defer heartbeatTicker.Stop()

	// Process any leftover startup events once on launch
	processOutboxEvents(ctx, db, wsHub)

	for {
		select {
		case <-ctx.Done():
			logger.Info("outbox worker stopped cleanly")
			return
		case <-outboxNotifyChan:
			// Event-triggered execution: runs only when an actual event is enqueued
			processOutboxEvents(ctx, db, wsHub)
		case <-heartbeatTicker.C:
			// Passive long-interval sweep
			processOutboxEvents(ctx, db, wsHub)
		}
	}
}

func processOutboxEvents(ctx context.Context, db *pgxpool.Pool, wsHub *services.WebSocketHub) {
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

package worker

import (
	"context"
	"encoding/json"
	"time"

	gooutbox "github.com/amarnathdbg101-coder/go/go-outbox"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"shopMe/internal/handler/services"
)

// StartOutboxWorker runs an atomic Background Event Dispatcher using go-outbox.
// It leverages PostgreSQL FOR UPDATE SKIP LOCKED concurrency, exponential backoff,
// and dead-letter queue handling without duplicating events across cluster replicas.
func StartOutboxWorker(ctx context.Context, db *pgxpool.Pool, logger *zap.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("outbox worker recovered from panic", zap.Any("panic", r))
		}
	}()

	sqlDB := stdlib.OpenDBFromPool(db)
	store := gooutbox.NewSQLStore(sqlDB, &gooutbox.SQLOptions{
		TableName:         "outbox_events",
		TopicColumn:       "event_type",
		AggregateIDColumn: "shop_id",
	})

	wsHub := services.GetWebSocketHub(logger)

	// Publisher forwards committed events to WebSocket hub in real-time
	publisher := gooutbox.FuncPublisher(func(ctx context.Context, evt *gooutbox.Event) error {
		var parsedPayload any
		if err := json.Unmarshal(evt.Payload, &parsedPayload); err != nil {
			parsedPayload = string(evt.Payload)
		}

		// Broadcast real-time WebSocket alert to shopkeeper counter in 0ms!
		wsHub.BroadcastToShop(evt.AggregateID, evt.Topic, parsedPayload)
		return nil
	})

	engine, err := gooutbox.New(store, publisher,
		gooutbox.WithPollInterval(1*time.Second),
		gooutbox.WithBatchSize(100),
		gooutbox.WithMaxRetries(5),
		gooutbox.WithCallbacks(nil, func(evt *gooutbox.Event, err error) {
			logger.Warn("outbox event dispatch failed, scheduled for retry",
				zap.String("id", evt.ID),
				zap.String("topic", evt.Topic),
				zap.Int("retry_count", evt.RetryCount),
				zap.Error(err),
			)
		}),
	)
	if err != nil {
		logger.Error("failed to initialize outbox engine", zap.Error(err))
		return
	}

	logger.Info("resilient transactional outbox worker started successfully (go-outbox)")
	engine.Start(ctx)
}

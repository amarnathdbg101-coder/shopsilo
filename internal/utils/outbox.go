package utils

import (
	"context"

	gooutbox "github.com/amarnathdbg101-coder/go/go-outbox"
	"github.com/jackc/pgx/v5"
)

// defaultOutboxStore provides standardized outbox persistence mapped to outbox_events table.
var defaultOutboxStore = gooutbox.NewSQLStore(nil, &gooutbox.SQLOptions{
	TableName:         "outbox_events",
	TopicColumn:       "event_type",
	AggregateIDColumn: "shop_id",
})

// EnqueueOutboxEventTx writes an event to outbox_events table within a database transaction (Transactional Outbox Pattern).
// Powered by go-outbox for consistent schema validation, zero-dual-write atomicity, and retry scheduling.
func EnqueueOutboxEventTx(ctx context.Context, tx pgx.Tx, shopID string, eventType string, payload interface{}) error {
	event, err := gooutbox.NewEvent(eventType, shopID, payload)
	if err != nil {
		return err
	}
	return defaultOutboxStore.Save(ctx, tx, event)
}

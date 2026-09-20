package utils

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// EnqueueOutboxEventTx writes an event to outbox_events table within a database transaction (Transactional Outbox Pattern)
func EnqueueOutboxEventTx(ctx context.Context, tx pgx.Tx, shopID string, eventType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox event payload: %w", err)
	}

	query := `
		INSERT INTO outbox_events (shop_id, event_type, payload, status, created_at)
		VALUES ($1, $2, $3, 'pending', NOW())
	`
	_, err = tx.Exec(ctx, query, shopID, eventType, payloadBytes)
	return err
}

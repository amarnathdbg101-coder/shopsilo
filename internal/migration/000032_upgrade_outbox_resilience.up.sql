-- 000032_upgrade_outbox_resilience.up.sql
-- Adds retry telemetry, exponential backoff scheduling, and dead-letter queue columns to outbox_events.

ALTER TABLE outbox_events 
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_outbox_retry_poll 
ON outbox_events(status, next_retry_at ASC) 
WHERE status IN ('pending', 'failed');

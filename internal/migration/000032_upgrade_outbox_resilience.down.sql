-- 000032_upgrade_outbox_resilience.down.sql
DROP INDEX IF EXISTS idx_outbox_retry_poll;

ALTER TABLE outbox_events 
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS max_retries,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS next_retry_at;

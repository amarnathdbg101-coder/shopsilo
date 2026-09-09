-- 000020_content_moderation_and_ban_evasion.down.sql

DROP TABLE IF EXISTS content_reports;
DROP TABLE IF EXISTS banned_entities;

ALTER TABLE shops
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS flagged_count,
    DROP COLUMN IF EXISTS suspension_reason,
    DROP COLUMN IF EXISTS creation_ip,
    DROP COLUMN IF EXISTS creation_user_agent,
    DROP COLUMN IF EXISTS device_fingerprint;

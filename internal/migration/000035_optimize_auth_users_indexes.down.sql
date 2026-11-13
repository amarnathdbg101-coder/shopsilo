-- Rollback Phase 1 & Phase 2 Auth Optimization Indexes
DROP INDEX IF EXISTS idx_users_lower_email;
DROP INDEX IF EXISTS idx_users_phone;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;

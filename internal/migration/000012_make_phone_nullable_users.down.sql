-- 000012_make_phone_nullable_users.down.sql
DROP INDEX IF EXISTS idx_users_phone_unique;
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

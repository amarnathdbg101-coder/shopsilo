-- 000012_make_phone_nullable_users.up.sql
ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;
UPDATE users SET phone = NULL WHERE phone = '';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_key;
DROP INDEX IF EXISTS idx_users_phone;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users(phone) WHERE phone IS NOT NULL AND phone != '';

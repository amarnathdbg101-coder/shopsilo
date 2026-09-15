-- 000039_create_phone_verifications_table.down.sql

DROP INDEX IF EXISTS idx_users_phone_unique;
DROP TABLE IF EXISTS phone_verifications;

-- 000010_create_system_and_outbox_tables.down.sql
DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS admin_system_errors CASCADE;
DROP TABLE IF EXISTS banned_entities CASCADE;

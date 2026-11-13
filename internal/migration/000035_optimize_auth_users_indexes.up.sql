-- Phase 1 & Phase 2 Auth Optimization Indexes & Schema Enhancements

-- 1. Functional B-Tree Index for case-insensitive email lookup
CREATE INDEX IF NOT EXISTS idx_users_lower_email ON users (LOWER(TRIM(email)));

-- 2. Filtered B-Tree Index for phone number lookup
CREATE INDEX IF NOT EXISTS idx_users_phone ON users (phone) WHERE phone IS NOT NULL AND phone != '';

-- 3. Add token_version column to users for instant 0ms multi-device session revocation
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 1;

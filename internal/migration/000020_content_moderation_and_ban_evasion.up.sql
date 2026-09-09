-- 000020_content_moderation_and_ban_evasion.up.sql
-- Anti-Abuse, Content Moderation, and Ban Evasion Protection

-- 1. Enhance shops table with moderation status and tracking columns
ALTER TABLE shops
    ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS flagged_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS suspension_reason TEXT,
    ADD COLUMN IF NOT EXISTS creation_ip VARCHAR(50),
    ADD COLUMN IF NOT EXISTS creation_user_agent VARCHAR(255),
    ADD COLUMN IF NOT EXISTS device_fingerprint VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_shops_status ON shops(status);

-- 2. Banned Entities (IPs, Devices, Phones, Image Perceptual Hashes)
CREATE TABLE IF NOT EXISTS banned_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL, -- 'ip', 'device_id', 'phone', 'image_hash'
    entity_value VARCHAR(255) NOT NULL UNIQUE,
    reason TEXT NOT NULL,
    banned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_banned_entities_type_val ON banned_entities(entity_type, entity_value);

-- 3. Content Reports & Grievance Redressal (IT Rules 2021 compliance)
CREATE TABLE IF NOT EXISTS content_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_type VARCHAR(30) NOT NULL, -- 'shop', 'product', 'review'
    target_id UUID NOT NULL,
    reason VARCHAR(50) NOT NULL, -- 'sexual_content', 'counterfeit_fake', 'scam_fraud', 'harassment', 'illegal_substance', 'other'
    details TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending', -- 'pending', 'reviewed', 'action_taken', 'dismissed'
    reporter_ip VARCHAR(50),
    reporter_user_agent VARCHAR(255),
    action_taken TEXT,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_content_reports_target ON content_reports(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_content_reports_status ON content_reports(status);
CREATE INDEX IF NOT EXISTS idx_content_reports_created ON content_reports(created_at);

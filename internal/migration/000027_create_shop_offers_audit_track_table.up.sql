CREATE TABLE IF NOT EXISTS shop_offers_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offer_id UUID NOT NULL,
    shop_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL,
    previous_title TEXT,
    previous_discount_text TEXT,
    previous_description TEXT,
    new_title TEXT,
    new_discount_text TEXT,
    new_description TEXT,
    changed_by_user_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_offers_audit_offer_id ON shop_offers_audit(offer_id);
CREATE INDEX IF NOT EXISTS idx_offers_audit_shop_id ON shop_offers_audit(shop_id);

-- 000029_cash_drawer_and_shift_telemetry.up.sql
-- Milestone 2: Cash Drawer Telemetry, Shift Accountability, Mid-day Safe Drops & Petty Cash-out Tagging

-- 1. Cash Drawer Shift Sessions
CREATE TABLE IF NOT EXISTS cash_drawer_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    staff_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    shift_name VARCHAR(50) NOT NULL DEFAULT 'General Shift',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    opening_cash DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (opening_cash >= 0),
    cash_sales_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    safe_drop_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    expenses_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    expected_cash DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    actual_cash DECIMAL(10, 2),
    discrepancy_amount DECIMAL(10, 2) DEFAULT 0.00,
    safe_drop_threshold DECIMAL(10, 2) NOT NULL DEFAULT 10000.00,
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'DISCREPANCY')),
    closing_notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cash_drawer_sessions_shop_staff 
ON cash_drawer_sessions(shop_id, staff_user_id, status);

CREATE INDEX IF NOT EXISTS idx_cash_drawer_sessions_opened 
ON cash_drawer_sessions(shop_id, opened_at DESC);

-- 2. Cash Drawer Events (Ledger of all cash moves in the physical drawer)
CREATE TABLE IF NOT EXISTS cash_drawer_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES cash_drawer_sessions(id) ON DELETE CASCADE,
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    staff_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('OPENING_FLOAT', 'CASH_SALE', 'SAFE_DROP', 'PETTY_EXPENSE', 'MANUAL_OPEN', 'BILL_VOID', 'SHIFT_CLOSE')),
    amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    bill_id UUID REFERENCES pos_bills(id) ON DELETE SET NULL,
    category VARCHAR(100) NOT NULL DEFAULT 'General',
    reason VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cash_drawer_events_session 
ON cash_drawer_events(session_id, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_cash_drawer_events_shop_type 
ON cash_drawer_events(shop_id, event_type, created_at DESC);


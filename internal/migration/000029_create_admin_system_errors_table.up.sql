CREATE TABLE IF NOT EXISTS admin_system_errors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    error_code VARCHAR(100) NOT NULL,
    user_friendly_msg TEXT NOT NULL,
    developer_stack_trace TEXT NOT NULL,
    request_path VARCHAR(255),
    user_id VARCHAR(100),
    user_role VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_admin_sys_errors_expires ON admin_system_errors(expires_at);
CREATE INDEX IF NOT EXISTS idx_admin_sys_errors_created ON admin_system_errors(created_at DESC);

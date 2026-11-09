-- 000033_create_ai_brain_tables.up.sql
-- Autonomous AI Business Partner (Conversations, Messages, Semantic Memory Graph, Action Audits, and Proactive Briefings)

-- 1. AI Conversations Session Table
CREATE TABLE IF NOT EXISTS ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT 'Business Strategy & Operations',
    active_persona VARCHAR(50) NOT NULL DEFAULT 'COO', -- 'COO', 'CA', 'ANALYST', 'STAFF'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_shop ON ai_conversations(shop_id, updated_at DESC);

-- 2. AI Messages Table (Itemized dialogue turns with tool calls and token tracking)
CREATE TABLE IF NOT EXISTS ai_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    tool_calls JSONB,
    tool_results JSONB,
    token_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_messages_convo ON ai_messages(conversation_id, created_at ASC);

-- 3. AI Semantic Memory Graph (Distilled Long-Term Memory Nodes)
CREATE TABLE IF NOT EXISTS ai_memory_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL, -- 'MERCHANT_RULE', 'CUSTOMER_HABIT', 'SUPPLIER_TERM', 'OPERATIONAL_PREFERENCE'
    subject VARCHAR(128) NOT NULL,
    predicate VARCHAR(64) NOT NULL,
    object_value TEXT NOT NULL,
    confidence_score DECIMAL(3, 2) NOT NULL DEFAULT 1.00,
    access_count INT NOT NULL DEFAULT 1,
    last_recalled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_memory_shop_cat ON ai_memory_nodes(shop_id, category);
CREATE INDEX IF NOT EXISTS idx_ai_memory_recalled ON ai_memory_nodes(shop_id, last_recalled_at DESC);

-- 4. AI Action Execution Audit Logs (Deterministic Traceability & Rollback)
CREATE TABLE IF NOT EXISTS ai_action_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    tool_name VARCHAR(100) NOT NULL,
    input_payload JSONB NOT NULL,
    output_result JSONB,
    status VARCHAR(30) NOT NULL DEFAULT 'EXECUTED', -- 'PENDING_APPROVAL', 'EXECUTED', 'CANCELLED', 'ROLLED_BACK'
    rollback_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_action_logs_shop ON ai_action_logs(shop_id, created_at DESC);

-- 5. AI Proactive Briefings & Anomaly Alerts (Morning Kickoff, Evening Settle, Theft Radar)
CREATE TABLE IF NOT EXISTS ai_proactive_briefings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
    briefing_type VARCHAR(50) NOT NULL, -- 'MORNING_KICKOFF', 'EVENING_SETTLE', 'ANOMALY_THEFT_ALERT', 'DEAD_STOCK_WARNING'
    title VARCHAR(255) NOT NULL,
    summary_markdown TEXT NOT NULL,
    action_items JSONB,
    is_dismissed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_briefings_unread ON ai_proactive_briefings(shop_id, is_dismissed, created_at DESC);

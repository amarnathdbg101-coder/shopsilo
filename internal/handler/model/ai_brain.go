// Package model provides database entity definitions for ShopSilo.
package model

import (
	"encoding/json"
	"time"
)

// AI Persona roles
const (
	PersonaCOO     = "COO"     // Business Growth Strategist & Chief Operating Officer
	PersonaCA      = "CA"      // Chartered Accountant, Cash Reconciliation & Tax Auditor
	PersonaAnalyst = "ANALYST" // Senior Data Analyst & Basket Association Specialist
	PersonaStaff   = "STAFF"   // POS Counter Operator & Staff Automation
)

// AI Semantic Memory categories
const (
	MemoryCatMerchantRule           = "MERCHANT_RULE"
	MemoryCatCustomerHabit          = "CUSTOMER_HABIT"
	MemoryCatSupplierTerm           = "SUPPLIER_TERM"
	MemoryCatOperationalPreference = "OPERATIONAL_PREFERENCE"
)

// AI Action execution statuses
const (
	ActionStatusPendingApproval = "PENDING_APPROVAL"
	ActionStatusExecuted        = "EXECUTED"
	ActionStatusCancelled       = "CANCELLED"
	ActionStatusRolledBack      = "ROLLED_BACK"
)

// AI Proactive Briefing types
const (
	BriefingMorningKickoff   = "MORNING_KICKOFF"
	BriefingEveningSettle    = "EVENING_SETTLE"
	BriefingAnomalyTheft     = "ANOMALY_THEFT_ALERT"
	BriefingDeadStockWarning = "DEAD_STOCK_WARNING"
)

// AIConversation represents an ongoing strategic dialog session.
type AIConversation struct {
	ID            string    `json:"id"`
	ShopID        string    `json:"shop_id"`
	UserID        string    `json:"user_id"`
	Title         string    `json:"title"`
	ActivePersona string    `json:"active_persona"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AIMessage represents a single turn in a conversation.
type AIMessage struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	Role           string          `json:"role"` // 'user', 'assistant', 'system', 'tool'
	Content        string          `json:"content"`
	ToolCalls      json.RawMessage `json:"tool_calls,omitempty"`
	ToolResults    json.RawMessage `json:"tool_results,omitempty"`
	TokenCount     int             `json:"token_count"`
	CreatedAt      time.Time       `json:"created_at"`
}

// AIMemoryNode represents an atomic distilled piece of business knowledge.
type AIMemoryNode struct {
	ID              string     `json:"id"`
	ShopID          string     `json:"shop_id"`
	Category        string     `json:"category"`
	Subject         string     `json:"subject"`
	Predicate       string     `json:"predicate"`
	ObjectValue     string     `json:"object_value"`
	ConfidenceScore float64    `json:"confidence_score"`
	AccessCount     int        `json:"access_count"`
	LastRecalledAt  time.Time  `json:"last_recalled_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// AIActionLog records every executed or challenged tool call for full traceability.
type AIActionLog struct {
	ID              string          `json:"id"`
	ShopID          string          `json:"shop_id"`
	UserID          string          `json:"user_id"`
	ToolName        string          `json:"tool_name"`
	InputPayload    json.RawMessage `json:"input_payload"`
	OutputResult    json.RawMessage `json:"output_result,omitempty"`
	Status          string          `json:"status"`
	RollbackPayload json.RawMessage `json:"rollback_payload,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// AIProactiveBriefing represents scheduled or anomaly-driven push insights.
type AIProactiveBriefing struct {
	ID              string          `json:"id"`
	ShopID          string          `json:"shop_id"`
	BriefingType    string          `json:"briefing_type"`
	Title           string          `json:"title"`
	SummaryMarkdown string          `json:"summary_markdown"`
	ActionItems     json.RawMessage `json:"action_items,omitempty"`
	IsDismissed     bool            `json:"is_dismissed"`
	CreatedAt       time.Time       `json:"created_at"`
}

package dto

import "time"

// AIPartnerChatRequest defines the incoming payload for chatting with the AI partner.
type AIPartnerChatRequest struct {
	Message string `json:"message" validate:"required,min=1"`
	Persona string `json:"persona,omitempty" validate:"omitempty,oneof=COO CA ANALYST STAFF"`
}

// AIActionConfirmationCard represents a high-impact action requiring merchant confirmation before execution.
type AIActionConfirmationCard struct {
	ActionID     string         `json:"action_id"`
	ToolName     string         `json:"tool_name"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	DiffPreview  map[string]any `json:"diff_preview"`
	IsReversible bool           `json:"is_reversible"`
}

// AIPartnerChatResponse returns the assistant's reply and any actionable follow-ups.
type AIPartnerChatResponse struct {
	ConversationID             string                    `json:"conversation_id"`
	ActivePersona              string                    `json:"active_persona"`
	Reply                      string                    `json:"reply"`
	SuggestedActions           []string                  `json:"suggested_actions,omitempty"`
	ActionPendingConfirmation *AIActionConfirmationCard `json:"action_pending_confirmation,omitempty"`
	CreatedAt                  time.Time                 `json:"created_at"`
}

// AIConfirmActionRequest defines user confirmation on an interactive action card.
type AIConfirmActionRequest struct {
	ActionID string `json:"action_id" validate:"required,uuid"`
	Approved bool   `json:"approved"`
}

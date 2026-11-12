package dto

import "time"

type MerchantAssistantChatRequest struct {
	ConversationID string `json:"conversation_id,omitempty"`
	Prompt         string `json:"prompt"`
}

type MerchantAssistantChatResponse struct {
	ConversationID string   `json:"conversation_id"`
	Text           string   `json:"text"`
	ToolsUsed      []string `json:"tools_used"`
}

type MerchantAssistantConversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MerchantAssistantMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type MerchantAssistantConversationsResponse struct {
	Conversations []MerchantAssistantConversation `json:"conversations"`
}

type MerchantAssistantMessagesResponse struct {
	Messages []MerchantAssistantMessage `json:"messages"`
}

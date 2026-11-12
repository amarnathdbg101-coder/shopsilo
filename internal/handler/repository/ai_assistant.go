package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"shopMe/internal/handler/dto"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var ErrAIAssistantConversationNotFound = errors.New("AI assistant conversation not found")

type AIAssistantRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewAIAssistantRepo(db *pgxpool.Pool, logger *zap.Logger) *AIAssistantRepo {
	return &AIAssistantRepo{db: db, logger: logger}
}

func (r *AIAssistantRepo) CreateConversation(ctx context.Context, shopID, userID, title string) (*dto.MerchantAssistantConversation, error) {
	const query = `
		INSERT INTO ai_conversations (shop_id, user_id, title, active_persona)
		VALUES ($1, $2, $3, 'COO')
		RETURNING id, title, updated_at
	`
	conversation := &dto.MerchantAssistantConversation{}
	if err := r.db.QueryRow(ctx, query, shopID, userID, title).Scan(
		&conversation.ID,
		&conversation.Title,
		&conversation.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("create AI assistant conversation: %w", err)
	}
	return conversation, nil
}

func (r *AIAssistantRepo) GetConversation(ctx context.Context, id, shopID, userID string) (*dto.MerchantAssistantConversation, error) {
	const query = `
		SELECT id, title, updated_at
		FROM ai_conversations
		WHERE id = $1 AND shop_id = $2 AND user_id = $3
	`
	conversation := &dto.MerchantAssistantConversation{}
	if err := r.db.QueryRow(ctx, query, id, shopID, userID).Scan(
		&conversation.ID,
		&conversation.Title,
		&conversation.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAIAssistantConversationNotFound
		}
		return nil, fmt.Errorf("get AI assistant conversation: %w", err)
	}
	return conversation, nil
}

func (r *AIAssistantRepo) ListConversations(ctx context.Context, shopID, userID string, limit int) ([]dto.MerchantAssistantConversation, error) {
	const query = `
		SELECT id, title, updated_at
		FROM ai_conversations
		WHERE shop_id = $1 AND user_id = $2
		ORDER BY updated_at DESC
		LIMIT $3
	`
	rows, err := r.db.Query(ctx, query, shopID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list AI assistant conversations: %w", err)
	}
	defer rows.Close()

	conversations := make([]dto.MerchantAssistantConversation, 0)
	for rows.Next() {
		var conversation dto.MerchantAssistantConversation
		if err := rows.Scan(&conversation.ID, &conversation.Title, &conversation.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan AI assistant conversation: %w", err)
		}
		conversations = append(conversations, conversation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate AI assistant conversations: %w", err)
	}
	return conversations, nil
}

func (r *AIAssistantRepo) ListMessages(ctx context.Context, conversationID string, limit int) ([]dto.MerchantAssistantMessage, error) {
	const query = `
		SELECT id, role, content, created_at
		FROM ai_messages
		WHERE conversation_id = $1 AND role IN ('user', 'assistant')
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list AI assistant messages: %w", err)
	}
	defer rows.Close()

	messages := make([]dto.MerchantAssistantMessage, 0)
	for rows.Next() {
		var message dto.MerchantAssistantMessage
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan AI assistant message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate AI assistant messages: %w", err)
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func (r *AIAssistantRepo) AddMessage(ctx context.Context, conversationID, role, content string, toolCalls, toolResults json.RawMessage) error {
	const query = `
		WITH inserted_message AS (
			INSERT INTO ai_messages (conversation_id, role, content, tool_calls, tool_results)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING conversation_id
		)
		UPDATE ai_conversations
		SET updated_at = NOW()
		WHERE id = (SELECT conversation_id FROM inserted_message)
	`
	if _, err := r.db.Exec(ctx, query, conversationID, role, content, nullableJSON(toolCalls), nullableJSON(toolResults)); err != nil {
		return fmt.Errorf("save AI assistant message: %w", err)
	}
	return nil
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return []byte(value)
}

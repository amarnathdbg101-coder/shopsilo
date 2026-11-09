package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AIBrainRepo handles persistence for AI conversations, semantic memory, audits, and briefings.
type AIBrainRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewAIBrainRepo creates an initialized AIBrainRepo.
func NewAIBrainRepo(db *pgxpool.Pool, logger *zap.Logger) *AIBrainRepo {
	return &AIBrainRepo{
		db:     db,
		logger: logger,
	}
}

// GetOrCreateConversation retrieves the latest conversation for the shop and user, or initializes a new session.
func (r *AIBrainRepo) GetOrCreateConversation(ctx context.Context, shopID, userID, persona string) (*model.AIConversation, error) {
	if persona == "" {
		persona = model.PersonaCOO
	}

	query := `
		SELECT id, shop_id, user_id, title, active_persona, created_at, updated_at
		FROM ai_conversations
		WHERE shop_id = $1 AND user_id = $2
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var c model.AIConversation
	err := r.db.QueryRow(ctx, query, shopID, userID).Scan(
		&c.ID, &c.ShopID, &c.UserID, &c.Title, &c.ActivePersona, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == nil {
		// Update active persona if changed
		if c.ActivePersona != persona {
			_, _ = r.db.Exec(ctx, "UPDATE ai_conversations SET active_persona = $1, updated_at = NOW() WHERE id = $2", persona, c.ID)
			c.ActivePersona = persona
		}
		return &c, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to query conversation: %w", err)
	}

	// Create new session
	insertQuery := `
		INSERT INTO ai_conversations (shop_id, user_id, title, active_persona, created_at, updated_at)
		VALUES ($1, $2, 'Daily Business Operations', $3, NOW(), NOW())
		RETURNING id, shop_id, user_id, title, active_persona, created_at, updated_at
	`
	err = r.db.QueryRow(ctx, insertQuery, shopID, userID, persona).Scan(
		&c.ID, &c.ShopID, &c.UserID, &c.Title, &c.ActivePersona, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return &c, nil
}

// AddMessage appends an itemized message turn to a conversation and bumps session updated_at.
func (r *AIBrainRepo) AddMessage(ctx context.Context, msg *model.AIMessage) error {
	query := `
		INSERT INTO ai_messages (conversation_id, role, content, tool_calls, tool_results, token_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query,
		msg.ConversationID, msg.Role, msg.Content, msg.ToolCalls, msg.ToolResults, msg.TokenCount,
	).Scan(&msg.ID, &msg.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert ai_message: %w", err)
	}

	// Bump conversation timestamp
	_, _ = r.db.Exec(ctx, "UPDATE ai_conversations SET updated_at = NOW() WHERE id = $1", msg.ConversationID)
	return nil
}

// GetRecentMessages retrieves the last N messages of a conversation in chronological order for sliding-window context.
func (r *AIBrainRepo) GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]*model.AIMessage, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, conversation_id, role, content, tool_calls, tool_results, token_count, created_at
		FROM (
			SELECT id, conversation_id, role, content, tool_calls, tool_results, token_count, created_at
			FROM ai_messages
			WHERE conversation_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		) sub
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent ai_messages: %w", err)
	}
	defer rows.Close()

	var messages []*model.AIMessage
	for rows.Next() {
		var m model.AIMessage
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.Role, &m.Content,
			&m.ToolCalls, &m.ToolResults, &m.TokenCount, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}

	return messages, nil
}

// SaveMemoryNode creates or updates a distilled semantic memory node with confidence reinforcement.
func (r *AIBrainRepo) SaveMemoryNode(ctx context.Context, node *model.AIMemoryNode) error {
	query := `
		INSERT INTO ai_memory_nodes (shop_id, category, subject, predicate, object_value, confidence_score, access_count, last_recalled_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		node.ShopID, node.Category, node.Subject, node.Predicate, node.ObjectValue,
		node.ConfidenceScore, node.AccessCount, node.ExpiresAt,
	).Scan(&node.ID, &node.CreatedAt)
}

// GetTopMemoryNodes fetches the most active and relevant long-term memory nodes for a shop.
func (r *AIBrainRepo) GetTopMemoryNodes(ctx context.Context, shopID string, limit int) ([]*model.AIMemoryNode, error) {
	if limit <= 0 {
		limit = 15
	}

	query := `
		SELECT id, shop_id, category, subject, predicate, object_value, confidence_score, access_count, last_recalled_at, expires_at, created_at
		FROM ai_memory_nodes
		WHERE shop_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY confidence_score DESC, last_recalled_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, shopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memory nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*model.AIMemoryNode
	for rows.Next() {
		var n model.AIMemoryNode
		if err := rows.Scan(
			&n.ID, &n.ShopID, &n.Category, &n.Subject, &n.Predicate, &n.ObjectValue,
			&n.ConfidenceScore, &n.AccessCount, &n.LastRecalledAt, &n.ExpiresAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		nodes = append(nodes, &n)
	}

	return nodes, nil
}

// RecordActionLog writes a deterministic action audit entry for AI tool executions.
func (r *AIBrainRepo) RecordActionLog(ctx context.Context, act *model.AIActionLog) error {
	query := `
		INSERT INTO ai_action_logs (shop_id, user_id, tool_name, input_payload, output_result, status, rollback_payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		act.ShopID, act.UserID, act.ToolName, act.InputPayload, act.OutputResult, act.Status, act.RollbackPayload,
	).Scan(&act.ID, &act.CreatedAt)
}

// CreateBriefing creates a proactive morning kickoff, evening settle, or anomaly warning.
func (r *AIBrainRepo) CreateBriefing(ctx context.Context, b *model.AIProactiveBriefing) error {
	query := `
		INSERT INTO ai_proactive_briefings (shop_id, briefing_type, title, summary_markdown, action_items, is_dismissed, created_at)
		VALUES ($1, $2, $3, $4, $5, false, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		b.ShopID, b.BriefingType, b.Title, b.SummaryMarkdown, b.ActionItems,
	).Scan(&b.ID, &b.CreatedAt)
}

// GetActiveBriefings returns all undismissed briefings for a shopkeeper.
func (r *AIBrainRepo) GetActiveBriefings(ctx context.Context, shopID string) ([]*model.AIProactiveBriefing, error) {
	query := `
		SELECT id, shop_id, briefing_type, title, summary_markdown, action_items, is_dismissed, created_at
		FROM ai_proactive_briefings
		WHERE shop_id = $1 AND is_dismissed = false
		ORDER BY created_at DESC
		LIMIT 5
	`

	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active briefings: %w", err)
	}
	defer rows.Close()

	var briefings []*model.AIProactiveBriefing
	for rows.Next() {
		var b model.AIProactiveBriefing
		if err := rows.Scan(
			&b.ID, &b.ShopID, &b.BriefingType, &b.Title, &b.SummaryMarkdown,
			&b.ActionItems, &b.IsDismissed, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		briefings = append(briefings, &b)
	}

	return briefings, nil
}

// DismissBriefing marks a proactive alert card as acknowledged/dismissed.
func (r *AIBrainRepo) DismissBriefing(ctx context.Context, shopID, briefingID string) error {
	query := `UPDATE ai_proactive_briefings SET is_dismissed = true WHERE id = $1 AND shop_id = $2`
	_, err := r.db.Exec(ctx, query, briefingID, shopID)
	return err
}

// GetActiveShopIDsWithRecentMessages returns shop IDs with message activity since the given timestamp.
func (r *AIBrainRepo) GetActiveShopIDsWithRecentMessages(ctx context.Context, since time.Time) ([]string, error) {
	query := `
		SELECT DISTINCT c.shop_id 
		FROM ai_conversations c
		JOIN ai_messages m ON m.conversation_id = c.id
		WHERE m.created_at >= $1
	`
	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get active shops with messages: %w", err)
	}
	defer rows.Close()

	var shopIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		shopIDs = append(shopIDs, id)
	}
	return shopIDs, nil
}

// GetDailyDialogueDigest compiles recent message text for a shop into a single consolidated string.
func (r *AIBrainRepo) GetDailyDialogueDigest(ctx context.Context, shopID string, since time.Time) (string, error) {
	query := `
		SELECT m.role, m.content
		FROM ai_messages m
		JOIN ai_conversations c ON m.conversation_id = c.id
		WHERE c.shop_id = $1 AND m.created_at >= $2
		ORDER BY m.created_at ASC
		LIMIT 100
	`
	rows, err := r.db.Query(ctx, query, shopID, since)
	if err != nil {
		return "", fmt.Errorf("failed to fetch daily dialogue: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return "", err
		}
		sb.WriteString(strings.ToUpper(role))
		sb.WriteString(": ")
		sb.WriteString(content)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// GetActionLog fetches a specific action audit log by ID and shop.
func (r *AIBrainRepo) GetActionLog(ctx context.Context, shopID, actionID string) (*model.AIActionLog, error) {
	query := `
		SELECT id, shop_id, user_id, tool_name, input_payload, output_result, status, rollback_payload, created_at
		FROM ai_action_logs
		WHERE id = $1 AND shop_id = $2
	`
	var a model.AIActionLog
	err := r.db.QueryRow(ctx, query, actionID, shopID).Scan(
		&a.ID, &a.ShopID, &a.UserID, &a.ToolName,
		&a.InputPayload, &a.OutputResult, &a.Status,
		&a.RollbackPayload, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateActionLogStatus transitions the action log status and writes output/rollback payloads.
func (r *AIBrainRepo) UpdateActionLogStatus(ctx context.Context, actionID, status string, outputResult, rollbackPayload any) error {
	var outBytes, rollBytes []byte
	if outputResult != nil {
		outBytes, _ = json.Marshal(outputResult)
	}
	if rollbackPayload != nil {
		rollBytes, _ = json.Marshal(rollbackPayload)
	}

	query := `
		UPDATE ai_action_logs
		SET status = $1,
		    output_result = COALESCE($2, output_result),
		    rollback_payload = COALESCE($3, rollback_payload)
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, status, outBytes, rollBytes, actionID)
	return err
}

// ListActionLogs retrieves the recent action logs for audit and rollback visualization.
func (r *AIBrainRepo) ListActionLogs(ctx context.Context, shopID string, limit int) ([]*model.AIActionLog, error) {
	if limit <= 0 {
		limit = 20
	}
	query := `
		SELECT id, shop_id, user_id, tool_name, input_payload, output_result, status, rollback_payload, created_at
		FROM ai_action_logs
		WHERE shop_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, shopID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list action logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.AIActionLog
	for rows.Next() {
		var a model.AIActionLog
		if err := rows.Scan(
			&a.ID, &a.ShopID, &a.UserID, &a.ToolName,
			&a.InputPayload, &a.OutputResult, &a.Status,
			&a.RollbackPayload, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &a)
	}
	return logs, nil
}

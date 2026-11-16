package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PostgresStore provides production-grade persistence for audit logs backed by PostgreSQL JSONB.
type PostgresStore struct {
	db        *sql.DB
	tableName string
}

// PostgresOptions configures PostgreSQL store settings.
type PostgresOptions struct {
	TableName string // Default: "audit_logs"
}

// NewPostgresStore creates a new PostgreSQL store backed by standard *sql.DB.
func NewPostgresStore(db *sql.DB, opts *PostgresOptions) *PostgresStore {
	tableName := "audit_logs"
	if opts != nil && opts.TableName != "" {
		tableName = opts.TableName
	}
	return &PostgresStore{
		db:        db,
		tableName: tableName,
	}
}

// SaveBatch executes an atomic bulk INSERT for the batch of audit entries.
// Ek single SQL statement me multi-row insert karke database network roundtrips ko 90% kam karta hai.
func (s *PostgresStore) SaveBatch(ctx context.Context, entries []*Entry) error {
	if len(entries) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(entries))
	valueArgs := make([]any, 0, len(entries)*11)

	argIndex := 1
	for _, e := range entries {
		oldJSON, _ := json.Marshal(e.OldState)
		newJSON, _ := json.Marshal(e.NewState)

		// ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4,
			argIndex+5, argIndex+6, argIndex+7, argIndex+8, argIndex+9, argIndex+10,
		)
		valueStrings = append(valueStrings, placeholders)

		valueArgs = append(valueArgs,
			e.ID,
			e.ActorID,
			e.ActorEmail,
			e.Action,
			e.Entity,
			e.EntityID,
			string(oldJSON),
			string(newJSON),
			e.IPAddress,
			e.UserAgent,
			e.Timestamp,
		)

		argIndex += 11
	}

	stmt := fmt.Sprintf(`
		INSERT INTO %s (
			id, actor_id, actor_email, action, entity, entity_id,
			old_state, new_state, ip_address, user_agent, created_at
		) VALUES %s
	`, s.tableName, strings.Join(valueStrings, ","))

	_, err := s.db.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("auditlog: postgres batch insert failed: %w", err)
	}

	return nil
}

// Query retrieves historical entries from PostgreSQL using parameterized filters.
func (s *PostgresStore) Query(ctx context.Context, filter Filter) ([]*Entry, error) {
	var whereClauses []string
	var args []any
	argIndex := 1

	if filter.Entity != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("entity = $%d", argIndex))
		args = append(args, filter.Entity)
		argIndex++
	}
	if filter.EntityID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("entity_id = $%d", argIndex))
		args = append(args, filter.EntityID)
		argIndex++
	}
	if filter.ActorID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_id = $%d", argIndex))
		args = append(args, filter.ActorID)
		argIndex++
	}
	if filter.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, filter.Action)
		argIndex++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	limit := 50
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset > 0 {
		offset = filter.Offset
	}

	query := fmt.Sprintf(`
		SELECT id, actor_id, actor_email, action, entity, entity_id,
		       old_state, new_state, ip_address, user_agent, created_at
		FROM %s
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, s.tableName, whereSQL, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("auditlog: postgres query failed: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		var e Entry
		var oldRaw, newRaw sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorEmail, &e.Action, &e.Entity, &e.EntityID,
			&oldRaw, &newRaw, &e.IPAddress, &e.UserAgent, &createdAt,
		)
		if err != nil {
			return nil, err
		}

		e.Timestamp = createdAt
		if oldRaw.Valid {
			var m any
			_ = json.Unmarshal([]byte(oldRaw.String), &m)
			e.OldState = m
		}
		if newRaw.Valid {
			var m any
			_ = json.Unmarshal([]byte(newRaw.String), &m)
			e.NewState = m
		}

		entries = append(entries, &e)
	}

	return entries, nil
}

// Close is a no-op for PostgresStore as the underlying *sql.DB lifecycle is managed externally.
func (s *PostgresStore) Close() error {
	return nil
}

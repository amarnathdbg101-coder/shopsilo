package gooutbox

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

// SQLStore provides production-grade persistence for outbox events backed by PostgreSQL / MySQL 8+.
type SQLStore struct {
	db                *sql.DB
	tableName         string
	topicColumn       string
	aggregateIDColumn string
}

// SQLOptions configures SQLStore parameters and column name mappings.
type SQLOptions struct {
	TableName         string // Default: "outbox_events"
	TopicColumn       string // Default: "topic" (can be "event_type", "action", etc.)
	AggregateIDColumn string // Default: "aggregate_id" (can be "shop_id", "user_id", etc.)
}

// NewSQLStore creates a new SQLStore backed by *sql.DB with optional column customizations.
func NewSQLStore(db *sql.DB, opts *SQLOptions) *SQLStore {
	tableName := "outbox_events"
	topicCol := "topic"
	aggCol := "aggregate_id"

	if opts != nil {
		if opts.TableName != "" {
			tableName = opts.TableName
		}
		if opts.TopicColumn != "" {
			topicCol = opts.TopicColumn
		}
		if opts.AggregateIDColumn != "" {
			aggCol = opts.AggregateIDColumn
		}
	}

	return &SQLStore{
		db:                db,
		tableName:         tableName,
		topicColumn:       topicCol,
		aggregateIDColumn: aggCol,
	}
}

// InitSchema creates the outbox table and high-performance indexing in PostgreSQL.
func (s *SQLStore) InitSchema(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			%s VARCHAR(128) NOT NULL,
			%s VARCHAR(128),
			payload BYTEA NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			retry_count INT NOT NULL DEFAULT 0,
			max_retries INT NOT NULL DEFAULT 5,
			last_error TEXT,
			next_retry_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMP WITH TIME ZONE
		);
		CREATE INDEX IF NOT EXISTS idx_%s_poll ON %s(status, next_retry_at ASC) WHERE status IN ('pending', 'failed');
	`, s.tableName, s.topicColumn, s.aggregateIDColumn, s.tableName, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Save writes an event inside the caller's active transaction (tx) or fallback DB connection.
// tx supports *sql.Tx, any ExecContext implementor, or any Exec(ctx, query, args...) implementor (e.g. pgx.Tx).
func (s *SQLStore) Save(ctx context.Context, tx any, event *Event) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (id, %s, %s, payload, status, retry_count, max_retries, next_retry_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.tableName, s.topicColumn, s.aggregateIDColumn)

	// 1. If tx is provided, attempt execution through transaction
	if tx != nil {
		// A. Standard database/sql transaction or executor
		if execer, ok := tx.(interface {
			ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		}); ok {
			_, err := execer.ExecContext(ctx, query,
				event.ID, event.Topic, event.AggregateID, event.Payload,
				string(event.Status), event.RetryCount, event.MaxRetries,
				event.NextRetryAt, event.CreatedAt,
			)
			return err
		}

		// B. Check for pgx.Tx or similar (Exec(ctx, sql, args...))
		val := reflect.ValueOf(tx)
		method := val.MethodByName("Exec")
		if method.IsValid() && method.Type().NumIn() >= 2 {
			args := []reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(query),
			}
			execArgs := []any{
				event.ID, event.Topic, event.AggregateID, event.Payload,
				string(event.Status), event.RetryCount, event.MaxRetries,
				event.NextRetryAt, event.CreatedAt,
			}
			for _, a := range execArgs {
				args = append(args, reflect.ValueOf(a))
			}
			res := method.Call(args)
			// Last return value is typically error
			if len(res) > 0 {
				lastVal := res[len(res)-1]
				if !lastVal.IsNil() {
					if err, ok := lastVal.Interface().(error); ok {
						return err
					}
				}
			}
			return nil
		}
	}

	// 2. Fallback to standard DB exec
	if s.db == nil {
		return ErrStoreNil
	}

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.Topic, event.AggregateID, event.Payload,
		string(event.Status), event.RetryCount, event.MaxRetries,
		event.NextRetryAt, event.CreatedAt,
	)
	return err
}

// FetchAndLock atomically selects and locks a batch of events using PostgreSQL FOR UPDATE SKIP LOCKED.
func (s *SQLStore) FetchAndLock(ctx context.Context, batchSize int) ([]*Event, error) {
	if batchSize <= 0 {
		batchSize = 50
	}
	if s.db == nil {
		return nil, ErrStoreNil
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'processing'
		WHERE id IN (
			SELECT id
			FROM %s
			WHERE status IN ('pending', 'failed')
			  AND next_retry_at <= NOW()
			ORDER BY next_retry_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, %s, %s, payload, status, retry_count, max_retries, COALESCE(last_error, ''), next_retry_at, created_at
	`, s.tableName, s.tableName, s.topicColumn, s.aggregateIDColumn)

	rows, err := s.db.QueryContext(ctx, query, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		var statusStr string
		var aggID sql.NullString
		var lastErr string

		if err := rows.Scan(
			&e.ID, &e.Topic, &aggID, &e.Payload,
			&statusStr, &e.RetryCount, &e.MaxRetries,
			&lastErr, &e.NextRetryAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}

		e.Status = Status(statusStr)
		if aggID.Valid {
			e.AggregateID = aggID.String
		}
		e.LastError = lastErr
		events = append(events, &e)
	}

	return events, rows.Err()
}

// MarkCompleted marks the event as completed.
func (s *SQLStore) MarkCompleted(ctx context.Context, id string) error {
	if s.db == nil {
		return ErrStoreNil
	}
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'completed', processed_at = NOW(), last_error = NULL
		WHERE id = $1
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// MarkFailed increments retry count and sets next retry schedule.
func (s *SQLStore) MarkFailed(ctx context.Context, id string, err error, nextRetryAfter time.Duration) error {
	if s.db == nil {
		return ErrStoreNil
	}
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'failed',
		    retry_count = retry_count + 1,
		    last_error = $2,
		    next_retry_at = NOW() + $3 * INTERVAL '1 second'
		WHERE id = $1
	`, s.tableName)

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	_, qErr := s.db.ExecContext(ctx, query, id, errMsg, int(nextRetryAfter.Seconds()))
	return qErr
}

// MarkDeadLetter moves an event to dead letter status.
func (s *SQLStore) MarkDeadLetter(ctx context.Context, id string, err error) error {
	if s.db == nil {
		return ErrStoreNil
	}
	query := fmt.Sprintf(`
		UPDATE %s
		SET status = 'dead_letter',
		    last_error = $2
		WHERE id = $1
	`, s.tableName)

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	_, qErr := s.db.ExecContext(ctx, query, id, errMsg)
	return qErr
}

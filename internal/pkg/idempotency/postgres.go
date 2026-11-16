package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"time"
)

// PostgresStorage implements the Storage interface backed by PostgreSQL using standard library database/sql.
// It leverages PostgreSQL session advisory locks (pg_try_advisory_lock) for sub-millisecond in-flight execution locking.
type PostgresStorage struct {
	db        *sql.DB
	tableName string
}

// NewPostgresStorage initializes a PostgreSQL-backed idempotency store.
func NewPostgresStorage(db *sql.DB, tableName ...string) *PostgresStorage {
	tbl := "idempotency_records"
	if len(tableName) > 0 && tableName[0] != "" {
		tbl = tableName[0]
	}
	return &PostgresStorage{
		db:        db,
		tableName: tbl,
	}
}

// hashKeyToInt64 hashes an arbitrary idempotency string into a signed 64-bit int for PostgreSQL advisory locks.
func hashKeyToInt64(key string) int64 {
	h := fnv.New64a()
	h.Write([]byte(key))
	return int64(h.Sum64())
}

// InitSchema creates the idempotency table and index if they do not exist.
func (p *PostgresStorage) InitSchema(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key VARCHAR(255) PRIMARY KEY,
			status_code INT NOT NULL,
			headers JSONB NOT NULL,
			body BYTEA NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_%s_expires ON %s (expires_at);
	`, p.tableName, p.tableName, p.tableName)

	_, err := p.db.ExecContext(ctx, query)
	return err
}

// Lock acquires a distributed in-flight lock using PostgreSQL advisory locks.
func (p *PostgresStorage) Lock(ctx context.Context, key string, _ time.Duration) (bool, error) {
	lockID := hashKeyToInt64(key)
	var acquired bool
	err := p.db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
	if err != nil {
		return false, fmt.Errorf("idempotency: advisory lock failed: %w", err)
	}
	return acquired, nil
}

// Unlock releases the PostgreSQL advisory lock.
func (p *PostgresStorage) Unlock(ctx context.Context, key string) error {
	lockID := hashKeyToInt64(key)
	var released bool
	err := p.db.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", lockID).Scan(&released)
	if err != nil {
		return fmt.Errorf("idempotency: advisory unlock failed: %w", err)
	}
	return nil
}

// Get retrieves a cached Record from PostgreSQL if not expired.
func (p *PostgresStorage) Get(ctx context.Context, key string) (*Record, error) {
	query := fmt.Sprintf(`
		SELECT status_code, headers, body, created_at, expires_at
		FROM %s
		WHERE key = $1 AND expires_at > NOW()
	`, p.tableName)

	var (
		statusCode   int
		headersJSON  []byte
		body         []byte
		createdAt    time.Time
		expiresAt    time.Time
	)

	err := p.db.QueryRowContext(ctx, query, key).Scan(&statusCode, &headersJSON, &body, &createdAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var headers http.Header
	if err := json.Unmarshal(headersJSON, &headers); err != nil {
		return nil, fmt.Errorf("idempotency: unmarshal headers: %w", err)
	}

	return &Record{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
		CreatedAt:  createdAt,
		ExpiresAt:  expiresAt,
	}, nil
}

// Set persists a completed response record into PostgreSQL with upsert semantics.
func (p *PostgresStorage) Set(ctx context.Context, key string, record *Record, ttl time.Duration) error {
	headersJSON, err := json.Marshal(record.Headers)
	if err != nil {
		return fmt.Errorf("idempotency: marshal headers: %w", err)
	}

	expiresAt := time.Now().Add(ttl)

	query := fmt.Sprintf(`
		INSERT INTO %s (key, status_code, headers, body, created_at, expires_at)
		VALUES ($1, $2, $3, $4, NOW(), $5)
		ON CONFLICT (key) DO UPDATE SET
			status_code = EXCLUDED.status_code,
			headers     = EXCLUDED.headers,
			body        = EXCLUDED.body,
			expires_at  = EXCLUDED.expires_at
	`, p.tableName)

	_, err = p.db.ExecContext(ctx, query, key, record.StatusCode, headersJSON, record.Body, expiresAt)
	return err
}

// Delete removes an idempotency record from PostgreSQL.
func (p *PostgresStorage) Delete(ctx context.Context, key string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE key = $1", p.tableName)
	_, err := p.db.ExecContext(ctx, query, key)
	return err
}

// PurgeExpired synchronously deletes all expired records to free PostgreSQL storage.
func (p *PostgresStorage) PurgeExpired(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at <= NOW()", p.tableName)
	res, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close implements the Storage interface. (Leaves *sql.DB lifecycle to caller).
func (p *PostgresStorage) Close() error {
	return nil
}

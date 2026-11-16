package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	UpdatePassword(ctx context.Context, id string, newPasswordHash string) error
}

type MemoryUserStore struct {
	mu      sync.RWMutex
	users   map[string]*User
	byEmail map[string]string
	counter int64
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users:   make(map[string]*User),
		byEmail: make(map[string]string),
	}
}

func (m *MemoryUserStore) Create(ctx context.Context, user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.byEmail[user.Email]; exists {
		return ErrEmailAlreadyExists
	}

	if user.ID == "" {
		user.ID = generateUniqueID()
	}

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	cloned := *user
	m.users[user.ID] = &cloned
	m.byEmail[user.Email] = user.ID
	return nil
}

func (m *MemoryUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, exists := m.byEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}

	user := m.users[id]
	cloned := *user
	return &cloned, nil
}

func (m *MemoryUserStore) GetByID(ctx context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	cloned := *user
	return &cloned, nil
}

func (m *MemoryUserStore) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrUserNotFound
	}

	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now()
	return nil
}

type PostgresUserStore struct {
	db        *sql.DB
	tableName string
}

type PostgresOptions struct {
	TableName string
}

func NewPostgresUserStore(db *sql.DB, opts *PostgresOptions) *PostgresUserStore {
	tableName := "users"
	if opts != nil && opts.TableName != "" {
		tableName = opts.TableName
	}
	return &PostgresUserStore{
		db:        db,
		tableName: tableName,
	}
}

func (s *PostgresUserStore) Create(ctx context.Context, user *User) error {
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	if user.ID == "" {
		user.ID = generateUniqueID()
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("auth: postgres insert failed: %w", err)
	}
	return nil
}

func (s *PostgresUserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM %s
		WHERE email = $1
		LIMIT 1
	`, s.tableName)

	row := s.db.QueryRowContext(ctx, query, email)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("auth: postgres query failed: %w", err)
	}
	return &u, nil
}

func (s *PostgresUserStore) GetByID(ctx context.Context, id string) (*User, error) {
	query := fmt.Sprintf(`
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM %s
		WHERE id = $1
		LIMIT 1
	`, s.tableName)

	row := s.db.QueryRowContext(ctx, query, id)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("auth: postgres query failed: %w", err)
	}
	return &u, nil
}

func (s *PostgresUserStore) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`, s.tableName)

	result, err := s.db.ExecContext(ctx, query, newPasswordHash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("auth: postgres update password failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "23505") || contains(err.Error(), "unique constraint") || contains(err.Error(), "duplicate key")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && searchSubstr(s, substr)))
}

func searchSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateUniqueID() string {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return fmt.Sprintf("usr_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("usr_%d_%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

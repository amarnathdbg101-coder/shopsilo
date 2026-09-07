// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateEmail  = errors.New("email already registered")
)

type UserRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewUserRepo(db *pgxpool.Pool, logger *zap.Logger) *UserRepo {
	return &UserRepo{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (email, password_hash, full_name, phone, role, is_active, created_at, updated_at)
		VALUES (LOWER($1), $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, email, password_hash, full_name, COALESCE(phone, ''), role, is_active, created_at, updated_at
	`
	created := &model.User{}
	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(u.Email),
		u.PasswordHash,
		strings.TrimSpace(u.FullName),
		u.Phone,
		u.Role,
		u.IsActive,
	).Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.FullName,
		&created.Phone,
		&created.Role,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return nil, ErrDuplicateEmail
		}
		r.logger.Error("failed to create user", zap.Error(err), zap.String("email", u.Email))
		return nil, err
	}
	return created, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER(TRIM($1))
		LIMIT 1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Phone,
		&u.AvatarURL,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		r.logger.Error("failed to find user by email", zap.Error(err), zap.String("email", email))
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Phone,
		&u.AvatarURL,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		r.logger.Error("failed to find user by id", zap.Error(err), zap.String("id", id))
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) UpdateAvatar(ctx context.Context, userID, avatarURL string) error {
	query := `
		UPDATE users
		SET avatar_url = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, avatarURL, userID)
	if err != nil {
		r.logger.Error("failed to update user avatar", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		r.logger.Error("failed to update user password", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, userID, role string) error {
	query := `
		UPDATE users
		SET role = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, role, userID)
	if err != nil {
		r.logger.Error("failed to update user role", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdateRoleWithTx(ctx context.Context, tx pgx.Tx, userID, role string) error {
	query := `
		UPDATE users
		SET role = $1, updated_at = NOW()
		WHERE id = $2
	`
	cmdTag, err := tx.Exec(ctx, query, role, userID)
	if err != nil {
		r.logger.Error("failed to update user role in tx", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}




// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"fmt"
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

func (r *UserRepo) FindByEmailOrPhone(ctx context.Context, identifier string) (*model.User, error) {
	trimmed := strings.TrimSpace(identifier)
	query := `
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1) OR (phone != '' AND phone = $1)
		LIMIT 1
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, trimmed).Scan(
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
		r.logger.Error("failed to find user by email or phone", zap.Error(err), zap.String("identifier", identifier))
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

func (r *UserRepo) DeactivateUser(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`
	cmdTag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.logger.Error("failed to deactivate user", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdateProfile(ctx context.Context, userID, fullName, phone string) (*model.User, error) {
	query := `
		UPDATE users
		SET full_name = $1, phone = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, strings.TrimSpace(fullName), strings.TrimSpace(phone), userID).Scan(
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
		r.logger.Error("failed to update user profile", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) ListUsersForAdmin(ctx context.Context, role, search string, limit, offset int) ([]*model.User, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if role != "" && role != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, strings.TrimSpace(role))
		argIdx++
	}

	if search != "" {
		searchTerm := "%" + strings.TrimSpace(search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query admin users", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	totalCount := 0

	for rows.Next() {
		u := &model.User{}
		err := rows.Scan(
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
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan admin user row", zap.Error(err))
			return nil, 0, err
		}
		users = append(users, u)
	}

	if users == nil {
		users = []*model.User{}
	}

	return users, totalCount, nil
}

func (r *UserRepo) UpdateUserStatusForAdmin(ctx context.Context, userID string, isActive *bool, role *string) (*model.User, error) {
	var cleanRole *string
	if role != nil && strings.TrimSpace(*role) != "" {
		trimmed := strings.TrimSpace(*role)
		cleanRole = &trimmed
	}

	query := `
		UPDATE users
		SET is_active = COALESCE($1, is_active),
		    role = COALESCE($2, role),
		    updated_at = NOW()
		WHERE id = $3
		RETURNING id, email, password_hash, full_name, COALESCE(phone, ''), COALESCE(avatar_url, ''), role, is_active, created_at, updated_at
	`
	u := &model.User{}
	err := r.db.QueryRow(ctx, query, isActive, cleanRole, userID).Scan(
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
		r.logger.Error("failed to update user status by admin", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}
	return u, nil
}


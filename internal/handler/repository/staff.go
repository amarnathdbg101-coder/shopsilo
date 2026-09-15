// Package repository handles database queries.
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
	ErrStaffNotFound      = errors.New("staff member not found")
	ErrStaffPhoneExists   = errors.New("phone number already registered for another staff member in this shop")
	ErrInvalidStaffPIN    = errors.New("invalid 4-digit PIN or staff credentials")
	ErrStaffDeactivated   = errors.New("staff account is deactivated")
)

type StaffRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewStaffRepo(db *pgxpool.Pool, logger *zap.Logger) *StaffRepo {
	return &StaffRepo{
		db:     db,
		logger: logger,
	}
}

func (r *StaffRepo) Create(ctx context.Context, s *model.ShopStaff) (*model.ShopStaff, error) {
	query := `
		INSERT INTO shop_staff (shop_id, full_name, phone, pin_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, shop_id, full_name, phone, pin_hash, role, is_active, created_at, updated_at
	`
	created := &model.ShopStaff{}
	err := r.db.QueryRow(
		ctx,
		query,
		s.ShopID,
		strings.TrimSpace(s.FullName),
		strings.TrimSpace(s.Phone),
		s.PinHash,
		s.Role,
		s.IsActive,
	).Scan(
		&created.ID,
		&created.ShopID,
		&created.FullName,
		&created.Phone,
		&created.PinHash,
		&created.Role,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrStaffPhoneExists
		}
		r.logger.Error("failed to create staff member", zap.Error(err), zap.String("shop_id", s.ShopID))
		return nil, err
	}
	return created, nil
}

func (r *StaffRepo) FindByPhoneAndShopID(ctx context.Context, shopID, phone string) (*model.ShopStaff, error) {
	query := `
		SELECT id, shop_id, full_name, phone, pin_hash, role, is_active, created_at, updated_at
		FROM shop_staff
		WHERE shop_id = $1 AND phone = $2
		LIMIT 1
	`
	s := &model.ShopStaff{}
	err := r.db.QueryRow(ctx, query, shopID, strings.TrimSpace(phone)).Scan(
		&s.ID,
		&s.ShopID,
		&s.FullName,
		&s.Phone,
		&s.PinHash,
		&s.Role,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStaffNotFound
		}
		r.logger.Error("failed to find staff member", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	return s, nil
}

func (r *StaffRepo) FindByID(ctx context.Context, id, shopID string) (*model.ShopStaff, error) {
	query := `
		SELECT id, shop_id, full_name, phone, pin_hash, role, is_active, created_at, updated_at
		FROM shop_staff
		WHERE id = $1 AND shop_id = $2
		LIMIT 1
	`
	s := &model.ShopStaff{}
	err := r.db.QueryRow(ctx, query, id, shopID).Scan(
		&s.ID,
		&s.ShopID,
		&s.FullName,
		&s.Phone,
		&s.PinHash,
		&s.Role,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStaffNotFound
		}
		r.logger.Error("failed to find staff member by id", zap.Error(err), zap.String("id", id))
		return nil, err
	}
	return s, nil
}

func (r *StaffRepo) ListByShopID(ctx context.Context, shopID string) ([]*model.ShopStaff, error) {
	query := `
		SELECT id, shop_id, full_name, phone, role, is_active, created_at, updated_at
		FROM shop_staff
		WHERE shop_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to list shop staff", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*model.ShopStaff
	for rows.Next() {
		s := &model.ShopStaff{}
		err := rows.Scan(
			&s.ID,
			&s.ShopID,
			&s.FullName,
			&s.Phone,
			&s.Role,
			&s.IsActive,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}

	if list == nil {
		list = []*model.ShopStaff{}
	}

	return list, nil
}

func (r *StaffRepo) Update(ctx context.Context, s *model.ShopStaff) (*model.ShopStaff, error) {
	query := `
		UPDATE shop_staff
		SET full_name = $1, phone = $2, pin_hash = $3, role = $4, is_active = $5, updated_at = NOW()
		WHERE id = $6 AND shop_id = $7
		RETURNING id, shop_id, full_name, phone, pin_hash, role, is_active, created_at, updated_at
	`
	updated := &model.ShopStaff{}
	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(s.FullName),
		strings.TrimSpace(s.Phone),
		s.PinHash,
		s.Role,
		s.IsActive,
		s.ID,
		s.ShopID,
	).Scan(
		&updated.ID,
		&updated.ShopID,
		&updated.FullName,
		&updated.Phone,
		&updated.PinHash,
		&updated.Role,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStaffNotFound
		}
		r.logger.Error("failed to update staff member", zap.Error(err), zap.String("id", s.ID))
		return nil, err
	}
	return updated, nil
}

func (r *StaffRepo) Delete(ctx context.Context, id, shopID string) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM shop_staff WHERE id = $1 AND shop_id = $2`, id, shopID)
	if err != nil {
		r.logger.Error("failed to delete staff member", zap.Error(err), zap.String("id", id))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrStaffNotFound
	}
	return nil
}

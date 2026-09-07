// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
)

type CategoryRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewCategoryRepo(db *pgxpool.Pool, logger *zap.Logger) *CategoryRepo {
	return &CategoryRepo{
		db:     db,
		logger: logger,
	}
}

func (r *CategoryRepo) FindAll(ctx context.Context) ([]*model.Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE is_active = true
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to list categories", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var categories []*model.Category
	for rows.Next() {
		c := &model.Category{}
		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Slug,
			&c.Description,
			&c.ImageURL,
			&c.ParentID,
			&c.IsActive,
			&c.CreatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan category row", zap.Error(err))
			return nil, err
		}
		categories = append(categories, c)
	}

	if categories == nil {
		categories = []*model.Category{}
	}
	return categories, nil
}

func (r *CategoryRepo) FindByID(ctx context.Context, id string) (*model.Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE id = $1 AND is_active = true
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.Name,
		&c.Slug,
		&c.Description,
		&c.ImageURL,
		&c.ParentID,
		&c.IsActive,
		&c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		r.logger.Error("failed to find category by id", zap.Error(err), zap.String("category_id", id))
		return nil, err
	}
	return c, nil
}

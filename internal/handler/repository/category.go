// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"time"

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
		WHERE id = $1
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

func (r *CategoryRepo) FindBySlug(ctx context.Context, slug string) (*model.Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE slug = $1
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, slug).Scan(
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
		r.logger.Error("failed to find category by slug", zap.Error(err), zap.String("slug", slug))
		return nil, err
	}
	return c, nil
}

func (r *CategoryRepo) AdminFindAll(ctx context.Context) ([]*dto.AdminCategoryItem, error) {
	query := `
		SELECT c.id, c.name, c.slug, COALESCE(c.description, ''), COALESCE(c.image_url, ''), c.parent_id, c.is_active,
		       COALESCE(p.cnt, 0) AS product_count, c.created_at
		FROM categories c
		LEFT JOIN (
			SELECT category_id, COUNT(*) AS cnt
			FROM products
			GROUP BY category_id
		) p ON p.category_id = c.id
		ORDER BY c.created_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to list admin categories", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var list []*dto.AdminCategoryItem
	for rows.Next() {
		item := &dto.AdminCategoryItem{}
		var createdAt time.Time
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.ImageURL,
			&item.ParentID,
			&item.IsActive,
			&item.ProductCount,
			&createdAt,
		)
		if err != nil {
			r.logger.Error("failed to scan admin category item", zap.Error(err))
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, item)
	}

	if list == nil {
		list = []*dto.AdminCategoryItem{}
	}
	return list, nil
}

func (r *CategoryRepo) Create(ctx context.Context, c *model.Category) (*model.Category, error) {
	query := `
		INSERT INTO categories (name, slug, description, image_url, parent_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
	`
	created := &model.Category{}
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.Slug,
		c.Description,
		c.ImageURL,
		c.ParentID,
		c.IsActive,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.ImageURL,
		&created.ParentID,
		&created.IsActive,
		&created.CreatedAt,
	)
	if err != nil {
		r.logger.Error("failed to create category", zap.Error(err), zap.String("name", c.Name))
		return nil, err
	}
	return created, nil
}

func (r *CategoryRepo) Update(ctx context.Context, c *model.Category) (*model.Category, error) {
	query := `
		UPDATE categories
		SET name = $1, slug = $2, description = $3, image_url = $4, parent_id = $5, is_active = $6
		WHERE id = $7
		RETURNING id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
	`
	updated := &model.Category{}
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.Slug,
		c.Description,
		c.ImageURL,
		c.ParentID,
		c.IsActive,
		c.ID,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.ImageURL,
		&updated.ParentID,
		&updated.IsActive,
		&updated.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		r.logger.Error("failed to update category", zap.Error(err), zap.String("category_id", c.ID))
		return nil, err
	}
	return updated, nil
}

func (r *CategoryRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM categories WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete category", zap.Error(err), zap.String("category_id", id))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}


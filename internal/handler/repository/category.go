// Package repository handle db query.
package repository

import (
	"shopMe/internal/reuse"
	"strings"
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrCategoryNotFound = errors.New("category not found")

	catCacheMu       sync.RWMutex
	cachedCategories []*model.Category
	catCacheExpires  time.Time
)

func invalidateCategoryCache() {
	catCacheMu.Lock()
	cachedCategories = nil
	catCacheExpires = time.Time{}
	catCacheMu.Unlock()
}

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
	now := time.Now()
	catCacheMu.RLock()
	if len(cachedCategories) > 0 && now.Before(catCacheExpires) {
		result := make([]*model.Category, len(cachedCategories))
		copy(result, cachedCategories)
		catCacheMu.RUnlock()
		return result, nil
	}
	catCacheMu.RUnlock()

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

	catCacheMu.Lock()
	cachedCategories = categories
	catCacheExpires = now.Add(5 * time.Minute)
	catCacheMu.Unlock()

	result := make([]*model.Category, len(categories))
	copy(result, categories)
	return result, nil
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

func (r *CategoryRepo) FindByNameOrSlug(ctx context.Context, term string) (*model.Category, error) {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" {
		return nil, ErrCategoryNotFound
	}
	slug := reuse.Slugify(trimmed)

	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE is_active = true AND (
			slug = $1 OR 
			slug = $2 OR 
			LOWER(name) = $3 OR 
			name ILIKE $4
		)
		ORDER BY 
			CASE WHEN slug = $1 THEN 1 WHEN slug = $2 THEN 2 WHEN LOWER(name) = $3 THEN 3 ELSE 4 END ASC
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, strings.ToLower(trimmed), slug, strings.ToLower(trimmed), "%"+trimmed+"%").Scan(
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
		return nil, err
	}
	return c, nil
}

func (r *CategoryRepo) FindDefault(ctx context.Context) (*model.Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE is_active = true
		ORDER BY CASE WHEN slug IN ('general-store', 'kirana-grocery') THEN 0 ELSE 1 END, name ASC
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query).Scan(
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
	invalidateCategoryCache()
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
	invalidateCategoryCache()
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
	invalidateCategoryCache()
	return nil
}

// ListByShop returns all active global categories combined with custom categories created by this shop.
// Custom categories created by the shop appear first.
func (r *CategoryRepo) ListByShop(ctx context.Context, shopID string) ([]*dto.ShopCategoryItem, error) {
	query := `
		SELECT c.id, c.name, c.slug, COALESCE(c.description, ''), COALESCE(c.image_url, ''),
		       COALESCE(c.icon, '📦'), c.shop_id, (c.shop_id IS NOT NULL) AS is_custom, c.is_active,
		       COALESCE(p.cnt, 0) AS product_count, c.created_at
		FROM categories c
		LEFT JOIN (
			SELECT category_id, COUNT(*) AS cnt
			FROM products
			WHERE shop_id = $1
			GROUP BY category_id
		) p ON p.category_id = c.id
		WHERE (c.shop_id = $1 OR c.shop_id IS NULL) AND c.is_active = true
		ORDER BY (c.shop_id IS NOT NULL) DESC, c.name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to list shop categories", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*dto.ShopCategoryItem
	for rows.Next() {
		item := &dto.ShopCategoryItem{}
		var createdAt time.Time
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.ImageURL,
			&item.Icon,
			&item.ShopID,
			&item.IsCustom,
			&item.IsActive,
			&item.ProductCount,
			&createdAt,
		)
		if err != nil {
			r.logger.Error("failed to scan shop category item", zap.Error(err))
			return nil, err
		}
		item.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, item)
	}

	if list == nil {
		list = []*dto.ShopCategoryItem{}
	}
	return list, nil
}

// FindShopCategoryBySlug looks up a category by slug either within the shop's custom categories or global defaults.
func (r *CategoryRepo) FindShopCategoryBySlug(ctx context.Context, shopID, slug string) (*model.Category, error) {
	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE slug = $1 AND (shop_id = $2 OR shop_id IS NULL) AND is_active = true
		ORDER BY (shop_id IS NOT NULL) DESC
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, slug, shopID).Scan(
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
		return nil, err
	}
	return c, nil
}

// FindShopCategoryByNameOrSlug looks up a category by slug or name, checking the shop's
// custom categories first, and then falling back to active global default categories.
func (r *CategoryRepo) FindShopCategoryByNameOrSlug(ctx context.Context, shopID, term string) (*model.Category, error) {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" {
		return nil, ErrCategoryNotFound
	}
	slug := reuse.Slugify(trimmed)

	query := `
		SELECT id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), parent_id, is_active, created_at
		FROM categories
		WHERE (shop_id = $1 OR shop_id IS NULL) AND is_active = true AND (
			slug = $2 OR 
			slug = $3 OR 
			LOWER(name) = $4 OR 
			name ILIKE $5
		)
		ORDER BY 
			(shop_id IS NOT NULL) DESC,
			CASE WHEN slug = $2 THEN 1 WHEN slug = $3 THEN 2 WHEN LOWER(name) = $4 THEN 3 ELSE 4 END ASC
		LIMIT 1
	`
	c := &model.Category{}
	err := r.db.QueryRow(ctx, query, shopID, strings.ToLower(trimmed), slug, strings.ToLower(trimmed), "%"+trimmed+"%").Scan(
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
		return nil, err
	}
	return c, nil
}

// CreateShopCategory inserts a custom category belonging specifically to a shop.
func (r *CategoryRepo) CreateShopCategory(ctx context.Context, shopID string, c *model.Category) (*model.Category, error) {
	icon := strings.TrimSpace(c.Icon)
	if icon == "" {
		icon = "📦"
	}

	query := `
		INSERT INTO categories (name, slug, description, image_url, icon, shop_id, is_custom, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, true, NOW())
		RETURNING id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), COALESCE(icon, '📦'), shop_id, is_custom, is_active, created_at
	`
	created := &model.Category{}
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.Slug,
		c.Description,
		c.ImageURL,
		icon,
		shopID,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.ImageURL,
		&created.Icon,
		&created.ShopID,
		&created.IsCustom,
		&created.IsActive,
		&created.CreatedAt,
	)
	if err != nil {
		r.logger.Error("failed to create shop category", zap.Error(err), zap.String("shop_id", shopID), zap.String("name", c.Name))
		return nil, err
	}
	invalidateCategoryCache()
	return created, nil
}

// UpdateShopCategory updates a custom category. Only the owning shop can modify it.
func (r *CategoryRepo) UpdateShopCategory(ctx context.Context, shopID, categoryID string, c *model.Category) (*model.Category, error) {
	icon := strings.TrimSpace(c.Icon)
	if icon == "" {
		icon = "📦"
	}

	query := `
		UPDATE categories
		SET name = $1, slug = $2, description = $3, image_url = $4, icon = $5, is_active = $6
		WHERE id = $7 AND shop_id = $8
		RETURNING id, name, slug, COALESCE(description, ''), COALESCE(image_url, ''), COALESCE(icon, '📦'), shop_id, is_custom, is_active, created_at
	`
	updated := &model.Category{}
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.Slug,
		c.Description,
		c.ImageURL,
		icon,
		c.IsActive,
		categoryID,
		shopID,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.ImageURL,
		&updated.Icon,
		&updated.ShopID,
		&updated.IsCustom,
		&updated.IsActive,
		&updated.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		r.logger.Error("failed to update shop category", zap.Error(err), zap.String("category_id", categoryID))
		return nil, err
	}
	invalidateCategoryCache()
	return updated, nil
}

// DeleteShopCategory removes a shop-specific custom category if owned by the shop.
func (r *CategoryRepo) DeleteShopCategory(ctx context.Context, shopID, categoryID string) error {
	query := `DELETE FROM categories WHERE id = $1 AND shop_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, categoryID, shopID)
	if err != nil {
		r.logger.Error("failed to delete shop category", zap.Error(err), zap.String("category_id", categoryID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	invalidateCategoryCache()
	return nil
}

// CountProductsByCategoryID returns how many products currently belong to this category for the shop.
func (r *CategoryRepo) CountProductsByCategoryID(ctx context.Context, shopID, categoryID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE category_id = $1 AND shop_id = $2`, categoryID, shopID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}


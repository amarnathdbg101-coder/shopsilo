// Package repository handle db query.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrProductSlugTaken = errors.New("product slug is already in use")
	ErrSKUTaken         = errors.New("product SKU is already in use")
)

type ProductRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewProductRepo(db *pgxpool.Pool, logger *zap.Logger) *ProductRepo {
	return &ProductRepo{
		db:     db,
		logger: logger,
	}
}

func computeProfit(p *model.Product) {
	if p == nil {
		return
	}
	if p.Inventory != nil {
		p.StockQuantity = p.Inventory.AvailableQuantity
		if p.Inventory.LowStockThreshold <= 0 {
			p.Inventory.LowStockThreshold = 1
		}
		p.LowStockThreshold = p.Inventory.LowStockThreshold
		p.MinStock = p.Inventory.LowStockThreshold
	} else if p.LowStockThreshold <= 0 {
		p.LowStockThreshold = 1
		p.MinStock = 1
	} else {
		p.MinStock = p.LowStockThreshold
	}
	if p.CostPrice > 0 {
		p.UnitProfit = p.Price - p.CostPrice
		p.ProfitMarginPct = math.Round((p.UnitProfit/p.Price)*1000) / 10
	}
}

func (r *ProductRepo) Create(ctx context.Context, p *model.Product, initialStock int) (*model.Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	imagesJSON, _ := json.Marshal(p.Images)
	if p.Images == nil {
		imagesJSON = []byte("[]")
	}
	attributesJSON, _ := json.Marshal(p.Attributes)
	if p.Attributes == nil {
		attributesJSON = []byte("{}")
	}

	query := `
		INSERT INTO products (shop_id, name, slug, description, sku, price, cost_price, compare_price, category_id, images, weight, is_active, is_featured, tags, attributes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0), category_id, images, COALESCE(weight, 0), is_active, is_featured, tags, COALESCE(attributes, '{}'::jsonb), created_at, updated_at
	`
	created := &model.Product{}
	var imagesBytes, attributesBytes []byte

	err = tx.QueryRow(
		ctx,
		query,
		p.ShopID,
		strings.TrimSpace(p.Name),
		strings.TrimSpace(p.Slug),
		strings.TrimSpace(p.Description),
		strings.TrimSpace(p.SKU),
		p.Price,
		p.CostPrice,
		p.ComparePrice,
		p.CategoryID,
		imagesJSON,
		p.Weight,
		p.IsActive,
		p.IsFeatured,
		p.Tags,
		attributesJSON,
	).Scan(
		&created.ID,
		&created.ShopID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.SKU,
		&created.Price,
		&created.CostPrice,
		&created.ComparePrice,
		&created.CategoryID,
		&imagesBytes,
		&created.Weight,
		&created.IsActive,
		&created.IsFeatured,
		&created.Tags,
		&attributesBytes,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "sku") {
				return nil, ErrSKUTaken
			}
			return nil, ErrProductSlugTaken
		}
		r.logger.Error("failed to create product", zap.Error(err), zap.String("shop_id", p.ShopID))
		return nil, err
	}

	created.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &created.Images)
	}

	created.Attributes = make(map[string]interface{})
	if len(attributesBytes) > 0 {
		_ = json.Unmarshal(attributesBytes, &created.Attributes)
	}

	// Insert into inventory with shop-owner configured min stock (default: 1)
	lowStock := 1
	if p.LowStockThreshold > 0 {
		lowStock = p.LowStockThreshold
	} else if p.MinStock > 0 {
		lowStock = p.MinStock
	}

	invQuery := `
		INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
		VALUES ($1, $2, 0, $3, NOW())
	`
	if _, err := tx.Exec(ctx, invQuery, created.ID, initialStock, lowStock); err != nil {
		r.logger.Error("failed to create inventory for product", zap.Error(err), zap.String("product_id", created.ID))
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit product creation: %w", err)
	}

	created.Inventory = &model.Inventory{
		ProductID:         created.ID,
		Quantity:          initialStock,
		ReservedQuantity:  0,
		AvailableQuantity: initialStock,
		LowStockThreshold: lowStock,
	}
	created.MinStock = lowStock
	created.LowStockThreshold = lowStock

	computeProfit(created)
	return created, nil
}

func (r *ProductRepo) FindByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(p.floor_price, 0), COALESCE(p.allow_bargain, true)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1 AND p.is_active = true
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes, attributesBytes []byte
	inv := &model.Inventory{ProductID: id}

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.ShopID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.SKU,
		&p.Price,
		&p.CostPrice,
		&p.ComparePrice,
		&p.CategoryID,
		&imagesBytes,
		&p.Weight,
		&p.IsActive,
		&p.IsFeatured,
		&p.Tags,
		&attributesBytes,
		&p.CreatedAt,
		&p.UpdatedAt,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
		&p.FloorPrice,
		&p.AllowBargain,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		r.logger.Error("failed to find product by id", zap.Error(err), zap.String("product_id", id))
		return nil, err
	}

	p.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &p.Images)
	}

	p.Attributes = make(map[string]interface{})
	if len(attributesBytes) > 0 {
		_ = json.Unmarshal(attributesBytes, &p.Attributes)
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

// FindByIDs fetches multiple active products belonging to shopID in a single batch query with inventory JOIN.
func (r *ProductRepo) FindByIDs(ctx context.Context, shopID string, ids []string) (map[string]*model.Product, error) {
	if len(ids) == 0 {
		return make(map[string]*model.Product), nil
	}

	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(p.floor_price, 0), COALESCE(p.allow_bargain, true)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND p.id = ANY($2) AND p.is_active = true
	`

	rows, err := r.db.Query(ctx, query, shopID, ids)
	if err != nil {
		r.logger.Error("failed to query products by ids", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	productsMap := make(map[string]*model.Product, len(ids))
	for rows.Next() {
		p := &model.Product{}
		var imagesBytes, attributesBytes []byte
		inv := &model.Inventory{}

		err := rows.Scan(
			&p.ID,
			&p.ShopID,
			&p.Name,
			&p.Slug,
			&p.Description,
			&p.SKU,
			&p.Price,
			&p.CostPrice,
			&p.ComparePrice,
			&p.CategoryID,
			&imagesBytes,
			&p.Weight,
			&p.IsActive,
			&p.IsFeatured,
			&p.Tags,
			&attributesBytes,
			&p.CreatedAt,
			&p.UpdatedAt,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.LowStockThreshold,
			&p.FloorPrice,
			&p.AllowBargain,
		)
		if err != nil {
			r.logger.Error("failed to scan product batch row", zap.Error(err))
			return nil, err
		}

		p.Images = []string{}
		if len(imagesBytes) > 0 {
			_ = json.Unmarshal(imagesBytes, &p.Images)
		}

		p.Attributes = make(map[string]interface{})
		if len(attributesBytes) > 0 {
			_ = json.Unmarshal(attributesBytes, &p.Attributes)
		}

		inv.ProductID = p.ID
		inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
		p.Inventory = inv

		computeProfit(p)
		productsMap[p.ID] = p
	}

	return productsMap, nil
}

func (r *ProductRepo) FindBySlug(ctx context.Context, slug string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.slug = LOWER(TRIM($1)) AND p.is_active = true
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes, attributesBytes []byte
	inv := &model.Inventory{}

	err := r.db.QueryRow(ctx, query, slug).Scan(
		&p.ID,
		&p.ShopID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.SKU,
		&p.Price,
		&p.CostPrice,
		&p.ComparePrice,
		&p.CategoryID,
		&imagesBytes,
		&p.Weight,
		&p.IsActive,
		&p.IsFeatured,
		&p.Tags,
		&attributesBytes,
		&p.CreatedAt,
		&p.UpdatedAt,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		r.logger.Error("failed to find product by slug", zap.Error(err), zap.String("slug", slug))
		return nil, err
	}

	p.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &p.Images)
	}

	p.Attributes = make(map[string]interface{})
	if len(attributesBytes) > 0 {
		_ = json.Unmarshal(attributesBytes, &p.Attributes)
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

// FindBySKU looks up an active product in a shop by its SKU or barcode.
func (r *ProductRepo) FindBySKU(ctx context.Context, shopID, sku string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND UPPER(TRIM(p.sku)) = UPPER(TRIM($2))
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes, attributesBytes []byte
	inv := &model.Inventory{}

	err := r.db.QueryRow(ctx, query, shopID, sku).Scan(
		&p.ID,
		&p.ShopID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.SKU,
		&p.Price,
		&p.CostPrice,
		&p.ComparePrice,
		&p.CategoryID,
		&imagesBytes,
		&p.Weight,
		&p.IsActive,
		&p.IsFeatured,
		&p.Tags,
		&attributesBytes,
		&p.CreatedAt,
		&p.UpdatedAt,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		r.logger.Error("failed to find product by sku", zap.Error(err), zap.String("shop_id", shopID), zap.String("sku", sku))
		return nil, err
	}

	p.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &p.Images)
	}

	p.Attributes = make(map[string]interface{})
	if len(attributesBytes) > 0 {
		_ = json.Unmarshal(attributesBytes, &p.Attributes)
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

func (r *ProductRepo) FindAll(ctx context.Context, filter dto.ProductFilter) ([]*model.Product, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, "p.is_active = true")

	if filter.ShopID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.shop_id = $%d", argIdx))
		args = append(args, filter.ShopID)
		argIdx++
	}

	if filter.CategoryID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.category_id = $%d", argIdx))
		args = append(args, filter.CategoryID)
		argIdx++
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR p.sku ILIKE $%d OR p.description ILIKE $%d OR p.slug ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	if filter.MinPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("p.price >= $%d", argIdx))
		args = append(args, filter.MinPrice)
		argIdx++
	}

	if filter.MaxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("p.price <= $%d", argIdx))
		args = append(args, filter.MaxPrice)
		argIdx++
	}

	if filter.IsFeatured != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("p.is_featured = $%d", argIdx))
		args = append(args, *filter.IsFeatured)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	orderBy := "p.created_at DESC"
	switch filter.SortBy {
	case "price_asc":
		orderBy = "p.price ASC"
	case "price_desc":
		orderBy = "p.price DESC"
	case "newest":
		orderBy = "p.created_at DESC"
	case "oldest":
		orderBy = "p.created_at ASC"
	case "stock_desc":
		orderBy = "COALESCE(i.quantity, 0) DESC, p.created_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(s.phone, ''), COALESCE(s.address, ''), COALESCE(s.city, ''),
		       s.latitude, s.longitude, COALESCE(c.name, ''),
		       COUNT(*) OVER() AS total_count
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN shops s ON s.id = p.shop_id
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query products", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var products []*model.Product
	totalCount := 0

	for rows.Next() {
		p := &model.Product{}
		var imagesBytes, attributesBytes []byte
		inv := &model.Inventory{}

		err := rows.Scan(
			&p.ID,
			&p.ShopID,
			&p.Name,
			&p.Slug,
			&p.Description,
			&p.SKU,
			&p.Price,
			&p.CostPrice,
			&p.ComparePrice,
			&p.CategoryID,
			&imagesBytes,
			&p.Weight,
			&p.IsActive,
			&p.IsFeatured,
			&p.Tags,
			&attributesBytes,
			&p.CreatedAt,
			&p.UpdatedAt,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.LowStockThreshold,
			&p.ShopName,
			&p.ShopSlug,
			&p.ShopPhone,
			&p.ShopAddress,
			&p.ShopCity,
			&p.ShopLatitude,
			&p.ShopLongitude,
			&p.CategoryName,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan product row", zap.Error(err))
			return nil, 0, err
		}

		p.Images = []string{}
		if len(imagesBytes) > 0 {
			_ = json.Unmarshal(imagesBytes, &p.Images)
		}

		p.Attributes = make(map[string]interface{})
		if len(attributesBytes) > 0 {
			_ = json.Unmarshal(attributesBytes, &p.Attributes)
		}

		inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
		p.Inventory = inv

		computeProfit(p)
		products = append(products, p)
	}

	return products, totalCount, nil
}

func (r *ProductRepo) Update(ctx context.Context, p *model.Product, stock *int) (*model.Product, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	imagesJSON, _ := json.Marshal(p.Images)
	if p.Images == nil {
		imagesJSON = []byte("[]")
	}

	attributesJSON, _ := json.Marshal(p.Attributes)
	if p.Attributes == nil {
		attributesJSON = []byte("{}")
	}

	query := `
		UPDATE products
		SET name = $1, slug = $2, description = $3, sku = $4, price = $5, cost_price = $6, compare_price = $7,
		    category_id = $8, images = $9, weight = $10, is_active = $11, is_featured = $12, tags = $13, attributes = $14, updated_at = NOW()
		WHERE id = $15 AND shop_id = $16
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0), category_id, images, COALESCE(weight, 0), is_active, is_featured, tags, COALESCE(attributes, '{}'::jsonb), created_at, updated_at
	`
	updated := &model.Product{}
	var imagesBytes, attributesBytes []byte

	err = tx.QueryRow(
		ctx,
		query,
		strings.TrimSpace(p.Name),
		strings.TrimSpace(p.Slug),
		strings.TrimSpace(p.Description),
		strings.TrimSpace(p.SKU),
		p.Price,
		p.CostPrice,
		p.ComparePrice,
		p.CategoryID,
		imagesJSON,
		p.Weight,
		p.IsActive,
		p.IsFeatured,
		p.Tags,
		attributesJSON,
		p.ID,
		p.ShopID,
	).Scan(
		&updated.ID,
		&updated.ShopID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.SKU,
		&updated.Price,
		&updated.CostPrice,
		&updated.ComparePrice,
		&updated.CategoryID,
		&imagesBytes,
		&updated.Weight,
		&updated.IsActive,
		&updated.IsFeatured,
		&updated.Tags,
		&attributesBytes,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "sku") {
				return nil, ErrSKUTaken
			}
			return nil, ErrProductSlugTaken
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		r.logger.Error("failed to update product", zap.Error(err), zap.String("product_id", p.ID))
		return nil, err
	}

	updated.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &updated.Images)
	}

	lowLimit := 1
	if p.LowStockThreshold > 0 {
		lowLimit = p.LowStockThreshold
	} else if p.MinStock > 0 {
		lowLimit = p.MinStock
	}

	if stock != nil {
		_, err = tx.Exec(ctx, `UPDATE inventory SET quantity = $1, low_stock_threshold = $2, updated_at = NOW() WHERE product_id = $3`, *stock, lowLimit, updated.ID)
		if err != nil {
			r.logger.Error("failed to update inventory", zap.Error(err), zap.String("product_id", updated.ID))
			return nil, err
		}
		updated.Inventory = &model.Inventory{
			ProductID:         updated.ID,
			Quantity:          *stock,
			ReservedQuantity:  0,
			AvailableQuantity: *stock,
			LowStockThreshold: lowLimit,
		}
		updated.StockQuantity = *stock
		updated.MinStock = lowLimit
		updated.LowStockThreshold = lowLimit
	} else {
		_, _ = tx.Exec(ctx, `UPDATE inventory SET low_stock_threshold = $1, updated_at = NOW() WHERE product_id = $2`, lowLimit, updated.ID)
		var qty, reserved, lowStock int
		_ = tx.QueryRow(ctx, `SELECT COALESCE(quantity, 0), COALESCE(reserved_quantity, 0), COALESCE(low_stock_threshold, 1) FROM inventory WHERE product_id = $1`, updated.ID).Scan(&qty, &reserved, &lowStock)
		avail := qty - reserved
		if avail < 0 {
			avail = 0
		}
		updated.Inventory = &model.Inventory{
			ProductID:         updated.ID,
			Quantity:          qty,
			ReservedQuantity:  reserved,
			AvailableQuantity: avail,
			LowStockThreshold: lowStock,
		}
		updated.StockQuantity = avail
		updated.MinStock = lowStock
		updated.LowStockThreshold = lowStock
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit product update: %w", err)
	}

	computeProfit(updated)
	return updated, nil
}

func (r *ProductRepo) Delete(ctx context.Context, id, shopID string) error {
	query := `
		DELETE FROM products
		WHERE id = $1 AND shop_id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, id, shopID)
	if err != nil {
		r.logger.Error("failed to delete product", zap.Error(err), zap.String("product_id", id))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *ProductRepo) AdjustInventoryStock(ctx context.Context, shopID, productID string, adjustment int, threshold *int) (*model.Inventory, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND shop_id = $2)`, productID, shopID).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrProductNotFound
	}

	query := `
		UPDATE inventory
		SET quantity = GREATEST(0, quantity + $1),
		    low_stock_threshold = COALESCE($2, low_stock_threshold),
		    updated_at = NOW()
		WHERE product_id = $3
		RETURNING product_id, quantity, reserved_quantity, low_stock_threshold
	`
	inv := &model.Inventory{}
	err = r.db.QueryRow(ctx, query, adjustment, threshold, productID).Scan(
		&inv.ProductID,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
	)
	if err != nil {
		r.logger.Error("failed to adjust inventory", zap.Error(err), zap.String("product_id", productID))
		return nil, err
	}
	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	if inv.AvailableQuantity < 0 {
		inv.AvailableQuantity = 0
	}
	return inv, nil
}

func (r *ProductRepo) GetLowStockProducts(ctx context.Context, shopID string) ([]*dto.LowStockProduct, error) {
	query := `
		SELECT p.id, p.name, p.sku, p.price,
		       i.quantity, i.reserved_quantity, i.low_stock_threshold
		FROM products p
		JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true AND i.quantity <= i.low_stock_threshold
		ORDER BY i.quantity ASC, p.name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to get low stock products", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var items []*dto.LowStockProduct
	for rows.Next() {
		item := &dto.LowStockProduct{}
		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.SKU,
			&item.Price,
			&item.CurrentStock,
			&item.ReservedStock,
			&item.LowStockThreshold,
		)
		if err != nil {
			r.logger.Error("failed to scan low stock row", zap.Error(err))
			return nil, err
		}

		avail := item.CurrentStock - item.ReservedStock
		if avail < 0 {
			avail = 0
		}
		item.AvailableStock = avail

		suggested := (item.LowStockThreshold * 3) - item.CurrentStock
		if suggested < 10 {
			suggested = 10
		}
		item.SuggestedReorderQty = suggested

		items = append(items, item)
	}

	return items, nil
}

// GetMonthlyProfitAnalytics aggregates sales, revenue, cost and net profit for a shop in a specific month (combining POS bills and completed reservations).
func (r *ProductRepo) GetMonthlyProfitAnalytics(ctx context.Context, shopID string, year, month int) (*dto.MonthlyProfitResponse, error) {
	query := `
		WITH pos_sales AS (
			SELECT
				COUNT(b.id) AS sales_count,
				COALESCE(SUM(b.total_amount), 0.0) AS revenue,
				COALESCE(SUM(b.total_cost), 0.0) AS cost,
				COALESCE(SUM(items.qty), 0) AS items_count
			FROM pos_bills b
			LEFT JOIN (
				SELECT bill_id, SUM(quantity) AS qty
				FROM pos_bill_items
				GROUP BY bill_id
			) items ON items.bill_id = b.id
			WHERE b.shop_id = $1
			  AND EXTRACT(YEAR FROM b.created_at) = $2
			  AND EXTRACT(MONTH FROM b.created_at) = $3
		),
		res_sales AS (
			SELECT 
				COUNT(r.id) AS sales_count,
				COALESCE(SUM(r.quantity), 0) AS items_count,
				COALESCE(SUM(r.quantity * p.price), 0.0) AS revenue,
				COALESCE(SUM(r.quantity * COALESCE(p.cost_price, 0)), 0.0) AS cost
			FROM reservations r
			JOIN products p ON r.product_id = p.id
			WHERE r.shop_id = $1
			  AND r.status = 'completed'
			  AND EXTRACT(YEAR FROM r.completed_at) = $2
			  AND EXTRACT(MONTH FROM r.completed_at) = $3
		)
		SELECT
			COALESCE(p.sales_count, 0) + COALESCE(r.sales_count, 0),
			COALESCE(p.items_count, 0) + COALESCE(r.items_count, 0),
			COALESCE(p.revenue, 0.0) + COALESCE(r.revenue, 0.0),
			COALESCE(p.cost, 0.0) + COALESCE(r.cost, 0.0)
		FROM pos_sales p
		CROSS JOIN res_sales r;
	`
	var totalSales, totalItems int
	var totalRevenue, totalCost float64
	err := r.db.QueryRow(ctx, query, shopID, year, month).Scan(&totalSales, &totalItems, &totalRevenue, &totalCost)
	if err != nil {
		r.logger.Error("failed to get monthly profit analytics", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}

	netProfit := totalRevenue - totalCost
	var avgMargin float64
	if totalRevenue > 0 {
		avgMargin = math.Round((netProfit/totalRevenue)*1000) / 10
	}

	monthName := time.Month(month).String()

	return &dto.MonthlyProfitResponse{
		Month:               monthName,
		Year:                year,
		TotalCompletedSales: totalSales,
		TotalItemsSold:      totalItems,
		TotalRevenue:        totalRevenue,
		TotalCost:           totalCost,
		NetProfit:           netProfit,
		AverageMarginPct:    avgMargin,
	}, nil
}

// GetProductPerformanceMatrix categorizes products into Best, Worst, Old/Dead Stock, and New Arrivals.
func (r *ProductRepo) GetProductPerformanceMatrix(ctx context.Context, shopID string) (*dto.ProductMatrixResponse, error) {
	query := `
		SELECT 
			p.id, p.name, p.sku, p.price, COALESCE(p.cost_price, 0),
			COALESCE(i.quantity, 0),
			COALESCE(sold.sold_qty, 0) AS total_sold_qty,
			COALESCE(EXTRACT(DAY FROM (NOW() - p.created_at))::int, 0) AS days_in_stock
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN (
			SELECT product_id, SUM(quantity) AS sold_qty
			FROM reservations
			WHERE status = 'completed' AND shop_id = $1
			GROUP BY product_id
		) sold ON sold.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to get product performance matrix", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var allItems []*dto.ProductPerformanceItem
	for rows.Next() {
		item := &dto.ProductPerformanceItem{}
		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.SKU,
			&item.Price,
			&item.CostPrice,
			&item.CurrentStock,
			&item.TotalSoldQty,
			&item.DaysInStock,
		)
		if err != nil {
			r.logger.Error("failed to scan performance item", zap.Error(err))
			return nil, err
		}

		if item.CostPrice > 0 {
			item.UnitProfit = item.Price - item.CostPrice
			item.ProfitMarginPct = math.Round((item.UnitProfit/item.Price)*1000) / 10
		}
		item.TotalProfit = float64(item.TotalSoldQty) * item.UnitProfit

		allItems = append(allItems, item)
	}

	matrix := &dto.ProductMatrixResponse{
		BestProfitable:  []*dto.ProductPerformanceItem{},
		WorstProfitable: []*dto.ProductPerformanceItem{},
		OldDeadStock:    []*dto.ProductPerformanceItem{},
		NewArrivals:     []*dto.ProductPerformanceItem{},
	}

	// 1. Sort by total profit desc for BestProfitable (top 5)
	bestList := make([]*dto.ProductPerformanceItem, len(allItems))
	copy(bestList, allItems)
	sort.Slice(bestList, func(i, j int) bool {
		return bestList[i].TotalProfit > bestList[j].TotalProfit
	})
	if len(bestList) > 5 {
		bestList = bestList[:5]
	}
	matrix.BestProfitable = bestList

	// 2. Sort by margin asc for WorstProfitable (bottom 5 with lowest margin)
	worstList := make([]*dto.ProductPerformanceItem, len(allItems))
	copy(worstList, allItems)
	sort.Slice(worstList, func(i, j int) bool {
		return worstList[i].ProfitMarginPct < worstList[j].ProfitMarginPct
	})
	if len(worstList) > 5 {
		worstList = worstList[:5]
	}
	matrix.WorstProfitable = worstList

	// 3. Old / Dead Stock (days_in_stock >= 60 and total_sold_qty <= 1)
	for _, it := range allItems {
		if it.DaysInStock >= 60 && it.TotalSoldQty <= 1 {
			matrix.OldDeadStock = append(matrix.OldDeadStock, it)
		}
		if it.DaysInStock <= 30 {
			matrix.NewArrivals = append(matrix.NewArrivals, it)
		}
	}

	return matrix, nil
}

// FindNearbyProducts searches for in-stock products in shops within radiusKm from (lat, lng), sorted by distance.
func (r *ProductRepo) FindNearbyProducts(
	ctx context.Context,
	lat, lng, radiusKm float64,
	query, category string,
	openNowOnly bool,
	page, limit int,
) ([]*dto.NearbyProductItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	var whereClauses []string
	args := []interface{}{lat, lng, radiusKm}
	argIdx := 4

	whereClauses = append(whereClauses, "p.is_active = true")
	whereClauses = append(whereClauses, "s.is_active = true")
	whereClauses = append(whereClauses, "s.status = 'active'")
	whereClauses = append(whereClauses, "s.latitude IS NOT NULL AND s.longitude IS NOT NULL")
	whereClauses = append(whereClauses, "(i.quantity - i.reserved_quantity) > 0")

	// Haversine distance condition in km
	distanceFormula := `(6371 * acos(LEAST(1.0, GREATEST(-1.0,
		cos(radians($1)) * cos(radians(s.latitude)) * cos(radians(s.longitude) - radians($2)) +
		sin(radians($1)) * sin(radians(s.latitude))
	))))`

	whereClauses = append(whereClauses, fmt.Sprintf("%s <= $3", distanceFormula))

	if openNowOnly {
		whereClauses = append(whereClauses, "s.is_open = true")
	}

	trimmedQ := strings.TrimSpace(query)
	if trimmedQ != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR p.description ILIKE $%d OR p.sku ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+trimmedQ+"%")
		argIdx++
	}

	trimmedCat := strings.TrimSpace(category)
	if trimmedCat != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.category ILIKE $%d", argIdx))
		args = append(args, "%"+trimmedCat+"%")
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	sqlQuery := fmt.Sprintf(`
		SELECT
			p.id, p.name, p.slug, p.price, p.images, (i.quantity - i.reserved_quantity) AS avail_qty,
			s.id AS shop_id, s.name AS shop_name, s.slug AS shop_slug, s.address AS shop_address, s.phone AS shop_phone,
			s.is_open, s.opening_time, s.closing_time, s.weekly_off,
			%s AS distance_km,
			COUNT(*) OVER() AS total_count
		FROM products p
		JOIN shops s ON p.shop_id = s.id
		JOIN inventory i ON p.id = i.product_id
		WHERE %s
		ORDER BY distance_km ASC, p.price ASC
		LIMIT $%d OFFSET $%d
	`, distanceFormula, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		r.logger.Error("failed to find nearby products", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var items []*dto.NearbyProductItem
	totalCount := 0

	for rows.Next() {
		item := &dto.NearbyProductItem{}
		var imagesBytes []byte
		var openTime, closeTime, weeklyOff string

		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.Slug,
			&item.Price,
			&imagesBytes,
			&item.AvailableQuantity,
			&item.ShopID,
			&item.ShopName,
			&item.ShopSlug,
			&item.ShopAddress,
			&item.ShopPhone,
			&item.IsOpen,
			&openTime,
			&closeTime,
			&weeklyOff,
			&item.DistanceKm,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan nearby product row", zap.Error(err))
			return nil, 0, err
		}

		item.Images = []string{}
		if len(imagesBytes) > 0 {
			_ = json.Unmarshal(imagesBytes, &item.Images)
		}

		dummyShop := &model.Shop{
			IsOpen:      item.IsOpen,
			IsActive:    true,
			OpeningTime: openTime,
			ClosingTime: closeTime,
			WeeklyOff:   weeklyOff,
		}
		computeShopOpenStatus(dummyShop)
		item.IsCurrentlyOpen = dummyShop.IsCurrentlyOpen

		item.DistanceKm = math.Round(item.DistanceKm*100) / 100
		items = append(items, item)
	}

	return items, totalCount, nil
}

// CreateStockAlert registers a customer's request to be alerted when an out-of-stock product is restocked.
func (r *ProductRepo) CreateStockAlert(ctx context.Context, productID, shopID, phone, name string) error {
	query := `
		INSERT INTO product_stock_alerts (product_id, shop_id, customer_phone, customer_name, notified, created_at)
		VALUES ($1, $2, $3, $4, false, NOW())
		ON CONFLICT (product_id, customer_phone) DO UPDATE 
		SET customer_name = EXCLUDED.customer_name, notified = false, created_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, productID, shopID, strings.TrimSpace(phone), strings.TrimSpace(name))
	if err != nil {
		r.logger.Error("failed to create product stock alert", zap.Error(err), zap.String("product_id", productID))
		return err
	}
	return nil
}

// GetDemandWatchlist aggregates unnotified customer interest for out-of-stock and low-stock items.
func (r *ProductRepo) GetDemandWatchlist(ctx context.Context, shopID string) ([]*dto.DemandWatchlistItem, error) {
	query := `
		SELECT 
			p.id, p.name, p.sku, 
			(i.quantity - i.reserved_quantity) AS current_stock,
			COUNT(a.id) AS waiting_customers_count
		FROM product_stock_alerts a
		JOIN products p ON a.product_id = p.id
		JOIN inventory i ON p.id = i.product_id
		WHERE a.shop_id = $1 AND a.notified = false
		GROUP BY p.id, p.name, p.sku, i.quantity, i.reserved_quantity
		ORDER BY waiting_customers_count DESC, p.name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to query demand watchlist", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*dto.DemandWatchlistItem
	for rows.Next() {
		var item dto.DemandWatchlistItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.SKU, &item.CurrentStock, &item.WaitingCustomersCount); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	if list == nil {
		list = []*dto.DemandWatchlistItem{}
	}
	return list, nil
}

// CountWaitingCustomers returns the number of customers awaiting a restock of the product.
func (r *ProductRepo) CountWaitingCustomers(ctx context.Context, productID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM product_stock_alerts WHERE product_id = $1 AND notified = false`, productID).Scan(&count)
	return count, err
}

// CreateBargainDeal saves a locked customer offer or accepted counter-offer.
func (r *ProductRepo) CreateBargainDeal(ctx context.Context, deal *model.ProductBargainDeal) error {
	query := `
		INSERT INTO product_bargain_deals (
			product_id, shop_id, customer_phone, customer_name, deal_code,
			offered_price, agreed_price, bundle_quantity, status, expires_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(
		ctx, query,
		deal.ProductID, deal.ShopID, strings.TrimSpace(deal.CustomerPhone), strings.TrimSpace(deal.CustomerName),
		deal.DealCode, deal.OfferedPrice, deal.AgreedPrice, deal.BundleQuantity, deal.Status, deal.ExpiresAt,
	).Scan(&deal.ID, &deal.CreatedAt)
}

// GetValidBargainDeal fetches an active unexpired bargain deal for checkout redemption.
func (r *ProductRepo) GetValidBargainDeal(ctx context.Context, productID, dealCode string) (*model.ProductBargainDeal, error) {
	query := `
		SELECT id, product_id, shop_id, customer_phone, COALESCE(customer_name, ''),
		       deal_code, offered_price, agreed_price, bundle_quantity, status, expires_at, created_at
		FROM product_bargain_deals
		WHERE product_id = $1 AND UPPER(deal_code) = UPPER($2) AND status = 'accepted' AND expires_at > NOW()
		LIMIT 1
	`
	var deal model.ProductBargainDeal
	err := r.db.QueryRow(ctx, query, productID, strings.TrimSpace(dealCode)).Scan(
		&deal.ID, &deal.ProductID, &deal.ShopID, &deal.CustomerPhone, &deal.CustomerName,
		&deal.DealCode, &deal.OfferedPrice, &deal.AgreedPrice, &deal.BundleQuantity, &deal.Status,
		&deal.ExpiresAt, &deal.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("bargain deal not found or expired")
		}
		return nil, err
	}
	return &deal, nil
}

// RedeemBargainDeal marks a deal code as redeemed upon POS sale completion.
func (r *ProductRepo) RedeemBargainDeal(ctx context.Context, dealID string) error {
	_, err := r.db.Exec(ctx, `UPDATE product_bargain_deals SET status = 'redeemed' WHERE id = $1`, dealID)
	return err
}

// FindActiveProductsByTokens searches active store products by a set of keyword tokens for Parchi parsing.
func (r *ProductRepo) FindActiveProductsByTokens(ctx context.Context, shopID string, tokens []string, limit int) ([]*model.Product, error) {
	if len(tokens) == 0 {
		return []*model.Product{}, nil
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	conditions := make([]string, len(tokens))
	args := []interface{}{shopID}
	for i, t := range tokens {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(t))+"%")
		conditions[i] = fmt.Sprintf("(LOWER(p.name) LIKE $%d OR LOWER(COALESCE(p.sku, '')) LIKE $%d)", len(args), len(args))
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.name, p.sku, p.price, COALESCE(p.cost_price, 0),
		       COALESCE(i.quantity, 0) - COALESCE(i.reserved_quantity, 0) AS available_stock
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true AND (%s)
		ORDER BY available_stock DESC, p.name ASC
		LIMIT %d
	`, strings.Join(conditions, " OR "), limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		var p model.Product
		var availStock int
		if err := rows.Scan(&p.ID, &p.ShopID, &p.Name, &p.SKU, &p.Price, &p.CostPrice, &availStock); err != nil {
			return nil, err
		}
		p.Inventory = &model.Inventory{AvailableQuantity: availStock, Quantity: availStock}
		products = append(products, &p)
	}
	return products, nil
}



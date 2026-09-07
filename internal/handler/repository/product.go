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

	query := `
		INSERT INTO products (shop_id, name, slug, description, sku, price, cost_price, compare_price, category_id, images, weight, is_active, is_featured, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0), category_id, images, COALESCE(weight, 0), is_active, is_featured, tags, created_at, updated_at
	`
	created := &model.Product{}
	var imagesBytes []byte

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

	// Insert into inventory
	invQuery := `
		INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
		VALUES ($1, $2, 0, 10, NOW())
	`
	if _, err := tx.Exec(ctx, invQuery, created.ID, initialStock); err != nil {
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
		LowStockThreshold: 10,
	}

	computeProfit(created)
	return created, nil
}

func (r *ProductRepo) FindByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 10)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1 AND p.is_active = true
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes []byte
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
		r.logger.Error("failed to find product by id", zap.Error(err), zap.String("product_id", id))
		return nil, err
	}

	p.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &p.Images)
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

func (r *ProductRepo) FindBySlug(ctx context.Context, slug string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 10)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.slug = LOWER(TRIM($1)) AND p.is_active = true
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes []byte
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
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 10),
		       COUNT(*) OVER() AS total_count
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
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
		var imagesBytes []byte
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
			&p.CreatedAt,
			&p.UpdatedAt,
			&inv.Quantity,
			&inv.ReservedQuantity,
			&inv.LowStockThreshold,
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

	query := `
		UPDATE products
		SET name = $1, slug = $2, description = $3, sku = $4, price = $5, cost_price = $6, compare_price = $7,
		    category_id = $8, images = $9, weight = $10, is_active = $11, is_featured = $12, tags = $13, updated_at = NOW()
		WHERE id = $14 AND shop_id = $15
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0), category_id, images, COALESCE(weight, 0), is_active, is_featured, tags, created_at, updated_at
	`
	updated := &model.Product{}
	var imagesBytes []byte

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

	if stock != nil {
		_, err = tx.Exec(ctx, `UPDATE inventory SET quantity = $1, updated_at = NOW() WHERE product_id = $2`, *stock, updated.ID)
		if err != nil {
			r.logger.Error("failed to update inventory", zap.Error(err), zap.String("product_id", updated.ID))
			return nil, err
		}
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

// GetMonthlyProfitAnalytics aggregates sales, revenue, cost and net profit for a shop in a specific month.
func (r *ProductRepo) GetMonthlyProfitAnalytics(ctx context.Context, shopID string, year, month int) (*dto.MonthlyProfitResponse, error) {
	query := `
		SELECT 
			COUNT(r.id) AS total_sales,
			COALESCE(SUM(r.quantity), 0) AS total_items,
			COALESCE(SUM(r.quantity * p.price), 0.0) AS total_revenue,
			COALESCE(SUM(r.quantity * COALESCE(p.cost_price, 0)), 0.0) AS total_cost
		FROM reservations r
		JOIN products p ON r.product_id = p.id
		WHERE r.shop_id = $1
		  AND r.status = 'completed'
		  AND EXTRACT(YEAR FROM r.completed_at) = $2
		  AND EXTRACT(MONTH FROM r.completed_at) = $3
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

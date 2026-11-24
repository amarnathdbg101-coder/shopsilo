package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"shopMe/internal/handler/model"
)

// MasterCatalogRepo handles database operations for the universal master product catalog.
type MasterCatalogRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewMasterCatalogRepo(db *pgxpool.Pool, logger *zap.Logger) *MasterCatalogRepo {
	return &MasterCatalogRepo{
		db:     db,
		logger: logger,
	}
}

// FindByBarcode performs an instant O(log n) indexed lookup on master_catalog_products.
func (r *MasterCatalogRepo) FindByBarcode(ctx context.Context, barcode string) (*model.MasterCatalogProduct, error) {
	cleanBarcode := strings.TrimSpace(strings.ToLower(barcode))
	if cleanBarcode == "" {
		return nil, errors.New("barcode cannot be empty")
	}

	query := `
		SELECT 
			id, barcode, name, brand, category, unit, mrp, selling_price,
			COALESCE(description, ''), COALESCE(image_url, ''),
			COALESCE(hsn_code, ''), gst_rate, source, trust_score, is_verified,
			scan_count, created_at, updated_at
		FROM master_catalog_products
		WHERE TRIM(LOWER(barcode)) = $1 AND deleted_at IS NULL
		LIMIT 1;
	`

	var p model.MasterCatalogProduct
	err := r.db.QueryRow(ctx, query, cleanBarcode).Scan(
		&p.ID, &p.Barcode, &p.Name, &p.Brand, &p.Category, &p.Unit, &p.MRP, &p.SellingPrice,
		&p.Description, &p.ImageURL, &p.HSNCode, &p.GSTRate, &p.Source, &p.TrustScore,
		&p.IsVerified, &p.ScanCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		r.logger.Error("master catalog barcode lookup query failed", zap.Error(err), zap.String("barcode", cleanBarcode))
		return nil, err
	}

	// Increment scan popularity count asynchronously
	go func(id string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = r.db.Exec(bgCtx, `UPDATE master_catalog_products SET scan_count = scan_count + 1 WHERE id = $1`, id)
	}(p.ID)

	return &p, nil
}

// FindShopProductByBarcode looks up whether the shop owner already has this product in active inventory.
func (r *MasterCatalogRepo) FindShopProductByBarcode(ctx context.Context, shopID, barcode string) (*model.Product, error) {
	cleanBarcode := strings.TrimSpace(strings.ToLower(barcode))
	if cleanBarcode == "" || shopID == "" {
		return nil, nil
	}

	query := `
		SELECT 
			p.id, p.shop_id, p.name, p.slug, p.description, COALESCE(p.sku, ''),
			p.price, COALESCE(p.cost_price, 0.0), COALESCE(p.compare_price, 0.0),
			COALESCE(p.weight, 0.0), p.images,
			p.tags, p.attributes, p.is_active, p.is_featured,
			COALESCE(i.quantity, 0), COALESCE(i.low_stock_threshold, 5),
			p.created_at, p.updated_at
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 
		  AND (
		      TRIM(LOWER(COALESCE(p.sku, ''))) = $2
		      OR TRIM(LOWER(COALESCE(p.attributes->>'barcode', ''))) = $2
		  )
		  AND p.is_active = true
		  AND p.deleted_at IS NULL
		LIMIT 1;
	`

	var p model.Product
	var rawImages []byte
	var rawAttrs []byte
	var invQuantity, invLowStock int

	err := r.db.QueryRow(ctx, query, shopID, cleanBarcode).Scan(
		&p.ID, &p.ShopID, &p.Name, &p.Slug, &p.Description, &p.SKU,
		&p.Price, &p.CostPrice, &p.ComparePrice, &p.Weight,
		&rawImages, &p.Tags, &rawAttrs, &p.IsActive, &p.IsFeatured,
		&invQuantity, &invLowStock, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	p.StockQuantity = invQuantity
	p.LowStockThreshold = invLowStock
	p.UnitProfit = p.Price - p.CostPrice
	if p.Price > 0 {
		p.ProfitMarginPct = (p.UnitProfit / p.Price) * 100
	}

	return &p, nil
}

// BulkUpsert executes high-speed batch inserts using PostgreSQL COPY or unnested parameterized batching.
func (r *MasterCatalogRepo) BulkUpsert(ctx context.Context, items []model.MasterCatalogProduct) (inserted int, updated int, err error) {
	if len(items) == 0 {
		return 0, 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Safe: no-op if committed

	query := `
		INSERT INTO master_catalog_products (
			barcode, name, brand, category, unit, mrp, selling_price,
			description, image_url, hsn_code, gst_rate, source, trust_score, is_verified, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW()
		)
		ON CONFLICT (TRIM(LOWER(barcode))) WHERE deleted_at IS NULL
		DO UPDATE SET
			name = EXCLUDED.name,
			brand = CASE WHEN EXCLUDED.brand <> '' THEN EXCLUDED.brand ELSE master_catalog_products.brand END,
			category = CASE WHEN EXCLUDED.category <> '' THEN EXCLUDED.category ELSE master_catalog_products.category END,
			unit = CASE WHEN EXCLUDED.unit <> '' THEN EXCLUDED.unit ELSE master_catalog_products.unit END,
			mrp = CASE WHEN EXCLUDED.mrp > 0 THEN EXCLUDED.mrp ELSE master_catalog_products.mrp END,
			selling_price = CASE WHEN EXCLUDED.selling_price > 0 THEN EXCLUDED.selling_price ELSE master_catalog_products.selling_price END,
			image_url = CASE WHEN EXCLUDED.image_url <> '' THEN EXCLUDED.image_url ELSE master_catalog_products.image_url END,
			hsn_code = CASE WHEN EXCLUDED.hsn_code <> '' THEN EXCLUDED.hsn_code ELSE master_catalog_products.hsn_code END,
			gst_rate = EXCLUDED.gst_rate,
			is_verified = EXCLUDED.is_verified,
			updated_at = NOW();
	`

	batch := &pgx.Batch{}
	for _, item := range items {
		cleanBarcode := strings.TrimSpace(item.Barcode)
		if cleanBarcode == "" || strings.TrimSpace(item.Name) == "" {
			continue
		}
		brand := strings.TrimSpace(item.Brand)
		cat := strings.TrimSpace(item.Category)
		if cat == "" {
			cat = "General Store"
		}
		unit := strings.TrimSpace(item.Unit)
		if unit == "" {
			unit = "piece"
		}

		batch.Queue(query,
			cleanBarcode, strings.TrimSpace(item.Name), brand, cat, unit,
			item.MRP, item.SellingPrice, strings.TrimSpace(item.Description),
			strings.TrimSpace(item.ImageURL), strings.TrimSpace(item.HSNCode),
			item.GSTRate, "admin_seed", 100, true,
		)
	}

	br := tx.SendBatch(ctx, batch)
	processed := 0
	for i := 0; i < batch.Len(); i++ {
		_, execErr := br.Exec()
		if execErr != nil {
			r.logger.Warn("batch item failed during master catalog seed", zap.Error(execErr), zap.Int("index", i))
		} else {
			processed++
		}
	}
	if closeErr := br.Close(); closeErr != nil {
		return 0, 0, fmt.Errorf("failed closing batch results: %w", closeErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return 0, 0, fmt.Errorf("failed committing seed transaction: %w", commitErr)
	}

	return processed, 0, nil
}

// HardDeleteProduct permanently deletes a product and its inventory row (Admin Only).
func (r *MasterCatalogRepo) HardDeleteProduct(ctx context.Context, productID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete from inventory first
	_, _ = tx.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, productID)

	// Delete from products
	res, err := tx.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("product not found")
	}

	return tx.Commit(ctx)
}

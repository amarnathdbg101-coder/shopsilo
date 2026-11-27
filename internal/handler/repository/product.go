// Package repository handle db query.
package repository

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// generateUUIDv4 generates an RFC 4122 version 4 UUID for ultra-fast bulk product & inventory ingestion.
func generateUUIDv4() string {
	var b [16]byte
	_, err := crand.Read(b[:])
	if err != nil {
		return fmt.Sprintf("%08x-%04x-4%03x-%04x-%012x",
			time.Now().UnixNano()&0xffffffff,
			rand.Int31n(0xffff),
			rand.Int31n(0x0fff),
			0x8000|(rand.Int31n(0x3fff)),
			rand.Int63n(0xffffffffffff),
		)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	)
}

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrProductSlugTaken = errors.New("product slug is already in use")
	ErrSKUTaken         = errors.New("product SKU is already in use")
)

type searchCacheEntry struct {
	products   []*model.Product
	totalCount int
	expiresAt  time.Time
}

type nearbySearchCacheEntry struct {
	items      []*dto.NearbyProductItem
	totalCount int
	expiresAt  time.Time
}

type suggestionCacheEntry struct {
	suggestions []string
	expiresAt   time.Time
}

type ProductRepo struct {
	db              *pgxpool.Pool
	logger          *zap.Logger
	searchMu        sync.RWMutex
	searchCache     map[string]searchCacheEntry
	nearbyCache     map[string]nearbySearchCacheEntry
	suggestionCache map[string]suggestionCacheEntry
}

func NewProductRepo(db *pgxpool.Pool, logger *zap.Logger) *ProductRepo {
	return &ProductRepo{
		db:              db,
		logger:          logger,
		searchCache:     make(map[string]searchCacheEntry),
		nearbyCache:     make(map[string]nearbySearchCacheEntry),
		suggestionCache: make(map[string]suggestionCacheEntry),
	}
}

// InvalidateSearchCache flushes in-memory search caches upon catalog mutation.
func (r *ProductRepo) InvalidateSearchCache() {
	r.searchMu.Lock()
	defer r.searchMu.Unlock()
	r.searchCache = make(map[string]searchCacheEntry)
	r.nearbyCache = make(map[string]nearbySearchCacheEntry)
	r.suggestionCache = make(map[string]suggestionCacheEntry)
}

func (r *ProductRepo) getCachedSearch(key string) ([]*model.Product, int, bool) {
	r.searchMu.RLock()
	defer r.searchMu.RUnlock()

	if entry, ok := r.searchCache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.products, entry.totalCount, true
		}
	}
	return nil, 0, false
}

func (r *ProductRepo) setCachedSearch(key string, products []*model.Product, totalCount int) {
	r.searchMu.Lock()
	defer r.searchMu.Unlock()

	if len(r.searchCache) >= 500 {
		now := time.Now()
		for k, v := range r.searchCache {
			if now.After(v.expiresAt) {
				delete(r.searchCache, k)
			}
		}
		if len(r.searchCache) >= 450 {
			r.searchCache = make(map[string]searchCacheEntry)
		}
	}

	r.searchCache[key] = searchCacheEntry{
		products:   products,
		totalCount: totalCount,
		expiresAt:  time.Now().Add(60 * time.Second),
	}
}

func (r *ProductRepo) getCachedNearbySearch(key string) ([]*dto.NearbyProductItem, int, bool) {
	r.searchMu.RLock()
	defer r.searchMu.RUnlock()

	if entry, ok := r.nearbyCache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.items, entry.totalCount, true
		}
	}
	return nil, 0, false
}

func (r *ProductRepo) setCachedNearbySearch(key string, items []*dto.NearbyProductItem, totalCount int) {
	r.searchMu.Lock()
	defer r.searchMu.Unlock()

	if len(r.nearbyCache) >= 500 {
		now := time.Now()
		for k, v := range r.nearbyCache {
			if now.After(v.expiresAt) {
				delete(r.nearbyCache, k)
			}
		}
		if len(r.nearbyCache) >= 450 {
			r.nearbyCache = make(map[string]nearbySearchCacheEntry)
		}
	}

	r.nearbyCache[key] = nearbySearchCacheEntry{
		items:      items,
		totalCount: totalCount,
		expiresAt:  time.Now().Add(45 * time.Second),
	}
}

func (r *ProductRepo) getCachedSuggestions(key string) ([]string, bool) {
	r.searchMu.RLock()
	defer r.searchMu.RUnlock()

	if entry, ok := r.suggestionCache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.suggestions, true
		}
	}
	return nil, false
}

func (r *ProductRepo) setCachedSuggestions(key string, suggestions []string) {
	r.searchMu.Lock()
	defer r.searchMu.Unlock()

	if len(r.suggestionCache) >= 2000 {
		now := time.Now()
		for k, v := range r.suggestionCache {
			if now.After(v.expiresAt) {
				delete(r.suggestionCache, k)
			}
		}
		if len(r.suggestionCache) >= 1800 {
			r.suggestionCache = make(map[string]suggestionCacheEntry)
		}
	}

	r.suggestionCache[key] = suggestionCacheEntry{
		suggestions: suggestions,
		expiresAt:   time.Now().Add(5 * time.Minute),
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

	if p.Tags == nil {
		p.Tags = []string{}
	}

	var catID *string
	trimmedCat := strings.TrimSpace(p.CategoryID)
	if trimmedCat != "" {
		if len(trimmedCat) == 36 && strings.Count(trimmedCat, "-") == 4 {
			catID = &trimmedCat
		} else {
			lookupTerm := trimmedCat
			if canonSlug, ok := utils.ResolveCanonicalCategorySlug(trimmedCat); ok {
				lookupTerm = canonSlug
			}
			var resolvedUUID string
			_ = tx.QueryRow(ctx, `SELECT id::text FROM categories WHERE (slug ILIKE $1 OR name ILIKE $1) AND (shop_id = $2 OR shop_id IS NULL) ORDER BY (shop_id IS NOT NULL) DESC LIMIT 1`, lookupTerm, p.ShopID).Scan(&resolvedUUID)
			if resolvedUUID != "" {
				catID = &resolvedUUID
			}
		}
	}

	query := `
		INSERT INTO products (
			shop_id, name, slug, description, sku, price, cost_price, compare_price,
			category_id, images, weight, is_active, is_featured, tags, attributes, is_price_public
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, $15::jsonb, $16
		)
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0),
		          COALESCE(category_id::text, ''), images, COALESCE(weight, 0), is_active, is_featured, COALESCE(tags, '{}'), COALESCE(attributes, '{}'::jsonb), created_at, updated_at,
		          COALESCE(is_price_public, true)
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
		catID,
		string(imagesJSON),
		p.Weight,
		p.IsActive,
		p.IsFeatured,
		p.Tags,
		string(attributesJSON),
		p.IsPricePublic,
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
		&created.IsPricePublic,
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

	invQuery := `
		INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold)
		VALUES ($1, $2, 0, $3)
		RETURNING product_id, quantity, reserved_quantity, low_stock_threshold
	`

	lowThreshold := 1
	if p.LowStockThreshold > 0 {
		lowThreshold = p.LowStockThreshold
	} else if p.MinStock > 0 {
		lowThreshold = p.MinStock
	}

	inv := &model.Inventory{}
	err = tx.QueryRow(ctx, invQuery, created.ID, initialStock, lowThreshold).Scan(
		&inv.ProductID,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
	)
	if err != nil {
		r.logger.Error("failed to create inventory for product", zap.Error(err), zap.String("product_id", created.ID))
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit product creation: %w", err)
	}

	r.InvalidateSearchCache()

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	created.Inventory = inv
	created.StockQuantity = inv.AvailableQuantity
	created.MinStock = inv.LowStockThreshold
	created.LowStockThreshold = inv.LowStockThreshold

	computeProfit(created)
	return created, nil
}

// SyncToMasterCatalog ensures any newly added merchant product is learned by the universal Master Catalog.
func (r *ProductRepo) SyncToMasterCatalog(ctx context.Context, p *model.Product, barcode, brand, unit, hsn, imgUrl string) error {
	cleanBarcode := strings.TrimSpace(barcode)
	if cleanBarcode == "" || p == nil {
		return nil
	}
	if brand == "" {
		brand = "General"
	}
	if unit == "" {
		unit = "pcs"
	}

	query := `
		INSERT INTO master_catalog_products (
			barcode, name, brand, category, unit, mrp, selling_price,
			description, image_url, hsn_code, gst_rate, source, trust_score, is_verified, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'General Store', $4, $5, $6, $7, $8, $9, 5, 'merchant_organic', 90, true, NOW(), NOW()
		)
		ON CONFLICT (barcode) DO UPDATE SET
			name = CASE WHEN master_catalog_products.name = '' THEN EXCLUDED.name ELSE master_catalog_products.name END,
			image_url = CASE WHEN master_catalog_products.image_url = '' THEN EXCLUDED.image_url ELSE master_catalog_products.image_url END,
			mrp = CASE WHEN master_catalog_products.mrp <= 0 THEN EXCLUDED.mrp ELSE master_catalog_products.mrp END,
			updated_at = NOW();
	`
	mrp := p.ComparePrice
	if mrp <= 0 {
		mrp = p.Price
	}

	_, err := r.db.Exec(ctx, query,
		cleanBarcode, p.Name, brand, unit, mrp, p.Price,
		p.Description, imgUrl, hsn,
	)
	return err
}

func (r *ProductRepo) FindByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       COALESCE(p.category_id::text, ''), COALESCE(p.images, '[]'::jsonb), COALESCE(p.weight, 0), p.is_active, p.is_featured, COALESCE(p.tags, '{}'), COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(s.phone, ''), COALESCE(s.address, ''), COALESCE(s.city, ''),
		       s.latitude, s.longitude, COALESCE(c.name, ''),
		       COALESCE(p.is_price_public, true)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN shops s ON s.id = p.shop_id
		LEFT JOIN categories c ON c.id = p.category_id
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
		&p.ShopName,
		&p.ShopSlug,
		&p.ShopPhone,
		&p.ShopAddress,
		&p.ShopCity,
		&p.ShopLatitude,
		&p.ShopLongitude,
		&p.CategoryName,
		&p.IsPricePublic,
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
	if p.CategoryName == "" && p.Attributes != nil {
		if cName, ok := p.Attributes["category_name"].(string); ok && cName != "" {
			p.CategoryName = cName
		} else if cName, ok := p.Attributes["category"].(string); ok && cName != "" {
			p.CategoryName = cName
		}
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

func (r *ProductRepo) FindByIDs(ctx context.Context, shopID string, ids []string) (map[string]*model.Product, error) {
	if len(ids) == 0 {
		return make(map[string]*model.Product), nil
	}

	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       COALESCE(p.category_id::text, ''), COALESCE(p.images, '[]'::jsonb), COALESCE(p.weight, 0), p.is_active, p.is_featured, COALESCE(p.tags, '{}'), COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(p.is_price_public, true)
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
			&p.IsPricePublic,
		)
		if err != nil {
			r.logger.Error("failed to scan product row", zap.Error(err))
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
		productsMap[p.ID] = p
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating product rows", zap.Error(err))
		return nil, err
	}

	return productsMap, nil
}

func (r *ProductRepo) FindBySlug(ctx context.Context, slug string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       COALESCE(p.category_id::text, ''), COALESCE(p.images, '[]'::jsonb), COALESCE(p.weight, 0), p.is_active, p.is_featured, COALESCE(p.tags, '{}'), COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(s.phone, ''), COALESCE(s.address, ''), COALESCE(s.city, ''),
		       s.latitude, s.longitude, COALESCE(c.name, ''),
		       COALESCE(p.is_price_public, true)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN shops s ON s.id = p.shop_id
		LEFT JOIN categories c ON c.id = p.category_id
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
		&p.ShopName,
		&p.ShopSlug,
		&p.ShopPhone,
		&p.ShopAddress,
		&p.ShopCity,
		&p.ShopLatitude,
		&p.ShopLongitude,
		&p.CategoryName,
		&p.IsPricePublic,
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
	if p.CategoryName == "" && p.Attributes != nil {
		if cName, ok := p.Attributes["category_name"].(string); ok && cName != "" {
			p.CategoryName = cName
		} else if cName, ok := p.Attributes["category"].(string); ok && cName != "" {
			p.CategoryName = cName
		}
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	computeProfit(p)
	return p, nil
}

func (r *ProductRepo) FindBySKU(ctx context.Context, shopID, sku string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       COALESCE(p.category_id::text, ''), COALESCE(p.images, '[]'::jsonb), COALESCE(p.weight, 0), p.is_active, p.is_featured, COALESCE(p.tags, '{}'), COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(p.is_price_public, true)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND (
			UPPER(TRIM(p.sku)) = UPPER(TRIM($2))
			OR p.attributes->>'barcode' = TRIM($2)
			OR p.attributes->>'Barcode' = TRIM($2)
			OR p.attributes->>'sku' = TRIM($2)
			OR p.id::text = TRIM($2)
			OR p.attributes::text ILIKE '%' || TRIM($2) || '%'
		)
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
		&p.IsPricePublic,
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
	cleanedSearch := utils.CleanSearchQuery(filter.Search)
	latStr, lngStr, radStr := "none", "none", "none"
	if filter.Lat != nil && filter.Lng != nil {
		latStr = fmt.Sprintf("%.3f", *filter.Lat)
		lngStr = fmt.Sprintf("%.3f", *filter.Lng)
		radVal := 10.0
		if filter.RadiusKm != nil && *filter.RadiusKm > 0 {
			radVal = *filter.RadiusKm
		}
		radStr = fmt.Sprintf("%.1f", radVal)
	}
	featStr := "all"
	if filter.IsFeatured != nil {
		featStr = fmt.Sprintf("%t", *filter.IsFeatured)
	}
	cacheKey := fmt.Sprintf("p_srch:%s:s:%s:c:%s:bc:%s:sz:%s:col:%s:sb:%s:p:%d:l:%d:min:%.0f:max:%.0f:feat:%s:lat:%s:lng:%s:rad:%s:city:%s",
		cleanedSearch, filter.ShopID, filter.CategoryID, strings.TrimSpace(filter.Barcode),
		strings.ToLower(strings.TrimSpace(filter.Size)), strings.ToLower(strings.TrimSpace(filter.Color)),
		filter.SortBy, filter.Page, filter.Limit, filter.MinPrice, filter.MaxPrice, featStr,
		latStr, lngStr, radStr, strings.ToLower(strings.TrimSpace(filter.City)))

	if cachedProds, cachedCount, found := r.getCachedSearch(cacheKey); found {
		return cachedProds, cachedCount, nil
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	} else if limit > 2000 {
		limit = 2000
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

	if filter.Barcode != "" {
		cleanBarcode := strings.TrimSpace(filter.Barcode)
		whereClauses = append(whereClauses, fmt.Sprintf("(p.attributes->>'barcode' = $%d OR UPPER(TRIM(COALESCE(p.sku, ''))) = UPPER($%d))", argIdx, argIdx))
		args = append(args, cleanBarcode)
		argIdx++
	}

	if filter.CategoryID != "" {
		catTerm := strings.TrimSpace(filter.CategoryID)

		var catID, catName, catSlug string
		_ = r.db.QueryRow(ctx, `SELECT COALESCE(id::text, ''), COALESCE(name, ''), COALESCE(slug, '') FROM categories WHERE id::text = $1 OR slug ILIKE $1 OR name ILIKE $1 LIMIT 1`, catTerm).Scan(&catID, &catName, &catSlug)

		targetID := catTerm
		if catID != "" {
			targetID = catID
		}

		var catOrs []string
		// Direct category ID match
		catOrs = append(catOrs, fmt.Sprintf("p.category_id::text = $%d", argIdx))
		args = append(args, targetID)
		argIdx++

		if catSlug != "" {
			catOrs = append(catOrs, fmt.Sprintf("(p.category_id::text = $%d OR p.category_id IN (SELECT id FROM categories WHERE slug ILIKE $%d))", argIdx, argIdx))
			args = append(args, catSlug)
			argIdx++
		}

		// Canonical slug and name synonym expansion
		synonymSlugs := utils.GetCategorySynonymSlugs(catTerm)
		synonymNames := utils.GetCategorySynonymNames(catTerm)

		if len(synonymSlugs) > 0 {
			catOrs = append(catOrs, fmt.Sprintf("p.category_id IN (SELECT id FROM categories WHERE slug = ANY($%d))", argIdx))
			args = append(args, synonymSlugs)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category_name', '')) = ANY($%d)", argIdx))
			args = append(args, synonymSlugs)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category', '')) = ANY($%d)", argIdx))
			args = append(args, synonymSlugs)
			argIdx++
		}

		if len(synonymNames) > 0 {
			var lowerNames []string
			for _, n := range synonymNames {
				lowerNames = append(lowerNames, strings.ToLower(n))
			}
			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category_name', '')) = ANY($%d)", argIdx))
			args = append(args, lowerNames)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category', '')) = ANY($%d)", argIdx))
			args = append(args, lowerNames)
			argIdx++
		}

		// Tokenized and Hinglish keyword fallback for unassigned or legacy products
		keywordSource := catName
		if keywordSource == "" {
			keywordSource = catSlug
		}
		if keywordSource == "" {
			keywordSource = catTerm
		}

		// Replace & with space
		cleanSource := strings.ReplaceAll(keywordSource, "&", " ")
		cleanSource = strings.ReplaceAll(cleanSource, "-", " ")
		groups := utils.ExpandHinglishSearchGroups(cleanSource)
		for _, g := range groups {
			for _, term := range g.Terms {
				term = strings.TrimSpace(term)
				if len(term) < 3 || term == "and" || term == "the" || term == "care" || term == "food" {
					continue
				}
				wild := "%" + term + "%"
				catOrs = append(catOrs, fmt.Sprintf("(p.name ILIKE $%d OR p.description ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d)", argIdx, argIdx, argIdx))
				args = append(args, wild)
				argIdx++
			}
		}

		if len(catOrs) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(catOrs, " OR ")+")")
		}
	}

	if filter.Search != "" {
		cleanSearchDigits := strings.TrimSpace(filter.Search)
		if len(cleanSearchDigits) >= 8 && len(cleanSearchDigits) <= 14 && isDigitsOnly(cleanSearchDigits) {
			whereClauses = append(whereClauses, fmt.Sprintf("(p.attributes->>'barcode' = $%d OR UPPER(TRIM(COALESCE(p.sku, ''))) = UPPER($%d) OR p.name ILIKE $%d)", argIdx, argIdx, argIdx+1))
			args = append(args, cleanSearchDigits, "%"+cleanSearchDigits+"%")
			argIdx += 2
		} else {
		groups := utils.ExpandHinglishSearchGroups(filter.Search)
		if len(groups) > 0 {
			for _, group := range groups {
				var groupOrs []string
				for _, term := range group.Terms {
					cleanTerm := strings.TrimSpace(term)
					if cleanTerm == "" {
						continue
					}
					wild := "%" + cleanTerm + "%"
					if len(cleanTerm) >= 3 {
						groupOrs = append(groupOrs, fmt.Sprintf("(p.name ILIKE $%d OR word_similarity($%d, p.name) >= 0.35 OR p.name %% $%d OR COALESCE(p.sku, '') ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR p.slug ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx+1, argIdx+1, argIdx, argIdx, argIdx, argIdx, argIdx))
						args = append(args, wild, cleanTerm)
						argIdx += 2
					} else {
						groupOrs = append(groupOrs, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR p.slug ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
						args = append(args, wild)
						argIdx++
					}
				}
				if len(groupOrs) > 0 {
					whereClauses = append(whereClauses, "("+strings.Join(groupOrs, " OR ")+")")
				}
			}
		} else {
			cleanTerm := strings.TrimSpace(filter.Search)
			wild := "%" + cleanTerm + "%"
			if len(cleanTerm) >= 3 {
				whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR word_similarity($%d, p.name) >= 0.35 OR p.name %% $%d OR COALESCE(p.sku, '') ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR p.slug ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx+1, argIdx+1, argIdx, argIdx, argIdx, argIdx, argIdx))
				args = append(args, wild, cleanTerm)
				argIdx += 2
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR p.slug ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
				args = append(args, wild)
				argIdx++
			}
		}
		}
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

	// ðŸ‘— Smart Size-First Query Filter (Apparel & Footwear)
	if strings.TrimSpace(filter.Size) != "" {
		cleanSize := strings.TrimSpace(filter.Size)
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			EXISTS (
				SELECT 1 FROM product_variants pv 
				WHERE pv.product_id = p.id 
				  AND pv.is_active = true 
				  AND UPPER(TRIM(pv.size)) = UPPER(TRIM($%d))
			) OR UPPER(TRIM(COALESCE(p.attributes->>'size', ''))) = UPPER(TRIM($%d))
		)`, argIdx, argIdx))
		args = append(args, cleanSize)
		argIdx++
	}

	// ðŸ’„ Shade & Color Variant Filter (Cosmetics & Apparel)
	if strings.TrimSpace(filter.Color) != "" {
		cleanColor := strings.TrimSpace(filter.Color)
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			EXISTS (
				SELECT 1 FROM product_variants pv 
				WHERE pv.product_id = p.id 
				  AND pv.is_active = true 
				  AND (UPPER(TRIM(pv.color)) = UPPER(TRIM($%d)) OR UPPER(TRIM(COALESCE(pv.shade_code, ''))) = UPPER(TRIM($%d)))
			) OR UPPER(TRIM(COALESCE(p.attributes->>'color', ''))) = UPPER(TRIM($%d))
		)`, argIdx, argIdx, argIdx))
		args = append(args, cleanColor)
		argIdx++
	}

	// Location & Distance Range Filter (strictly limits database scanning to shops within radius)
	hasGeoFilter := filter.ShopID == "" && filter.Lat != nil && filter.Lng != nil
	if hasGeoFilter {
		radius := 10.0
		if filter.RadiusKm != nil && *filter.RadiusKm > 0 {
			radius = *filter.RadiusKm
		}
		lat := *filter.Lat
		lng := *filter.Lng
		radiusMeters := radius * 1000.0

		// PostGIS Level 1 Spatial Architecture:
		// Leverages idx_shops_location_gist 2D R-Tree index for sub-millisecond filtering
		geoSubquery := fmt.Sprintf(`p.shop_id IN (
			SELECT id FROM shops
			WHERE is_active = true AND status = 'active'
			  AND location IS NOT NULL
			  AND ST_DWithin(location, ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography, %f)
		)`, lng, lat, radiusMeters)

		whereClauses = append(whereClauses, geoSubquery)
	} else if filter.ShopID == "" && strings.TrimSpace(filter.City) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.shop_id IN (SELECT id FROM shops WHERE is_active = true AND status = 'active' AND city ILIKE $%d)", argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.City)+"%")
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// 1. Get total count first (more efficient than COUNT(*) OVER() in PostgreSQL for large sets)
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM products p WHERE %s`, whereSQL)
	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		r.logger.Error("failed to count products", zap.Error(err))
		return nil, 0, err
	}

	if totalCount == 0 {
		if cacheKey != "" {
			r.setCachedSearch(cacheKey, []*model.Product{}, 0)
		}
		return []*model.Product{}, 0, nil
	}

	orderBy := "p.created_at DESC"
	var geoDistOrder string
	if hasGeoFilter {
		geoDistOrder = fmt.Sprintf("ST_Distance(s.location, ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography)", *filter.Lng, *filter.Lat)
		orderBy = fmt.Sprintf("%s ASC, p.created_at DESC", geoDistOrder)
	}
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
	default:
		if filter.Search != "" {
			cleanSearch := strings.TrimSpace(filter.Search)
			if len(cleanSearch) >= 3 {
				if hasGeoFilter {
					orderBy = fmt.Sprintf("GREATEST(word_similarity($%d, p.name), similarity(p.name, $%d)) DESC, %s ASC, p.created_at DESC", argIdx, argIdx, geoDistOrder)
				} else {
					orderBy = fmt.Sprintf("GREATEST(word_similarity($%d, p.name), similarity(p.name, $%d)) DESC, p.is_featured DESC, p.created_at DESC", argIdx, argIdx)
				}
				args = append(args, cleanSearch)
				argIdx++
			}
		}
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.compare_price, 0),
		       COALESCE(p.category_id::text, ''), COALESCE(p.images, '[]'::jsonb), COALESCE(p.weight, 0), p.is_active, p.is_featured, COALESCE(p.tags, '{}'), COALESCE(p.attributes, '{}'::jsonb), p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 1),
		       COALESCE(s.name, ''), COALESCE(s.slug, ''), COALESCE(s.phone, ''), COALESCE(s.address, ''), COALESCE(s.city, ''),
		       s.latitude, s.longitude, COALESCE(c.name, ''),
		       COALESCE(p.is_price_public, true)
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
			&p.IsPricePublic,
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

		if p.CategoryName == "" && p.Attributes != nil {
			if cName, ok := p.Attributes["category_name"].(string); ok && cName != "" {
				p.CategoryName = cName
			} else if cName, ok := p.Attributes["category"].(string); ok && cName != "" {
				p.CategoryName = cName
			}
		}

		inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
		p.Inventory = inv

		computeProfit(p)
		products = append(products, p)
	}

	if cacheKey != "" {
		r.setCachedSearch(cacheKey, products, totalCount)
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

	if p.Tags == nil {
		p.Tags = []string{}
	}

	var catID *string
	trimmedCat := strings.TrimSpace(p.CategoryID)
	if trimmedCat != "" {
		if len(trimmedCat) == 36 && strings.Count(trimmedCat, "-") == 4 {
			catID = &trimmedCat
		} else {
			lookupTerm := trimmedCat
			if canonSlug, ok := utils.ResolveCanonicalCategorySlug(trimmedCat); ok {
				lookupTerm = canonSlug
			}
			var resolvedUUID string
			_ = tx.QueryRow(ctx, `SELECT id::text FROM categories WHERE slug ILIKE $1 OR name ILIKE $1 LIMIT 1`, lookupTerm).Scan(&resolvedUUID)
			if resolvedUUID != "" {
				catID = &resolvedUUID
			}
		}
	}

	query := `
		UPDATE products
		SET name = $1, slug = $2, description = $3, sku = $4, price = $5, cost_price = $6, compare_price = $7,
		    category_id = $8, images = $9::jsonb, weight = $10, is_active = $11, is_featured = $12, tags = $13, attributes = $14::jsonb,
		    is_price_public = $15, updated_at = NOW()
		WHERE id = $16 AND shop_id = $17
		RETURNING id, shop_id, name, slug, COALESCE(description, ''), sku, price, COALESCE(cost_price, 0), COALESCE(compare_price, 0),
		          COALESCE(category_id::text, ''), images, COALESCE(weight, 0), is_active, is_featured, COALESCE(tags, '{}'), COALESCE(attributes, '{}'::jsonb), created_at, updated_at,
		          COALESCE(is_price_public, true)
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
		catID,
		string(imagesJSON),
		p.Weight,
		p.IsActive,
		p.IsFeatured,
		p.Tags,
		string(attributesJSON),
		p.IsPricePublic,
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
		&updated.IsPricePublic,
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

	updated.Attributes = make(map[string]interface{})
	if len(attributesBytes) > 0 {
		_ = json.Unmarshal(attributesBytes, &updated.Attributes)
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

	r.InvalidateSearchCache()

	computeProfit(updated)
	return updated, nil
}

func (r *ProductRepo) Delete(ctx context.Context, id, shopID string) error {
	// Soft delete product so past POS sales, receipts, and Khata bills remain intact!
	query := `
		UPDATE products
		SET is_active = false, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND shop_id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, id, shopID)
	if err != nil {
		r.logger.Error("failed to soft delete product", zap.Error(err), zap.String("product_id", id))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	r.InvalidateSearchCache()
	return nil
}

// BatchDelete soft deletes multiple products belonging to a shop in a single query.
func (r *ProductRepo) BatchDelete(ctx context.Context, shopID string, productIDs []string) (int64, error) {
	if len(productIDs) == 0 {
		return 0, nil
	}

	query := `
		UPDATE products
		SET is_active = false, deleted_at = NOW(), updated_at = NOW()
		WHERE shop_id = $1 AND id::text = ANY($2)
	`
	cmdTag, err := r.db.Exec(ctx, query, shopID, productIDs)
	if err != nil {
		r.logger.Error("failed to batch delete products", zap.Error(err), zap.String("shop_id", shopID))
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}

// BatchToggleStatus updates the active status for multiple products belonging to a shop in a single query.
func (r *ProductRepo) BatchToggleStatus(ctx context.Context, shopID string, productIDs []string, isActive bool) (int64, error) {
	if len(productIDs) == 0 {
		return 0, nil
	}

	query := `
		UPDATE products
		SET is_active = $1, updated_at = NOW()
		WHERE shop_id = $2 AND id::text = ANY($3)
	`
	cmdTag, err := r.db.Exec(ctx, query, isActive, shopID, productIDs)
	if err != nil {
		r.logger.Error("failed to batch toggle product status", zap.Error(err), zap.String("shop_id", shopID))
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
}


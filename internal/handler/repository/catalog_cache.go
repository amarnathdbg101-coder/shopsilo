package repository

import (
	"context"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
)

// CatalogMemoryCache provides sub-millisecond RAM lookups for hot master catalog items.
// Thread-safe with sync.RWMutex and bounded capacity to prevent memory leaks.
type CatalogMemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*model.MasterCatalogProduct
	keys    []string // for FIFO/LRU-style bound enforcement
	maxSize int
	logger  *zap.Logger
}

// NewCatalogMemoryCache creates an in-memory cache with specified maximum capacity.
func NewCatalogMemoryCache(maxSize int, logger *zap.Logger) *CatalogMemoryCache {
	if maxSize <= 0 {
		maxSize = 10000 // Default 10,000 items (approx 5MB RAM)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CatalogMemoryCache{
		items:   make(map[string]*model.MasterCatalogProduct, maxSize),
		keys:    make([]string, 0, maxSize),
		maxSize: maxSize,
		logger:  logger,
	}
}

// Get returns the cached product if present for any of the canonical variants.
func (c *CatalogMemoryCache) Get(barcode string) (*model.MasterCatalogProduct, bool) {
	variants := utils.GenerateBarcodeVariants(barcode)
	if len(variants) == 0 {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, v := range variants {
		key := strings.ToLower(v)
		if item, ok := c.items[key]; ok && item != nil {
			return item, true
		}
	}

	return nil, false
}

// Set stores a product under all its canonical barcode variations.
func (c *CatalogMemoryCache) Set(barcode string, item *model.MasterCatalogProduct) {
	if item == nil {
		return
	}

	variants := utils.GenerateBarcodeVariants(barcode)
	if len(variants) == 0 {
		variants = []string{strings.ToLower(strings.TrimSpace(barcode))}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range variants {
		key := strings.ToLower(v)
		if _, exists := c.items[key]; !exists {
			if len(c.items) >= c.maxSize && len(c.keys) > 0 {
				// Evict oldest entry
				oldest := c.keys[0]
				c.keys = c.keys[1:]
				delete(c.items, oldest)
			}
			c.keys = append(c.keys, key)
		}
		c.items[key] = item
	}
}

// Invalidate removes a barcode and its variants from cache.
func (c *CatalogMemoryCache) Invalidate(barcode string) {
	variants := utils.GenerateBarcodeVariants(barcode)
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, v := range variants {
		delete(c.items, strings.ToLower(v))
	}
}

// Len returns current number of entries in cache.
func (c *CatalogMemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// WarmUp pre-loads the top most-scanned or verified products from database into RAM.
func (c *CatalogMemoryCache) WarmUp(ctx context.Context, db *pgxpool.Pool, limit int) (int, error) {
	if db == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 1000
	}

	query := `
		SELECT 
			id, barcode, name, brand, category, unit, mrp, selling_price,
			COALESCE(description, ''), COALESCE(image_url, ''),
			COALESCE(hsn_code, ''), gst_rate, source, trust_score, is_verified,
			scan_count, created_at, updated_at
		FROM master_catalog_products
		WHERE deleted_at IS NULL AND is_verified = true
		ORDER BY scan_count DESC, updated_at DESC
		LIMIT $1;
	`

	rows, err := db.Query(ctx, query, limit)
	if err != nil {
		c.logger.Warn("catalog memory cache warm-up query failed", zap.Error(err))
		return 0, err
	}
	defer rows.Close()

	loaded := 0
	for rows.Next() {
		var p model.MasterCatalogProduct
		err := rows.Scan(
			&p.ID, &p.Barcode, &p.Name, &p.Brand, &p.Category, &p.Unit, &p.MRP, &p.SellingPrice,
			&p.Description, &p.ImageURL, &p.HSNCode, &p.GSTRate, &p.Source, &p.TrustScore,
			&p.IsVerified, &p.ScanCount, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}
		c.Set(p.Barcode, &p)
		loaded++
	}

	c.logger.Info("master catalog memory cache warmed up",
		zap.Int("products_loaded", loaded),
		zap.Int("total_keys", c.Len()),
	)
	return loaded, nil
}

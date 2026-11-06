// Package repository handles database operations.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
)

var (
	ErrLocationNotFound        = errors.New("location not found")
	ErrLocationCodeTaken       = errors.New("location code is already in use for this shop")
	ErrVariantNotFound         = errors.New("product variant not found")
	ErrVariantSKUTaken         = errors.New("variant SKU is already in use for this shop")
	ErrTransferNotFound        = errors.New("stock transfer not found")
	ErrInvalidTransferState       = errors.New("invalid stock transfer state transition")
	ErrInsufficientLocationStock  = errors.New("insufficient stock in source location for transfer")
)

// LocationRepo handles multi-store godown locations, variant matrices, and stock movements.
type LocationRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewLocationRepo initializes the repository with the PostgreSQL connection pool.
func NewLocationRepo(db *pgxpool.Pool, logger *zap.Logger) *LocationRepo {
	return &LocationRepo{
		db:     db,
		logger: logger,
	}
}

// ==============================================================================
// 1. PHYSICAL LOCATIONS (SHOWROOM & GODOWNS)
// ==============================================================================

// ListLocationsByShop returns all active locations for a given merchant shop.
func (r *LocationRepo) ListLocationsByShop(ctx context.Context, shopID string) ([]*model.Location, error) {
	query := `
		SELECT id, shop_id, name, code, type, COALESCE(address, ''), 
		       COALESCE(contact_person, ''), COALESCE(contact_phone, ''), 
		       is_default, is_active, created_at, updated_at
		FROM locations
		WHERE shop_id = $1 AND is_active = true
		ORDER BY is_default DESC, name ASC;
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to query shop locations", zap.String("shop_id", shopID), zap.Error(err))
		return nil, fmt.Errorf("failed to query locations: %w", err)
	}
	defer rows.Close()

	var list []*model.Location
	for rows.Next() {
		var loc model.Location
		var locType string
		if err := rows.Scan(
			&loc.ID, &loc.ShopID, &loc.Name, &loc.Code, &locType,
			&loc.Address, &loc.ContactPerson, &loc.ContactPhone,
			&loc.IsDefault, &loc.IsActive, &loc.CreatedAt, &loc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		loc.Type = model.LocationType(locType)
		list = append(list, &loc)
	}
	return list, nil
}

// GetLocationByID fetches a specific location belonging to a merchant shop.
func (r *LocationRepo) GetLocationByID(ctx context.Context, shopID, locationID string) (*model.Location, error) {
	query := `
		SELECT id, shop_id, name, code, type, COALESCE(address, ''), 
		       COALESCE(contact_person, ''), COALESCE(contact_phone, ''), 
		       is_default, is_active, created_at, updated_at
		FROM locations
		WHERE shop_id = $1 AND id = $2;
	`
	var loc model.Location
	var locType string
	err := r.db.QueryRow(ctx, query, shopID, locationID).Scan(
		&loc.ID, &loc.ShopID, &loc.Name, &loc.Code, &locType,
		&loc.Address, &loc.ContactPerson, &loc.ContactPhone,
		&loc.IsDefault, &loc.IsActive, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("failed to fetch location: %w", err)
	}
	loc.Type = model.LocationType(locType)
	return &loc, nil
}

// GetDefaultLocation returns the primary retail counter location for a shop.
func (r *LocationRepo) GetDefaultLocation(ctx context.Context, shopID string) (*model.Location, error) {
	query := `
		SELECT id, shop_id, name, code, type, COALESCE(address, ''), 
		       COALESCE(contact_person, ''), COALESCE(contact_phone, ''), 
		       is_default, is_active, created_at, updated_at
		FROM locations
		WHERE shop_id = $1 AND is_default = true
		LIMIT 1;
	`
	var loc model.Location
	var locType string
	err := r.db.QueryRow(ctx, query, shopID).Scan(
		&loc.ID, &loc.ShopID, &loc.Name, &loc.Code, &locType,
		&loc.Address, &loc.ContactPerson, &loc.ContactPhone,
		&loc.IsDefault, &loc.IsActive, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("failed to get default location: %w", err)
	}
	loc.Type = model.LocationType(locType)
	return &loc, nil
}

// CreateLocation adds a new physical facility (Godown, Warehouse, Showroom).
func (r *LocationRepo) CreateLocation(ctx context.Context, shopID string, input dto.CreateLocationRequest) (*model.Location, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// If this new location is marked as default, unset existing default
	if input.IsDefault {
		_, err := tx.Exec(ctx, `UPDATE locations SET is_default = false WHERE shop_id = $1;`, shopID)
		if err != nil {
			return nil, fmt.Errorf("failed to reset default locations: %w", err)
		}
	}

	query := `
		INSERT INTO locations (shop_id, name, code, type, address, contact_person, contact_phone, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)
		RETURNING id, created_at, updated_at;
	`
	var loc model.Location
	loc.ShopID = shopID
	loc.Name = input.Name
	loc.Code = input.Code
	loc.Type = input.Type
	loc.Address = input.Address
	loc.ContactPerson = input.ContactPerson
	loc.ContactPhone = input.ContactPhone
	loc.IsDefault = input.IsDefault
	loc.IsActive = true

	err = tx.QueryRow(ctx, query,
		shopID, input.Name, input.Code, string(input.Type),
		input.Address, input.ContactPerson, input.ContactPhone, input.IsDefault,
	).Scan(&loc.ID, &loc.CreatedAt, &loc.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create location", zap.Error(err))
		return nil, fmt.Errorf("failed to insert location: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &loc, nil
}

// UpdateLocation edits details of an existing location.
func (r *LocationRepo) UpdateLocation(ctx context.Context, shopID, locationID string, input dto.UpdateLocationRequest) (*model.Location, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if input.IsDefault != nil && *input.IsDefault {
		_, err := tx.Exec(ctx, `UPDATE locations SET is_default = false WHERE shop_id = $1;`, shopID)
		if err != nil {
			return nil, fmt.Errorf("failed to unset previous default location: %w", err)
		}
	}

	query := `
		UPDATE locations
		SET name = COALESCE($3, name),
		    type = COALESCE($4, type),
		    address = COALESCE($5, address),
		    contact_person = COALESCE($6, contact_person),
		    contact_phone = COALESCE($7, contact_phone),
		    is_default = COALESCE($8, is_default),
		    is_active = COALESCE($9, is_active),
		    updated_at = NOW()
		WHERE shop_id = $1 AND id = $2
		RETURNING id, shop_id, name, code, type, COALESCE(address, ''), 
		          COALESCE(contact_person, ''), COALESCE(contact_phone, ''), 
		          is_default, is_active, created_at, updated_at;
	`
	var loc model.Location
	var locType string
	var typeVal *string
	if input.Type != nil {
		str := string(*input.Type)
		typeVal = &str
	}

	err = tx.QueryRow(ctx, query,
		shopID, locationID, input.Name, typeVal,
		input.Address, input.ContactPerson, input.ContactPhone,
		input.IsDefault, input.IsActive,
	).Scan(
		&loc.ID, &loc.ShopID, &loc.Name, &loc.Code, &locType,
		&loc.Address, &loc.ContactPerson, &loc.ContactPhone,
		&loc.IsDefault, &loc.IsActive, &loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("failed to update location: %w", err)
	}
	loc.Type = model.LocationType(locType)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit update: %w", err)
	}

	return &loc, nil
}

// ==============================================================================
// 2. PRODUCT VARIANTS (CLOTH SIZE/COLOR & BEAUTY SHADES)
// ==============================================================================

// CreateVariant inserts a single SKU variant for a parent product.
func (r *LocationRepo) CreateVariant(ctx context.Context, shopID, productID string, input dto.CreateVariantRequest) (*model.ProductVariant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO product_variants (
			product_id, shop_id, sku, barcode, title, size, color, color_hex,
			shade_code, batch_number, expiry_date, price, cost_price, compare_price,
			floor_price, weight, image_url, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, true
		) RETURNING id, created_at, updated_at;
	`
	var v model.ProductVariant
	v.ProductID = productID
	v.ShopID = shopID
	v.SKU = input.SKU
	v.Barcode = input.Barcode
	v.Title = input.Title
	v.Size = input.Size
	v.Color = input.Color
	v.ColorHex = input.ColorHex
	v.ShadeCode = input.ShadeCode
	v.BatchNumber = input.BatchNumber
	v.ExpiryDate = input.ExpiryDate
	v.Price = input.Price
	v.CostPrice = input.CostPrice
	v.ComparePrice = input.ComparePrice
	v.FloorPrice = input.FloorPrice
	v.Weight = input.Weight
	v.ImageURL = input.ImageURL
	v.IsActive = true

	err = tx.QueryRow(ctx, query,
		productID, shopID, input.SKU, input.Barcode, input.Title,
		input.Size, input.Color, input.ColorHex, input.ShadeCode,
		input.BatchNumber, input.ExpiryDate, input.Price, input.CostPrice,
		input.ComparePrice, input.FloorPrice, input.Weight, input.ImageURL,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert variant: %w", err)
	}

	// Seed initial stock in target or default location if quantity > 0
	if input.InitialStock > 0 {
		targetLocID := input.LocationID
		if targetLocID == "" {
			err := tx.QueryRow(ctx, `SELECT id FROM locations WHERE shop_id = $1 AND is_default = true LIMIT 1;`, shopID).Scan(&targetLocID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve default location for initial stock: %w", err)
			}
		}

		stockQuery := `
			INSERT INTO location_inventory (shop_id, location_id, product_id, variant_id, quantity, reserved_quantity, low_stock_threshold)
			VALUES ($1, $2, $3, $4, $5, 0, 5)
			ON CONFLICT (location_id, product_id, variant_id) WHERE variant_id IS NOT NULL
			DO UPDATE SET quantity = location_inventory.quantity + EXCLUDED.quantity, updated_at = NOW();
		`
		_, err = tx.Exec(ctx, stockQuery, shopID, targetLocID, productID, v.ID, input.InitialStock)
		if err != nil {
			return nil, fmt.Errorf("failed to seed initial variant stock: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit variant: %w", err)
	}

	return &v, nil
}

// BulkCreateVariants saves multiple variants in a single atomic transaction.
func (r *LocationRepo) BulkCreateVariants(ctx context.Context, shopID, productID string, variants []dto.CreateVariantRequest, initialStockEach int, targetLocID string) ([]*model.ProductVariant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin bulk tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if targetLocID == "" && initialStockEach > 0 {
		err := tx.QueryRow(ctx, `SELECT id FROM locations WHERE shop_id = $1 AND is_default = true LIMIT 1;`, shopID).Scan(&targetLocID)
		if err != nil {
			return nil, fmt.Errorf("failed to find default location: %w", err)
		}
	}

	var results []*model.ProductVariant
	for _, input := range variants {
		query := `
			INSERT INTO product_variants (
				product_id, shop_id, sku, barcode, title, size, color, color_hex,
				shade_code, batch_number, expiry_date, price, cost_price, compare_price,
				floor_price, weight, image_url, is_active
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, true
			) RETURNING id, created_at, updated_at;
		`
		var v model.ProductVariant
		v.ProductID = productID
		v.ShopID = shopID
		v.SKU = input.SKU
		v.Barcode = input.Barcode
		v.Title = input.Title
		v.Size = input.Size
		v.Color = input.Color
		v.ColorHex = input.ColorHex
		v.ShadeCode = input.ShadeCode
		v.BatchNumber = input.BatchNumber
		v.ExpiryDate = input.ExpiryDate
		v.Price = input.Price
		v.CostPrice = input.CostPrice
		v.ComparePrice = input.ComparePrice
		v.FloorPrice = input.FloorPrice
		v.Weight = input.Weight
		v.ImageURL = input.ImageURL
		v.IsActive = true

		err := tx.QueryRow(ctx, query,
			productID, shopID, input.SKU, input.Barcode, input.Title,
			input.Size, input.Color, input.ColorHex, input.ShadeCode,
			input.BatchNumber, input.ExpiryDate, input.Price, input.CostPrice,
			input.ComparePrice, input.FloorPrice, input.Weight, input.ImageURL,
		).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to insert variant %s: %w", input.SKU, err)
		}

		if initialStockEach > 0 && targetLocID != "" {
			stockQuery := `
				INSERT INTO location_inventory (shop_id, location_id, product_id, variant_id, quantity, reserved_quantity, low_stock_threshold)
				VALUES ($1, $2, $3, $4, $5, 0, 5)
				ON CONFLICT (location_id, product_id, variant_id) WHERE variant_id IS NOT NULL
				DO UPDATE SET quantity = location_inventory.quantity + EXCLUDED.quantity, updated_at = NOW();
			`
			_, err = tx.Exec(ctx, stockQuery, shopID, targetLocID, productID, v.ID, initialStockEach)
			if err != nil {
				return nil, fmt.Errorf("failed to seed inventory for variant %s: %w", input.SKU, err)
			}
		}

		results = append(results, &v)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit bulk variants: %w", err)
	}

	return results, nil
}

// ListVariantsByProduct retrieves all active variant SKUs of a parent product.
func (r *LocationRepo) ListVariantsByProduct(ctx context.Context, shopID, productID string, locationID string) ([]*model.ProductVariant, error) {
	query := `
		SELECT pv.id, pv.product_id, pv.shop_id, pv.sku, COALESCE(pv.barcode, ''), 
		       pv.title, COALESCE(pv.size, ''), COALESCE(pv.color, ''), COALESCE(pv.color_hex, ''), 
		       COALESCE(pv.shade_code, ''), COALESCE(pv.batch_number, ''), pv.expiry_date,
		       pv.price, pv.cost_price, pv.compare_price, pv.floor_price, pv.weight, 
		       COALESCE(pv.image_url, ''), pv.is_active,
		       COALESCE(li.quantity - li.reserved_quantity, 0) AS available_qty,
		       pv.created_at, pv.updated_at
		FROM product_variants pv
		LEFT JOIN location_inventory li ON li.variant_id = pv.id 
		     AND ($3 = '' OR li.location_id = $3::uuid)
		WHERE pv.shop_id = $1 AND pv.product_id = $2 AND pv.is_active = true
		ORDER BY pv.size ASC, pv.color ASC, pv.title ASC;
	`
	rows, err := r.db.Query(ctx, query, shopID, productID, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variants: %w", err)
	}
	defer rows.Close()

	var list []*model.ProductVariant
	for rows.Next() {
		var v model.ProductVariant
		if err := rows.Scan(
			&v.ID, &v.ProductID, &v.ShopID, &v.SKU, &v.Barcode,
			&v.Title, &v.Size, &v.Color, &v.ColorHex,
			&v.ShadeCode, &v.BatchNumber, &v.ExpiryDate,
			&v.Price, &v.CostPrice, &v.ComparePrice, &v.FloorPrice, &v.Weight,
			&v.ImageURL, &v.IsActive, &v.AvailableQty,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan variant: %w", err)
		}
		list = append(list, &v)
	}
	return list, nil
}

// ListVariantsPublic retrieves active variants of a product for public customer storefronts.
// Zeroes out cost_price and floor_price (Rule 15 - Enterprise Privacy Shield).
func (r *LocationRepo) ListVariantsPublic(ctx context.Context, productID string) ([]*model.ProductVariant, error) {
	query := `
		SELECT pv.id, pv.product_id, pv.shop_id, pv.sku, COALESCE(pv.barcode, ''), 
		       pv.title, COALESCE(pv.size, ''), COALESCE(pv.color, ''), COALESCE(pv.color_hex, ''), 
		       COALESCE(pv.shade_code, ''), COALESCE(pv.batch_number, ''), pv.expiry_date,
		       pv.price, 0.0 AS cost_price, pv.compare_price, 0.0 AS floor_price, pv.weight, 
		       COALESCE(pv.image_url, ''), pv.is_active,
		       COALESCE(SUM(li.quantity - li.reserved_quantity), 0) AS available_qty,
		       pv.created_at, pv.updated_at
		FROM product_variants pv
		LEFT JOIN location_inventory li ON li.variant_id = pv.id
		WHERE pv.product_id = $1 AND pv.is_active = true
		GROUP BY pv.id
		ORDER BY pv.size ASC, pv.color ASC, pv.title ASC;
	`
	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to list public variants: %w", err)
	}
	defer rows.Close()

	var list []*model.ProductVariant
	for rows.Next() {
		var v model.ProductVariant
		if err := rows.Scan(
			&v.ID, &v.ProductID, &v.ShopID, &v.SKU, &v.Barcode,
			&v.Title, &v.Size, &v.Color, &v.ColorHex,
			&v.ShadeCode, &v.BatchNumber, &v.ExpiryDate,
			&v.Price, &v.CostPrice, &v.ComparePrice, &v.FloorPrice, &v.Weight,
			&v.ImageURL, &v.IsActive, &v.AvailableQty,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan public variant: %w", err)
		}
		list = append(list, &v)
	}
	return list, nil
}

// GetVariantByBarcode looks up a specific variant by scannable barcode.
func (r *LocationRepo) GetVariantByBarcode(ctx context.Context, shopID, barcode string) (*model.ProductVariant, error) {
	query := `
		SELECT id, product_id, shop_id, sku, barcode, title, 
		       COALESCE(size, ''), COALESCE(color, ''), COALESCE(color_hex, ''), 
		       COALESCE(shade_code, ''), COALESCE(batch_number, ''), expiry_date,
		       price, cost_price, compare_price, floor_price, weight, 
		       COALESCE(image_url, ''), is_active, created_at, updated_at
		FROM product_variants
		WHERE shop_id = $1 AND barcode = $2 AND is_active = true
		LIMIT 1;
	`
	var v model.ProductVariant
	err := r.db.QueryRow(ctx, query, shopID, barcode).Scan(
		&v.ID, &v.ProductID, &v.ShopID, &v.SKU, &v.Barcode, &v.Title,
		&v.Size, &v.Color, &v.ColorHex,
		&v.ShadeCode, &v.BatchNumber, &v.ExpiryDate,
		&v.Price, &v.CostPrice, &v.ComparePrice, &v.FloorPrice, &v.Weight,
		&v.ImageURL, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVariantNotFound
		}
		return nil, fmt.Errorf("failed to lookup variant by barcode: %w", err)
	}
	return &v, nil
}

// UpdateVariant updates variant details (price, margins, barcode).
func (r *LocationRepo) UpdateVariant(ctx context.Context, shopID, variantID string, input dto.UpdateVariantRequest) (*model.ProductVariant, error) {
	query := `
		UPDATE product_variants
		SET barcode = COALESCE($3, barcode),
		    title = COALESCE($4, title),
		    size = COALESCE($5, size),
		    color = COALESCE($6, color),
		    color_hex = COALESCE($7, color_hex),
		    shade_code = COALESCE($8, shade_code),
		    batch_number = COALESCE($9, batch_number),
		    expiry_date = COALESCE($10, expiry_date),
		    price = COALESCE($11, price),
		    cost_price = COALESCE($12, cost_price),
		    compare_price = COALESCE($13, compare_price),
		    floor_price = COALESCE($14, floor_price),
		    image_url = COALESCE($15, image_url),
		    is_active = COALESCE($16, is_active),
		    updated_at = NOW()
		WHERE shop_id = $1 AND id = $2
		RETURNING id, product_id, shop_id, sku, COALESCE(barcode, ''), title, 
		          COALESCE(size, ''), COALESCE(color, ''), COALESCE(color_hex, ''), 
		          COALESCE(shade_code, ''), COALESCE(batch_number, ''), expiry_date,
		          price, cost_price, compare_price, floor_price, weight, 
		          COALESCE(image_url, ''), is_active, created_at, updated_at;
	`
	var v model.ProductVariant
	err := r.db.QueryRow(ctx, query,
		shopID, variantID, input.Barcode, input.Title,
		input.Size, input.Color, input.ColorHex, input.ShadeCode,
		input.BatchNumber, input.ExpiryDate, input.Price, input.CostPrice,
		input.ComparePrice, input.FloorPrice, input.ImageURL, input.IsActive,
	).Scan(
		&v.ID, &v.ProductID, &v.ShopID, &v.SKU, &v.Barcode, &v.Title,
		&v.Size, &v.Color, &v.ColorHex,
		&v.ShadeCode, &v.BatchNumber, &v.ExpiryDate,
		&v.Price, &v.CostPrice, &v.ComparePrice, &v.FloorPrice, &v.Weight,
		&v.ImageURL, &v.IsActive, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVariantNotFound
		}
		return nil, fmt.Errorf("failed to update variant: %w", err)
	}
	return &v, nil
}

// DeleteVariant marks a variant as inactive.
func (r *LocationRepo) DeleteVariant(ctx context.Context, shopID, variantID string) error {
	query := `UPDATE product_variants SET is_active = false, updated_at = NOW() WHERE shop_id = $1 AND id = $2;`
	res, err := r.db.Exec(ctx, query, shopID, variantID)
	if err != nil {
		return fmt.Errorf("failed to delete variant: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrVariantNotFound
	}
	return nil
}

// ==============================================================================
// 3. LOCATION INVENTORY (STOCK PER LOCATION & ROW-LOCK GUARDS)
// ==============================================================================

// GetLocationInventory returns the inventory records for a specific location.
func (r *LocationRepo) GetLocationInventory(ctx context.Context, shopID, locationID string) ([]*model.LocationInventory, error) {
	query := `
		SELECT li.id, li.shop_id, li.location_id, loc.name AS location_name,
		       li.product_id, p.name AS product_name,
		       li.variant_id, COALESCE(pv.title, '') AS variant_title, COALESCE(pv.sku, '') AS variant_sku,
		       li.quantity, li.reserved_quantity, (li.quantity - li.reserved_quantity) AS available_quantity,
		       li.low_stock_threshold, li.updated_at
		FROM location_inventory li
		JOIN locations loc ON loc.id = li.location_id
		JOIN products p ON p.id = li.product_id
		LEFT JOIN product_variants pv ON pv.id = li.variant_id
		WHERE li.shop_id = $1 AND li.location_id = $2
		ORDER BY p.name ASC, pv.title ASC;
	`
	rows, err := r.db.Query(ctx, query, shopID, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query location inventory: %w", err)
	}
	defer rows.Close()

	var list []*model.LocationInventory
	for rows.Next() {
		var item model.LocationInventory
		if err := rows.Scan(
			&item.ID, &item.ShopID, &item.LocationID, &item.LocationName,
			&item.ProductID, &item.ProductName,
			&item.VariantID, &item.VariantTitle, &item.VariantSKU,
			&item.Quantity, &item.ReservedQuantity, &item.AvailableQuantity,
			&item.LowStockThreshold, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan location inventory item: %w", err)
		}
		list = append(list, &item)
	}
	return list, nil
}

// AdjustLocationStock safely alters physical stock with row-level locks (FOR UPDATE).
func (r *LocationRepo) AdjustLocationStock(ctx context.Context, shopID, locationID, productID string, variantID *string, adjustment int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin stock adjustment tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentQty, reservedQty int
	var lockQuery string
	var args []interface{}

	if variantID != nil && *variantID != "" {
		lockQuery = `
			SELECT quantity, reserved_quantity 
			FROM location_inventory 
			WHERE location_id = $1 AND product_id = $2 AND variant_id = $3 
			FOR UPDATE;
		`
		args = []interface{}{locationID, productID, *variantID}
	} else {
		lockQuery = `
			SELECT quantity, reserved_quantity 
			FROM location_inventory 
			WHERE location_id = $1 AND product_id = $2 AND variant_id IS NULL 
			FOR UPDATE;
		`
		args = []interface{}{locationID, productID}
	}

	err = tx.QueryRow(ctx, lockQuery, args...).Scan(&currentQty, &reservedQty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if adjustment < 0 {
				return ErrInsufficientLocationStock
			}
			// Insert new record
			insertQuery := `
				INSERT INTO location_inventory (shop_id, location_id, product_id, variant_id, quantity, reserved_quantity, low_stock_threshold)
				VALUES ($1, $2, $3, $4, $5, 0, 5);
			`
			_, err = tx.Exec(ctx, insertQuery, shopID, locationID, productID, variantID, adjustment)
			if err != nil {
				return fmt.Errorf("failed to insert initial location inventory: %w", err)
			}
			return tx.Commit(ctx)
		}
		return fmt.Errorf("failed to lock location inventory row: %w", err)
	}

	newQty := currentQty + adjustment
	if newQty < 0 {
		return ErrInsufficientLocationStock
	}

	var updateQuery string
	if variantID != nil && *variantID != "" {
		updateQuery = `
			UPDATE location_inventory 
			SET quantity = $4, updated_at = NOW() 
			WHERE location_id = $1 AND product_id = $2 AND variant_id = $3;
		`
		_, err = tx.Exec(ctx, updateQuery, locationID, productID, *variantID, newQty)
	} else {
		updateQuery = `
			UPDATE location_inventory 
			SET quantity = $3, updated_at = NOW() 
			WHERE location_id = $1 AND product_id = $2 AND variant_id IS NULL;
		`
		_, err = tx.Exec(ctx, updateQuery, locationID, productID, newQty)
	}

	if err != nil {
		return fmt.Errorf("failed to update location inventory: %w", err)
	}

	return tx.Commit(ctx)
}

// ==============================================================================
// 4. STOCK TRANSFERS (GODOWN <-> SHOWROOM STATE MACHINE)
// ==============================================================================

// CreateStockTransfer creates a new transfer order and item lines in PENDING state.
func (r *LocationRepo) CreateStockTransfer(ctx context.Context, shopID, userID string, input dto.CreateStockTransferRequest) (*model.StockTransfer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transfer tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Generate sequential transfer number TRF-YYYYMM-XXXX
	now := time.Now()
	prefix := fmt.Sprintf("TRF-%04d%02d-", now.Year(), int(now.Month()))
	var count int
	countQuery := `SELECT COUNT(*) FROM stock_transfers WHERE shop_id = $1 AND transfer_number LIKE $2;`
	if err := tx.QueryRow(ctx, countQuery, shopID, prefix+"%").Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to count monthly transfers: %w", err)
	}
	transferNumber := fmt.Sprintf("%s%04d", prefix, count+1)

	headerQuery := `
		INSERT INTO stock_transfers (
			shop_id, transfer_number, from_location_id, to_location_id,
			status, notes, created_by
		) VALUES ($1, $2, $3, $4, 'PENDING', $5, $6)
		RETURNING id, created_at, updated_at;
	`
	var transfer model.StockTransfer
	transfer.ShopID = shopID
	transfer.TransferNumber = transferNumber
	transfer.FromLocationID = input.FromLocationID
	transfer.ToLocationID = input.ToLocationID
	transfer.Status = model.TransferStatusPending
	transfer.Notes = input.Notes
	transfer.CreatedBy = &userID

	err = tx.QueryRow(ctx, headerQuery,
		shopID, transferNumber, input.FromLocationID, input.ToLocationID,
		input.Notes, userID,
	).Scan(&transfer.ID, &transfer.CreatedAt, &transfer.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert stock transfer header: %w", err)
	}

	itemQuery := `
		INSERT INTO stock_transfer_items (
			transfer_id, product_id, variant_id, requested_qty, sent_qty, received_qty, notes
		) VALUES ($1, $2, $3, $4, 0, 0, $5)
		RETURNING id;
	`
	for _, it := range input.Items {
		var itemID string
		err := tx.QueryRow(ctx, itemQuery, transfer.ID, it.ProductID, it.VariantID, it.RequestedQty, it.Notes).Scan(&itemID)
		if err != nil {
			return nil, fmt.Errorf("failed to insert transfer item: %w", err)
		}
		transfer.Items = append(transfer.Items, model.StockTransferItem{
			ID:           itemID,
			TransferID:   transfer.ID,
			ProductID:    it.ProductID,
			VariantID:    it.VariantID,
			RequestedQty: it.RequestedQty,
			Notes:        it.Notes,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit stock transfer: %w", err)
	}

	return &transfer, nil
}

// ListStockTransfers lists transfers filtered by shop and optional status.
func (r *LocationRepo) ListStockTransfers(ctx context.Context, shopID string, status *model.TransferStatus) ([]*model.StockTransfer, error) {
	query := `
		SELECT st.id, st.shop_id, st.transfer_number, 
		       st.from_location_id, lfrom.name AS from_location_name,
		       st.to_location_id, lto.name AS to_location_name,
		       st.status, COALESCE(st.notes, ''),
		       st.created_by, COALESCE(u1.full_name, 'Staff') AS created_by_name,
		       st.received_by, COALESCE(u2.full_name, '') AS received_by_name,
		       st.dispatched_at, st.received_at, st.created_at, st.updated_at
		FROM stock_transfers st
		JOIN locations lfrom ON lfrom.id = st.from_location_id
		JOIN locations lto ON lto.id = st.to_location_id
		LEFT JOIN users u1 ON u1.id = st.created_by
		LEFT JOIN users u2 ON u2.id = st.received_by
		WHERE st.shop_id = $1 AND ($2 = '' OR st.status = $2)
		ORDER BY st.created_at DESC;
	`
	statusStr := ""
	if status != nil {
		statusStr = string(*status)
	}

	rows, err := r.db.Query(ctx, query, shopID, statusStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock transfers: %w", err)
	}
	defer rows.Close()

	var list []*model.StockTransfer
	for rows.Next() {
		var t model.StockTransfer
		var stStr string
		if err := rows.Scan(
			&t.ID, &t.ShopID, &t.TransferNumber,
			&t.FromLocationID, &t.FromLocationName,
			&t.ToLocationID, &t.ToLocationName,
			&stStr, &t.Notes,
			&t.CreatedBy, &t.CreatedByName,
			&t.ReceivedBy, &t.ReceivedByName,
			&t.DispatchedAt, &t.ReceivedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan stock transfer: %w", err)
		}
		t.Status = model.TransferStatus(stStr)
		list = append(list, &t)
	}
	return list, nil
}

// GetStockTransferByID retrieves a single transfer order with its line items.
func (r *LocationRepo) GetStockTransferByID(ctx context.Context, shopID, transferID string) (*model.StockTransfer, error) {
	query := `
		SELECT st.id, st.shop_id, st.transfer_number, 
		       st.from_location_id, lfrom.name AS from_location_name,
		       st.to_location_id, lto.name AS to_location_name,
		       st.status, COALESCE(st.notes, ''),
		       st.created_by, COALESCE(u1.full_name, 'Staff') AS created_by_name,
		       st.received_by, COALESCE(u2.full_name, '') AS received_by_name,
		       st.dispatched_at, st.received_at, st.created_at, st.updated_at
		FROM stock_transfers st
		JOIN locations lfrom ON lfrom.id = st.from_location_id
		JOIN locations lto ON lto.id = st.to_location_id
		LEFT JOIN users u1 ON u1.id = st.created_by
		LEFT JOIN users u2 ON u2.id = st.received_by
		WHERE st.shop_id = $1 AND st.id = $2;
	`
	var t model.StockTransfer
	var stStr string
	err := r.db.QueryRow(ctx, query, shopID, transferID).Scan(
		&t.ID, &t.ShopID, &t.TransferNumber,
		&t.FromLocationID, &t.FromLocationName,
		&t.ToLocationID, &t.ToLocationName,
		&stStr, &t.Notes,
		&t.CreatedBy, &t.CreatedByName,
		&t.ReceivedBy, &t.ReceivedByName,
		&t.DispatchedAt, &t.ReceivedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransferNotFound
		}
		return nil, fmt.Errorf("failed to get stock transfer: %w", err)
	}
	t.Status = model.TransferStatus(stStr)

	itemsQuery := `
		SELECT sti.id, sti.transfer_id, sti.product_id, p.name AS product_name,
		       sti.variant_id, COALESCE(pv.title, '') AS variant_title, COALESCE(pv.sku, '') AS variant_sku,
		       sti.requested_qty, sti.sent_qty, sti.received_qty, COALESCE(sti.notes, '')
		FROM stock_transfer_items sti
		JOIN products p ON p.id = sti.product_id
		LEFT JOIN product_variants pv ON pv.id = sti.variant_id
		WHERE sti.transfer_id = $1;
	`
	itemRows, err := r.db.Query(ctx, itemsQuery, transferID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transfer items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var it model.StockTransferItem
		if err := itemRows.Scan(
			&it.ID, &it.TransferID, &it.ProductID, &it.ProductName,
			&it.VariantID, &it.VariantTitle, &it.VariantSKU,
			&it.RequestedQty, &it.SentQty, &it.ReceivedQty, &it.Notes,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transfer item: %w", err)
		}
		t.Items = append(t.Items, it)
	}

	return &t, nil
}

// UpdateStockTransferStatus moves the transfer along the state machine and adjusts physical stock.
func (r *LocationRepo) UpdateStockTransferStatus(ctx context.Context, shopID, transferID, userID string, newStatus model.TransferStatus, notes string) (*model.StockTransfer, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin status update tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Fetch current status and lock the transfer row
	var currentStatusStr, fromLocID, toLocID string
	err = tx.QueryRow(ctx, `
		SELECT status, from_location_id, to_location_id 
		FROM stock_transfers 
		WHERE shop_id = $1 AND id = $2 
		FOR UPDATE;
	`, shopID, transferID).Scan(&currentStatusStr, &fromLocID, &toLocID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransferNotFound
		}
		return nil, fmt.Errorf("failed to lock transfer: %w", err)
	}

	currentStatus := model.TransferStatus(currentStatusStr)

	// Validate valid state transitions
	// PENDING -> APPROVED -> DISPATCHED -> IN_TRANSIT -> RECEIVED -> RECONCILED (or CANCELLED from PENDING/APPROVED)
	switch newStatus {
	case model.TransferStatusApproved:
		if currentStatus != model.TransferStatusPending {
			return nil, ErrInvalidTransferState
		}
	case model.TransferStatusDispatched:
		if currentStatus != model.TransferStatusApproved && currentStatus != model.TransferStatusPending {
			return nil, ErrInvalidTransferState
		}
		// When DISPATCHED: Deduct stock from source godown location
		itemRows, err := tx.Query(ctx, `SELECT product_id, variant_id, requested_qty FROM stock_transfer_items WHERE transfer_id = $1;`, transferID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch items for dispatch: %w", err)
		}
		type dispatchItem struct {
			pID  string
			vID  *string
			qReq int
		}
		var dItems []dispatchItem
		for itemRows.Next() {
			var it dispatchItem
			if err := itemRows.Scan(&it.pID, &it.vID, &it.qReq); err != nil {
				itemRows.Close()
				return nil, err
			}
			dItems = append(dItems, it)
		}
		itemRows.Close()

		for _, it := range dItems {
			// Deduct from source location
			deductQuery := `
				UPDATE location_inventory 
				SET quantity = quantity - $4, updated_at = NOW() 
				WHERE location_id = $1 AND product_id = $2 
				  AND (($3::uuid IS NULL AND variant_id IS NULL) OR (variant_id = $3::uuid))
				  AND quantity >= $4;
			`
			res, err := tx.Exec(ctx, deductQuery, fromLocID, it.pID, it.vID, it.qReq)
			if err != nil || res.RowsAffected() == 0 {
				return nil, fmt.Errorf("%w for product %s", ErrInsufficientLocationStock, it.pID)
			}
			// Update sent_qty on item line
			_, _ = tx.Exec(ctx, `UPDATE stock_transfer_items SET sent_qty = $2 WHERE transfer_id = $1 AND product_id = $3;`, transferID, it.qReq, it.pID)
		}

	case model.TransferStatusInTransit:
		if currentStatus != model.TransferStatusDispatched {
			return nil, ErrInvalidTransferState
		}
	case model.TransferStatusReceived:
		if currentStatus != model.TransferStatusInTransit && currentStatus != model.TransferStatusDispatched {
			return nil, ErrInvalidTransferState
		}
		_, _ = tx.Exec(ctx, `UPDATE stock_transfer_items SET received_qty = sent_qty WHERE transfer_id = $1;`, transferID)

	case model.TransferStatusReconciled:
		if currentStatus != model.TransferStatusReceived {
			return nil, ErrInvalidTransferState
		}
		// Credit stock into destination location
		itemRows, err := tx.Query(ctx, `SELECT product_id, variant_id, received_qty FROM stock_transfer_items WHERE transfer_id = $1;`, transferID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch received items: %w", err)
		}
		type creditItem struct {
			pID  string
			vID  *string
			qRec int
		}
		var cItems []creditItem
		for itemRows.Next() {
			var it creditItem
			if err := itemRows.Scan(&it.pID, &it.vID, &it.qRec); err != nil {
				itemRows.Close()
				return nil, err
			}
			cItems = append(cItems, it)
		}
		itemRows.Close()

		for _, it := range cItems {
			if it.vID != nil {
				creditQuery := `
					INSERT INTO location_inventory (shop_id, location_id, product_id, variant_id, quantity, reserved_quantity, low_stock_threshold)
					VALUES ($1, $2, $3, $4, $5, 0, 5)
					ON CONFLICT (location_id, product_id, variant_id) WHERE variant_id IS NOT NULL
					DO UPDATE SET quantity = location_inventory.quantity + EXCLUDED.quantity, updated_at = NOW();
				`
				_, err = tx.Exec(ctx, creditQuery, shopID, toLocID, it.pID, it.vID, it.qRec)
			} else {
				creditQuery := `
					INSERT INTO location_inventory (shop_id, location_id, product_id, quantity, reserved_quantity, low_stock_threshold)
					VALUES ($1, $2, $3, $4, 0, 5)
					ON CONFLICT (location_id, product_id) WHERE variant_id IS NULL
					DO UPDATE SET quantity = location_inventory.quantity + EXCLUDED.quantity, updated_at = NOW();
				`
				_, err = tx.Exec(ctx, creditQuery, shopID, toLocID, it.pID, it.qRec)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to credit inventory: %w", err)
			}
		}

	case model.TransferStatusCancelled:
		if currentStatus != model.TransferStatusPending && currentStatus != model.TransferStatusApproved {
			return nil, errors.New("cannot cancel a transfer that has already been dispatched")
		}
	}

	updateQuery := `
		UPDATE stock_transfers
		SET status = $3::varchar,
		    notes = CASE WHEN $4::text != '' THEN $4::text ELSE notes END,
		    dispatched_at = CASE WHEN $3::text = 'DISPATCHED' AND dispatched_at IS NULL THEN NOW() ELSE dispatched_at END,
		    received_at = CASE WHEN $3::text = 'RECEIVED' AND received_at IS NULL THEN NOW() ELSE received_at END,
		    received_by = CASE WHEN $3::text IN ('RECEIVED', 'RECONCILED') AND received_by IS NULL THEN NULLIF($5, '')::uuid ELSE received_by END,
		    updated_at = NOW()
		WHERE shop_id = $1 AND id = $2;
	`
	_, err = tx.Exec(ctx, updateQuery, shopID, transferID, string(newStatus), notes, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update transfer status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit status update: %w", err)
	}

	return r.GetStockTransferByID(ctx, shopID, transferID)
}

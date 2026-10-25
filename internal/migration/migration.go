package migration

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var migrationFS embed.FS

type migrationEntry struct {
	version  int64
	filename string
	content  string
}

// RunAutoMigrations executes all pending embedded .up.sql migrations natively via pgxpool.
// This completely eliminates runtime dependency on external filesystem paths in cloud/Docker
// and avoids broken advisory locks with Supabase PgBouncer transaction poolers.
func RunAutoMigrations(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 1. Ensure schema_migrations tracking table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			dirty BOOLEAN NOT NULL DEFAULT false
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize schema_migrations: %w", err)
	}

	// 2. Fetch applied versions
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations WHERE dirty = false;")
	if err != nil {
		return fmt.Errorf("failed to fetch schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err == nil {
			applied[v] = true
		}
	}

	// 3. Read embedded migration files
	dirEntries, err := migrationFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	var pending []migrationEntry
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			parts := strings.SplitN(entry.Name(), "_", 2)
			if len(parts) > 1 {
				v, parseErr := strconv.ParseInt(parts[0], 10, 64)
				if parseErr == nil {
					data, readErr := migrationFS.ReadFile(entry.Name())
					if readErr == nil {
						pending = append(pending, migrationEntry{
							version:  v,
							filename: entry.Name(),
							content:  string(data),
						})
					}
				}
			}
		}
	}

	// Sort migrations in ascending order
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].version < pending[j].version
	})

	// 4. Apply unapplied migrations sequentially
	for _, m := range pending {
		if applied[m.version] {
			continue
		}

		if _, err := pool.Exec(ctx, m.content); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", m.filename, err)
		}

		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version, dirty) VALUES ($1, false) ON CONFLICT (version) DO UPDATE SET dirty = false;", m.version)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", m.filename, err)
		}
	}

	return nil
}

// EnsureSchemaColumns runs fast, idempotent ALTER TABLE and index optimizations.
func EnsureSchemaColumns(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queries := []string{
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS floor_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS allow_bargain BOOLEAN DEFAULT false;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS is_price_public BOOLEAN DEFAULT true;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS compare_price DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE users ALTER COLUMN phone DROP NOT NULL;",
		"UPDATE users SET phone = NULL WHERE phone = '';",
		"ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_key;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users(phone) WHERE phone IS NOT NULL AND phone != '';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS upi_id VARCHAR(100) DEFAULT '';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS opening_time VARCHAR(10) DEFAULT '09:00';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS closing_time VARCHAR(10) DEFAULT '21:00';",
		"ALTER TABLE shops ADD COLUMN IF NOT EXISTS weekly_off VARCHAR(20) DEFAULT '';",
		"ALTER TABLE shops ALTER COLUMN phone DROP NOT NULL;",
		"ALTER TABLE shops ALTER COLUMN pincode DROP NOT NULL;",
		"ALTER TABLE shops ALTER COLUMN city DROP NOT NULL;",
		"ALTER TABLE categories ADD COLUMN IF NOT EXISTS image_url VARCHAR(500) DEFAULT '';",
		"ALTER TABLE categories ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES categories(id) ON DELETE SET NULL;",
		"INSERT INTO categories (name, slug, description, icon, is_active) VALUES ('General Store', 'general-store', 'All grocery and general store everyday items', '🛒', true) ON CONFLICT (slug) DO NOTHING;",

		// POS Bills & Sales table enhancements
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS customer_user_id UUID REFERENCES users(id) ON DELETE SET NULL;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS subtotal DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS total_cost DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cash_amount DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS online_amount DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS khata_amount DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;",
		"ALTER TABLE pos_bills ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP;",
		"ALTER TABLE pos_bills ALTER COLUMN cashier_user_id DROP NOT NULL;",

		// POS Bill Items enhancements
		"ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS product_sku VARCHAR(100) DEFAULT '';",
		"ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS unit_cost DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE pos_bill_items ALTER COLUMN product_id DROP NOT NULL;",

		// Khata table enhancements
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS customer_id UUID;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS customer_mobile VARCHAR(20);",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(20);",
		"ALTER TABLE customer_khata ALTER COLUMN customer_phone DROP NOT NULL;",
		"UPDATE customer_khata SET customer_mobile = customer_phone WHERE (customer_mobile IS NULL OR customer_mobile = '') AND customer_phone IS NOT NULL;",
		"UPDATE customer_khata SET customer_phone = customer_mobile WHERE (customer_phone IS NULL OR customer_phone = '') AND customer_mobile IS NOT NULL;",
		"CREATE INDEX IF NOT EXISTS idx_customer_khata_shop_mobile ON customer_khata(shop_id, customer_mobile);",
		"CREATE INDEX IF NOT EXISTS idx_customer_khata_shop_phone ON customer_khata(shop_id, customer_phone);",

		// Fix real geographic coordinates for known shop locations
		"UPDATE shops SET latitude = 28.5828, longitude = 77.3110, city = 'Noida' WHERE (LOWER(city) LIKE '%noida%' OR LOWER(address) LIKE '%noida%' OR LOWER(slug) LIKE '%shopsilo%') AND (latitude = 26.1542 OR latitude IS NULL);",
		"UPDATE shops SET latitude = 26.1542, longitude = 85.8918, city = 'Darbhanga' WHERE (LOWER(city) LIKE '%darbhanga%' OR LOWER(address) LIKE '%darbhanga%' OR LOWER(slug) LIKE '%om-cloth%') AND (latitude = 28.5828 OR latitude IS NULL);",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS closure_status VARCHAR(20) DEFAULT 'ACTIVE';",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS closure_otp VARCHAR(10);",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS closure_requested_by VARCHAR(20);",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS closure_requested_at TIMESTAMP;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS promise_to_pay_date DATE;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS installment_target DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS credit_otp_required BOOLEAN DEFAULT false;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS credit_otp_threshold DECIMAL(10, 2) DEFAULT 0;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS trust_score INT DEFAULT 100;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS trust_badge VARCHAR(20) DEFAULT 'TRUSTED';",

		// Khata transactions enhancements
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS shop_id UUID REFERENCES shops(id) ON DELETE CASCADE;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS bill_number VARCHAR(50);",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS payment_mode VARCHAR(30) DEFAULT 'CASH';",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS status VARCHAR(30) DEFAULT 'CONFIRMED';",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS dispute_reason TEXT;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS disputed_at TIMESTAMP;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS resolution_action VARCHAR(30);",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS resolution_notes TEXT;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS reversal_of_id UUID REFERENCES khata_transactions(id) ON DELETE SET NULL;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS upi_ref_no VARCHAR(100);",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS parchi_image_url TEXT;",
		"ALTER TABLE khata_transactions ADD COLUMN IF NOT EXISTS items_summary TEXT;",

		// POS Parked Bills & Live Multi-Customer Baskets
		`CREATE TABLE IF NOT EXISTS master_catalog_products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			barcode VARCHAR(64) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			brand VARCHAR(100) NOT NULL DEFAULT '',
			category VARCHAR(100) NOT NULL DEFAULT 'General Store',
			unit VARCHAR(50) NOT NULL DEFAULT 'piece',
			mrp DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
			selling_price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
			description TEXT,
			image_url TEXT,
			source VARCHAR(50) NOT NULL DEFAULT 'crowdsourced',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);,
		CREATE UNIQUE INDEX IF NOT EXISTS idx_master_catalog_barcode ON master_catalog_products (TRIM(LOWER(barcode)));,
		CREATE INDEX IF NOT EXISTS idx_master_catalog_brand ON master_catalog_products (TRIM(LOWER(brand)));,
		CREATE INDEX IF NOT EXISTS idx_master_catalog_category ON master_catalog_products (category);,
		ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS scan_count INT DEFAULT 1;,
		ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS trust_score INT DEFAULT 100;,
		ALTER TABLE master_catalog_products ADD COLUMN IF NOT EXISTS attributes JSONB NOT NULL DEFAULT '{}'::jsonb;,
		CREATE INDEX IF NOT EXISTS idx_master_catalog_attributes ON master_catalog_products USING GIN (attributes);,

		CREATE TABLE IF NOT EXISTS pos_parked_bills (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			label VARCHAR(100) NOT NULL DEFAULT '',
			customer_phone VARCHAR(20) NOT NULL DEFAULT '',
			cart_data JSONB NOT NULL DEFAULT '{}'::jsonb,
			total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pos_parked_bills_shop_id ON pos_parked_bills(shop_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pos_parked_bills_shop_label ON pos_parked_bills(shop_id, UPPER(TRIM(label))) WHERE label != '';`,

		// Performance Indexes (Foreign Key Speedups, Trigram GIN Indexes, Composite B-Trees)
		"CREATE EXTENSION IF NOT EXISTS pg_trgm;",
		"CREATE INDEX IF NOT EXISTS idx_products_trgm ON products USING gin (name gin_trgm_ops, description gin_trgm_ops, sku gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_shops_trgm ON shops USING gin (name gin_trgm_ops, city gin_trgm_ops, category gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_categories_trgm ON categories USING gin (name gin_trgm_ops, slug gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_shop_active ON products(shop_id, is_active);",
		"CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_created ON pos_bills(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_khata_tx_shop_created ON khata_transactions(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_reservations_user_active ON reservations(user_id, status, expires_at);",
		"CREATE INDEX IF NOT EXISTS idx_pos_bills_customer ON pos_bills(customer_user_id);",
		"CREATE INDEX IF NOT EXISTS idx_pos_bill_items_prod ON pos_bill_items(product_id);",
		"CREATE INDEX IF NOT EXISTS idx_reservations_product ON reservations(product_id);",
		"CREATE INDEX IF NOT EXISTS idx_khata_tx_reversal ON khata_transactions(reversal_of_id);",
		"CREATE INDEX IF NOT EXISTS idx_daily_expenses_shop_created ON daily_expenses(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_store_offers_shop_active ON store_offers(shop_id, is_active);",
		"CREATE INDEX IF NOT EXISTS idx_banned_entities_banned_by ON banned_entities(banned_by);",
		"CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);",
		// Step 2 Production Optimizations: Trigram & Geo Filtered Indexes
		"CREATE INDEX IF NOT EXISTS idx_products_name_trgm ON products USING gin (name gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_sku_trgm ON products USING gin (sku gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_desc_trgm ON products USING gin (description gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_shops_geo_active ON shops (latitude, longitude) WHERE is_active = true AND status = 'active' AND latitude IS NOT NULL AND longitude IS NOT NULL;",
		"CREATE INDEX IF NOT EXISTS idx_inventory_in_stock ON inventory (product_id) WHERE (quantity - reserved_quantity) > 0;",

		// Step 3 Production Optimizations: Barcode, 10-Digit Phone, Khata Ledger, & Expiry Cleanup
		"CREATE INDEX IF NOT EXISTS idx_products_shop_barcode ON products(shop_id, (attributes->>'barcode')) WHERE attributes ? 'barcode';",
		"CREATE INDEX IF NOT EXISTS idx_products_shop_sku_upper ON products(shop_id, UPPER(TRIM(sku))) WHERE sku IS NOT NULL AND sku != '';",
		"CREATE INDEX IF NOT EXISTS idx_products_attributes_gin ON products USING gin (attributes jsonb_path_ops);",
		"CREATE INDEX IF NOT EXISTS idx_pos_bills_customer_phone ON pos_bills(customer_phone, created_at DESC) WHERE customer_phone IS NOT NULL AND customer_phone != '';",
		"CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_status_date ON pos_bills(shop_id, status, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_customer_khata_balance ON customer_khata(shop_id, current_balance DESC) WHERE current_balance > 0;",
		"CREATE INDEX IF NOT EXISTS idx_users_phone_last10 ON users (RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10));",
		"CREATE INDEX IF NOT EXISTS idx_customer_khata_phone_last10 ON customer_khata (shop_id, RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10));",
		"CREATE INDEX IF NOT EXISTS idx_reservations_pickup_code ON reservations(shop_id, pickup_code) WHERE status = 'active';",
		"CREATE INDEX IF NOT EXISTS idx_reservations_active_expires ON reservations(expires_at) WHERE status = 'active';",
		"CREATE INDEX IF NOT EXISTS idx_products_shop_cat_active ON products(shop_id, category_id, is_active) WHERE is_active = true;",
		"CREATE INDEX IF NOT EXISTS idx_products_featured ON products(shop_id, is_featured, created_at DESC) WHERE is_featured = true AND is_active = true;",

		`INSERT INTO categories (name, slug, description, icon, is_active) VALUES
		('Kirana & Grocery', 'kirana-grocery', 'Daily essentials, rice, flour, oil, pulses and spices', '🛒', true),
		('Dairy & Breakfast', 'dairy-breakfast', 'Milk, curd, butter, paneer, eggs and bread', '🥛', true),
		('Snacks & Beverages', 'snacks-beverages', 'Biscuits, namkeen, cold drinks, tea and coffee', '☕', true),
		('Fruits & Vegetables', 'fruits-vegetables', 'Fresh local farm fruits and green vegetables', '🧺', true),
		('Personal Care', 'personal-care', 'Soaps, shampoos, toothpaste, skincare and haircare', '🧼', true),
		('Household Cleaning', 'household-cleaning', 'Detergents, floor cleaners, dishwash and mosquito repellents', '🧹', true),
		('Clothing & Apparel', 'clothing-apparel', 'Menswear, womenswear, kids wear and innerwear', '👕', true),
		('Footwear', 'footwear', 'Casual shoes, sandals, slippers and formal shoes', '👟', true),
		('Electronics & Accessories', 'electronics-accessories', 'Mobile chargers, earphones, cables, bulbs and batteries', '⚡', true),
		('Pharmacy & Healthcare', 'pharmacy-healthcare', 'Over-the-counter medicines, first aid, vitamins and supplements', '💊', true),
		('Baby Care', 'baby-care', 'Diapers, wipes, baby food and baby skincare', '👶', true),
		('Pet Care', 'pet-care', 'Dog food, cat food, pet shampoo and treats', '🐾', true),
		('Stationery & Office', 'stationery-office', 'Notebooks, pens, registers, tapes and craft items', '📚', true),
		('Pooja & Spiritual', 'pooja-spiritual', 'Agarbatti, diya, ghee, camphor and sacred items', '🪔', true),
		('Hardware & Tools', 'hardware-tools', 'Locks, nails, hammers, adhesives and tools', '🔧', true),
		('Home & Kitchen', 'home-kitchen', 'Cookware, utensils, storage containers and bottles', '🍳', true),
		('Beauty & Cosmetics', 'beauty-cosmetics', 'Makeup, perfumes, nail polish and face creams', '💄', true),
		('Sports & Fitness', 'sports-fitness', 'Cricket gear, footballs, gym bottles and badminton', '🏏', true),
		('Toys & Games', 'toys-games', 'Board games, puzzles, soft toys and toy cars', '🧸', true),
		('Bakery & Cakes', 'bakery-cakes', 'Fresh bread, cakes, pastries, cookies and patties', '🎂', true)
		ON CONFLICT (slug) DO UPDATE SET is_active = true;`,
	}

	for _, q := range queries {
		_, _ = pool.Exec(ctx, q)
	}
}

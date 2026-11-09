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
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS loyalty_points INT DEFAULT 0;",
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
		"ALTER TABLE categories ADD COLUMN IF NOT EXISTS shop_id UUID REFERENCES shops(id) ON DELETE CASCADE;",
		"ALTER TABLE categories ADD COLUMN IF NOT EXISTS icon VARCHAR(50) DEFAULT '📦';",
		"ALTER TABLE categories ADD COLUMN IF NOT EXISTS is_custom BOOLEAN DEFAULT false;",
		"ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_slug_key;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_global_slug ON categories(slug) WHERE shop_id IS NULL;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_shop_slug ON categories(shop_id, slug) WHERE shop_id IS NOT NULL;",
		"CREATE INDEX IF NOT EXISTS idx_categories_shop_lookup ON categories(shop_id, is_active);",
		"INSERT INTO categories (name, slug, description, icon, is_active) VALUES ('General Store', 'general-store', 'All grocery and general store everyday items', '🛒', true) ON CONFLICT DO NOTHING;",

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

		// Step 4 Production Optimizations: Search Speedups (Shop Address & Product Tags/Attributes Trigram)
		"CREATE INDEX IF NOT EXISTS idx_shops_address_trgm ON shops USING gin (address gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_tags_gin ON products USING gin (tags);",
		`CREATE OR REPLACE FUNCTION immutable_array_to_string(arr text[], sep text) 
		 RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ 
		     SELECT array_to_string(arr, sep); 
		 $$;`,
		"CREATE INDEX IF NOT EXISTS idx_products_tags_trgm ON products USING gin (immutable_array_to_string(tags, ' ') gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_attributes_trgm ON products USING gin ((attributes::text) gin_trgm_ops);",

		// Step 5 Production Optimizations: PostGIS Spatial 2D Geography & GIST Index (Rock-solid Fallback)
		`DO $$
		BEGIN
			BEGIN
				CREATE EXTENSION IF NOT EXISTS postgis;
			EXCEPTION WHEN OTHERS THEN
				NULL;
			END;
		END;
		$$;`,
		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'geography') THEN
				IF NOT EXISTS (
					SELECT 1 FROM information_schema.columns 
					WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
				) THEN
					ALTER TABLE shops ADD COLUMN location geography(Point, 4326);
				END IF;
			END IF;
		END;
		$$;`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
			) THEN
				UPDATE shops 
				SET location = ST_SetSRID(ST_MakePoint(longitude::double precision, latitude::double precision), 4326)::geography 
				WHERE latitude IS NOT NULL AND longitude IS NOT NULL AND location IS NULL
				  AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180;
			END IF;
		EXCEPTION WHEN OTHERS THEN
			NULL;
		END;
		$$;`,
		`DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
			) THEN
				CREATE INDEX IF NOT EXISTS idx_shops_location_gist 
				ON shops USING gist (location) 
				WHERE is_active = true AND status = 'active';
			END IF;
		EXCEPTION WHEN OTHERS THEN
			NULL;
		END;
		$$;`,
		`CREATE OR REPLACE FUNCTION sync_shop_location()
		RETURNS TRIGGER 
		SECURITY DEFINER
		SET search_path = public, extensions, pg_catalog
		AS $$
		BEGIN
		    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
		        BEGIN
		            IF NEW.latitude BETWEEN -90 AND 90 AND NEW.longitude BETWEEN -180 AND 180 THEN
		                NEW.location := ST_SetSRID(ST_MakePoint(NEW.longitude::double precision, NEW.latitude::double precision), 4326)::geography;
		            ELSE
		                NEW.location := NULL;
		            END IF;
		        EXCEPTION WHEN OTHERS THEN
		            NEW.location := NULL;
		        END;
		    ELSE
		        BEGIN
		            NEW.location := NULL;
		        EXCEPTION WHEN OTHERS THEN
		            NULL;
		        END;
		    END IF;
		    RETURN NEW;
		EXCEPTION WHEN OTHERS THEN
		    RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;`,
		`DO $$
		BEGIN
		    IF EXISTS (
		        SELECT 1 FROM information_schema.columns 
		        WHERE table_schema = current_schema() AND table_name = 'shops' AND column_name = 'location'
		    ) THEN
		        DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
		        CREATE TRIGGER trg_sync_shop_location
		        BEFORE INSERT OR UPDATE OF latitude, longitude ON shops
		        FOR EACH ROW EXECUTE FUNCTION sync_shop_location();
		    ELSE
		        DROP TRIGGER IF EXISTS trg_sync_shop_location ON shops;
		    END IF;
		END;
		$$;`,

		`CREATE TABLE IF NOT EXISTS procurement_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			category VARCHAR(100) NOT NULL DEFAULT 'General Store',
			qty VARCHAR(100) NOT NULL DEFAULT '1',
			estimated_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
			actual_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
			dealer_name VARCHAR(255) NOT NULL DEFAULT '',
			dealer_phone VARCHAR(50) NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			is_completed BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_procurement_items_shop_id ON procurement_items(shop_id);",
		"CREATE INDEX IF NOT EXISTS idx_procurement_items_completed ON procurement_items(shop_id, is_completed);",

		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS discount_text VARCHAR(100) NOT NULL DEFAULT '';",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS discount_type VARCHAR(20) NOT NULL DEFAULT 'percentage';",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS discount_value DECIMAL(10, 2) NOT NULL DEFAULT 0.00;",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS min_order_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00;",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS max_discount DECIMAL(10, 2) NOT NULL DEFAULT 0.00;",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS applicable_days VARCHAR(100) NOT NULL DEFAULT 'all';",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS banner_tag VARCHAR(50) NOT NULL DEFAULT '🔥 Special Offer';",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS promo_code VARCHAR(50) NOT NULL DEFAULT '';",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS min_points_required INT NOT NULL DEFAULT 0;",
		"ALTER TABLE store_offers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();",
		"CREATE INDEX IF NOT EXISTS idx_store_offers_shop_days ON store_offers(shop_id, is_active, applicable_days);",
		`CREATE TABLE IF NOT EXISTS shop_offers_audit (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			offer_id VARCHAR(100) NOT NULL,
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			action VARCHAR(50) NOT NULL,
			previous_title VARCHAR(200),
			previous_discount_text VARCHAR(100),
			previous_description TEXT,
			new_title VARCHAR(200),
			new_discount_text VARCHAR(100),
			new_description TEXT,
			changed_by_user_id VARCHAR(100),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_shop_offers_audit_shop ON shop_offers_audit(shop_id, created_at DESC);",

		`CREATE TABLE IF NOT EXISTS shop_expenses (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			category VARCHAR(100) NOT NULL,
			amount DECIMAL(10, 2) NOT NULL,
			notes TEXT,
			payment_method VARCHAR(50) NOT NULL DEFAULT 'cash',
			expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop_date ON shop_expenses(shop_id, expense_date DESC);",
		"CREATE INDEX IF NOT EXISTS idx_shop_expenses_shop_created ON shop_expenses(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_shop_expenses_category ON shop_expenses(shop_id, category);",

		`CREATE TABLE IF NOT EXISTS audit_logs (
			id VARCHAR(64) PRIMARY KEY,
			actor_id VARCHAR(64),
			actor_email VARCHAR(255),
			action VARCHAR(64) NOT NULL,
			entity VARCHAR(64) NOT NULL,
			entity_id VARCHAR(64),
			old_state JSONB,
			new_state JSONB,
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity, entity_id);",
		"CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_id);",
		"CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at DESC);",

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

		// Multi-Store Godown Locations, Variant Matrix & Stock Transfers
		`CREATE TABLE IF NOT EXISTS locations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			code VARCHAR(50) NOT NULL,
			type VARCHAR(30) NOT NULL DEFAULT 'RETAIL_STORE' CHECK (type IN ('RETAIL_STORE', 'WAREHOUSE', 'GODOWN', 'OUTLET')),
			address TEXT,
			contact_person VARCHAR(100),
			contact_phone VARCHAR(20),
			is_default BOOLEAN NOT NULL DEFAULT false,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(shop_id, code)
		);`,
		"CREATE INDEX IF NOT EXISTS idx_locations_shop ON locations(shop_id, is_active);",
		"CREATE INDEX IF NOT EXISTS idx_locations_default ON locations(shop_id) WHERE is_default = true;",
		`INSERT INTO locations (shop_id, name, code, type, is_default, is_active)
		 SELECT id, 'Main Showroom', 'MAIN', 'RETAIL_STORE', true, true FROM shops
		 ON CONFLICT (shop_id, code) DO NOTHING;`,

		`CREATE TABLE IF NOT EXISTS product_variants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			sku VARCHAR(100) NOT NULL,
			barcode VARCHAR(100),
			title VARCHAR(200) NOT NULL,
			size VARCHAR(30),
			color VARCHAR(50),
			color_hex VARCHAR(10),
			shade_code VARCHAR(50),
			batch_number VARCHAR(50),
			expiry_date DATE,
			price DECIMAL(10, 2) NOT NULL,
			cost_price DECIMAL(10, 2) DEFAULT 0,
			compare_price DECIMAL(10, 2) DEFAULT 0,
			floor_price DECIMAL(10, 2) DEFAULT 0,
			weight DECIMAL(8, 2) DEFAULT 0,
			image_url TEXT,
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(shop_id, sku)
		);`,
		"CREATE INDEX IF NOT EXISTS idx_product_variants_product ON product_variants(product_id);",
		"CREATE INDEX IF NOT EXISTS idx_product_variants_shop ON product_variants(shop_id, is_active);",
		"CREATE INDEX IF NOT EXISTS idx_product_variants_barcode ON product_variants(shop_id, barcode) WHERE barcode IS NOT NULL AND barcode != '';",
		"CREATE INDEX IF NOT EXISTS idx_product_variants_size ON product_variants(shop_id, size) WHERE size IS NOT NULL;",
		"CREATE INDEX IF NOT EXISTS idx_product_variants_color ON product_variants(shop_id, color) WHERE color IS NOT NULL;",

		`CREATE TABLE IF NOT EXISTS location_inventory (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			location_id UUID NOT NULL REFERENCES locations(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
			quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
			reserved_quantity INT NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
			low_stock_threshold INT NOT NULL DEFAULT 5,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);`,
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_loc_inv_variant_uniq ON location_inventory(location_id, product_id, variant_id) WHERE variant_id IS NOT NULL;",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_loc_inv_simple_uniq ON location_inventory(location_id, product_id) WHERE variant_id IS NULL;",
		"CREATE INDEX IF NOT EXISTS idx_loc_inv_location_prod ON location_inventory(location_id, product_id);",

		`CREATE TABLE IF NOT EXISTS stock_transfers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			transfer_number VARCHAR(50) NOT NULL,
			from_location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
			to_location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,
			status VARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'DISPATCHED', 'IN_TRANSIT', 'RECEIVED', 'RECONCILED', 'CANCELLED')),
			notes TEXT,
			created_by UUID REFERENCES users(id) ON DELETE SET NULL,
			received_by UUID REFERENCES users(id) ON DELETE SET NULL,
			dispatched_at TIMESTAMPTZ,
			received_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(shop_id, transfer_number)
		);`,
		"CREATE INDEX IF NOT EXISTS idx_stock_transfers_shop_status ON stock_transfers(shop_id, status);",

		`CREATE TABLE IF NOT EXISTS stock_transfer_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			transfer_id UUID NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
			requested_qty INT NOT NULL CHECK (requested_qty > 0),
			sent_qty INT NOT NULL DEFAULT 0 CHECK (sent_qty >= 0),
			received_qty INT NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
			notes TEXT
		);`,
		"CREATE INDEX IF NOT EXISTS idx_stock_transfer_items_transfer ON stock_transfer_items(transfer_id);",

		// Milestone 1: POS Financial Shield, Live Margin Sentinel & Fast-Grid Hotkeys
		"ALTER TABLE products ADD COLUMN IF NOT EXISTS min_margin_percent DECIMAL(5, 2) DEFAULT 10.00;",
		"ALTER TABLE product_variants ADD COLUMN IF NOT EXISTS min_margin_percent DECIMAL(5, 2) DEFAULT 10.00;",
		"ALTER TABLE pos_bill_items ADD COLUMN IF NOT EXISTS variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL;",
		"CREATE INDEX IF NOT EXISTS idx_pos_bill_items_variant ON pos_bill_items(variant_id) WHERE variant_id IS NOT NULL;",
		`CREATE TABLE IF NOT EXISTS shop_hotkeys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id UUID REFERENCES product_variants(id) ON DELETE CASCADE,
			display_name VARCHAR(100) NOT NULL,
			color_hex VARCHAR(10) NOT NULL DEFAULT '#2563EB',
			sort_order INT NOT NULL DEFAULT 0,
			time_slot VARCHAR(20) NOT NULL DEFAULT 'ALL_DAY' CHECK (time_slot IN ('ALL_DAY', 'MORNING', 'EVENING')),
			is_active BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_shop_hotkeys_lookup ON shop_hotkeys(shop_id, time_slot, is_active, sort_order ASC);",

		// Milestone 2: Cash Drawer Telemetry, Shift Accountability, Mid-day Safe Drops & Petty Cash-out Tagging
		`CREATE TABLE IF NOT EXISTS cash_drawer_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			staff_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
			shift_name VARCHAR(50) NOT NULL DEFAULT 'General Shift',
			opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			closed_at TIMESTAMPTZ,
			opening_cash DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (opening_cash >= 0),
			cash_sales_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			safe_drop_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			expenses_total DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			expected_cash DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			actual_cash DECIMAL(10, 2),
			discrepancy_amount DECIMAL(10, 2) DEFAULT 0.00,
			safe_drop_threshold DECIMAL(10, 2) NOT NULL DEFAULT 10000.00,
			status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED', 'DISCREPANCY')),
			closing_notes TEXT DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_cash_drawer_sessions_shop_staff ON cash_drawer_sessions(shop_id, staff_user_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_cash_drawer_sessions_opened ON cash_drawer_sessions(shop_id, opened_at DESC);",

		`CREATE TABLE IF NOT EXISTS cash_drawer_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES cash_drawer_sessions(id) ON DELETE CASCADE,
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			staff_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			event_type VARCHAR(30) NOT NULL CHECK (event_type IN ('OPENING_FLOAT', 'CASH_SALE', 'SAFE_DROP', 'PETTY_EXPENSE', 'MANUAL_OPEN', 'BILL_VOID', 'SHIFT_CLOSE')),
			amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			bill_id UUID REFERENCES pos_bills(id) ON DELETE SET NULL,
			category VARCHAR(100) NOT NULL DEFAULT 'General',
			reason VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_cash_drawer_events_session ON cash_drawer_events(session_id, created_at ASC);",
		"CREATE INDEX IF NOT EXISTS idx_cash_drawer_events_shop_type ON cash_drawer_events(shop_id, event_type, created_at DESC);",

		// Milestone 3: Rush-Hour Scan & Go Queue Buster, Company Damage/Expiry Claim Vault, and Reverse Returns Lookup
		`CREATE TABLE IF NOT EXISTS scan_and_go_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			customer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			customer_name VARCHAR(100) NOT NULL DEFAULT 'In-Store Shopper',
			customer_phone VARCHAR(20) NOT NULL DEFAULT '',
			session_code VARCHAR(10) UNIQUE NOT NULL,
			qr_token VARCHAR(64) UNIQUE NOT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'VERIFIED_AT_COUNTER', 'CONVERTED_TO_BILL', 'EXPIRED', 'CANCELLED')),
			total_items INT NOT NULL DEFAULT 0,
			subtotal DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
			expires_at TIMESTAMPTZ NOT NULL,
			converted_bill_id UUID REFERENCES pos_bills(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_scan_and_go_shop_status ON scan_and_go_sessions(shop_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_scan_and_go_code ON scan_and_go_sessions(session_code);",
		"CREATE INDEX IF NOT EXISTS idx_scan_and_go_token ON scan_and_go_sessions(qr_token);",

		`CREATE TABLE IF NOT EXISTS scan_and_go_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL REFERENCES scan_and_go_sessions(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
			product_name VARCHAR(255) NOT NULL,
			barcode VARCHAR(100) NOT NULL DEFAULT '',
			quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
			unit_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			total_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_scan_and_go_items_sess ON scan_and_go_items(session_id);",

		`CREATE TABLE IF NOT EXISTS damage_claims (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			product_id UUID REFERENCES products(id) ON DELETE SET NULL,
			variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
			product_name VARCHAR(255) NOT NULL,
			barcode VARCHAR(100) NOT NULL DEFAULT '',
			batch_number VARCHAR(100) NOT NULL DEFAULT '',
			expiry_date DATE,
			quantity INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
			unit_cost DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			total_loss DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			claim_type VARCHAR(50) NOT NULL DEFAULT 'EXPIRED' CHECK (claim_type IN ('EXPIRED', 'TRANSIT_DAMAGE', 'LEAKAGE_SPILL', 'RAT_PEST_BITE', 'FACTORY_DEFECT', 'OTHER')),
			distributor_name VARCHAR(255) NOT NULL DEFAULT '',
			distributor_phone VARCHAR(50) NOT NULL DEFAULT '',
			sales_rep_name VARCHAR(255) NOT NULL DEFAULT '',
			photo_url TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'PENDING_CLAIM' CHECK (status IN ('PENDING_CLAIM', 'SUBMITTED_TO_SR', 'REPLACED', 'CREDIT_NOTE_ISSUED', 'REJECTED')),
			resolution_notes TEXT NOT NULL DEFAULT '',
			resolved_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_damage_claims_shop_distributor ON damage_claims(shop_id, distributor_name);",
		"CREATE INDEX IF NOT EXISTS idx_damage_claims_shop_status ON damage_claims(shop_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_damage_claims_product ON damage_claims(product_id);",

		"CREATE INDEX IF NOT EXISTS idx_pos_bills_phone ON pos_bills(customer_phone) WHERE customer_phone IS NOT NULL AND customer_phone != '';",
		"CREATE INDEX IF NOT EXISTS idx_pos_bill_items_product ON pos_bill_items(product_id);",

		// Milestone 4: Product Batches, Near-Expiry Auto-Markdown Engine, and Khata Daily Micro-Installments
		`CREATE TABLE IF NOT EXISTS product_batches (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			variant_id UUID REFERENCES product_variants(id) ON DELETE SET NULL,
			batch_number VARCHAR(100) NOT NULL,
			mfg_date DATE,
			expiry_date DATE NOT NULL,
			quantity INT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
			cost_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			original_selling_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			current_selling_price DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			markdown_percent DECIMAL(5, 2) NOT NULL DEFAULT 0.00,
			is_markdown_applied BOOLEAN NOT NULL DEFAULT false,
			status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'NEAR_EXPIRY', 'EXPIRED', 'SOLD_OUT')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_product_batches_shop_expiry ON product_batches(shop_id, expiry_date ASC);",
		"CREATE INDEX IF NOT EXISTS idx_product_batches_product ON product_batches(product_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_product_batches_status ON product_batches(shop_id, status);",

		`CREATE TABLE IF NOT EXISTS khata_micro_plans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			customer_id UUID REFERENCES users(id) ON DELETE SET NULL,
			customer_phone VARCHAR(20) NOT NULL,
			customer_name VARCHAR(100) NOT NULL DEFAULT 'Khata Customer',
			total_debt DECIMAL(10, 2) NOT NULL CHECK (total_debt > 0),
			daily_amount DECIMAL(10, 2) NOT NULL CHECK (daily_amount > 0),
			frequency VARCHAR(20) NOT NULL DEFAULT 'DAILY' CHECK (frequency IN ('DAILY', 'WEEKLY', 'MONTHLY')),
			total_installments INT NOT NULL CHECK (total_installments > 0),
			paid_installments INT NOT NULL DEFAULT 0,
			total_paid_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
			remaining_balance DECIMAL(10, 2) NOT NULL,
			next_due_date DATE NOT NULL,
			status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'PAUSED', 'DEFAULTED')),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_khata_micro_plans_shop ON khata_micro_plans(shop_id, status);",
		"CREATE INDEX IF NOT EXISTS idx_khata_micro_plans_phone ON khata_micro_plans(customer_phone);",

		`CREATE TABLE IF NOT EXISTS khata_micro_payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			plan_id UUID NOT NULL REFERENCES khata_micro_plans(id) ON DELETE CASCADE,
			shop_id UUID NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
			installment_number INT NOT NULL,
			amount DECIMAL(10, 2) NOT NULL CHECK (amount > 0),
			payment_mode VARCHAR(20) NOT NULL DEFAULT 'CASH' CHECK (payment_mode IN ('CASH', 'UPI', 'QR')),
			transaction_reference VARCHAR(100) DEFAULT '',
			notes TEXT DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		"CREATE INDEX IF NOT EXISTS idx_khata_micro_payments_plan ON khata_micro_payments(plan_id, installment_number);",
	}

	for _, q := range queries {
		_, _ = pool.Exec(ctx, q)
	}
}

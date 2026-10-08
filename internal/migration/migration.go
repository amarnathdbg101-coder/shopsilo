package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunAutoMigrations(dbURL string) error {
	m, err := migrate.New("file://internal/migration", dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

// EnsureSchemaColumns runs fast, idempotent ALTER TABLE statements to guarantee all critical columns exist.
func EnsureSchemaColumns(pool *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

		// Khata table enhancements
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS customer_id UUID;",
		"ALTER TABLE customer_khata ADD COLUMN IF NOT EXISTS customer_mobile VARCHAR(20);",
		"UPDATE customer_khata SET customer_mobile = customer_phone WHERE customer_mobile IS NULL OR customer_mobile = '';",
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

		// Performance Indexes (Trigram GIN Indexes & B-tree composite indexes for fast catalog search)
		"CREATE EXTENSION IF NOT EXISTS pg_trgm;",
		"CREATE INDEX IF NOT EXISTS idx_products_trgm ON products USING gin (name gin_trgm_ops, description gin_trgm_ops, sku gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_shops_trgm ON shops USING gin (name gin_trgm_ops, city gin_trgm_ops, category gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_categories_trgm ON categories USING gin (name gin_trgm_ops, slug gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_products_shop_active ON products(shop_id, is_active);",
		"CREATE INDEX IF NOT EXISTS idx_pos_bills_shop_created ON pos_bills(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_khata_tx_shop_created ON khata_transactions(shop_id, created_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_reservations_user_active ON reservations(user_id, status, expires_at);",

		`INSERT INTO categories (name, slug, description, icon, is_active) VALUES
		('Kirana & Grocery', 'kirana-grocery', 'Daily essentials, rice, flour, oil, pulses and spices', '🛒', true),
		('Dairy & Breakfast', 'dairy-breakfast', 'Milk, curd, butter, paneer, eggs and bread', '🥛', true),
		('Snacks & Beverages', 'snacks-beverages', 'Biscuits, namkeen, cold drinks, tea and coffee', '🍪', true),
		('Fruits & Vegetables', 'fruits-vegetables', 'Fresh local farm fruits and green vegetables', '🥦', true),
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

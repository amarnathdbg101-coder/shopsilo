// Package repository handles database queries for hotkeys and margin bounds.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrHotkeyNotFound = errors.New("hotkey not found")
)

// ItemMarginBounds holds raw financial limits for pricing safety checks.
type ItemMarginBounds struct {
	ID               string
	Name             string
	SKU              string
	Price            float64
	CostPrice        float64
	FloorPrice       float64
	MinMarginPercent float64
	IsVariant        bool
}

// HotkeyRepo manages fast-grid tiles and price bounds queries.
type HotkeyRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewHotkeyRepo initializes HotkeyRepo.
func NewHotkeyRepo(db *pgxpool.Pool, logger *zap.Logger) *HotkeyRepo {
	return &HotkeyRepo{
		db:     db,
		logger: logger,
	}
}

// GetItemMarginBounds fetches exact financial constraints for a product or child variant.
func (r *HotkeyRepo) GetItemMarginBounds(ctx context.Context, shopID, productID string, variantID *string) (*ItemMarginBounds, error) {
	// If variantID is provided, check the specific variant first
	if variantID != nil && strings.TrimSpace(*variantID) != "" {
		query := `
			SELECT pv.id, pv.title, pv.sku, pv.price, pv.cost_price, pv.floor_price, 
			       COALESCE(pv.min_margin_percent, 10.00)
			FROM product_variants pv
			WHERE pv.id = $1 AND pv.shop_id = $2 AND pv.is_active = true
			LIMIT 1;
		`
		var b ItemMarginBounds
		err := r.db.QueryRow(ctx, query, *variantID, shopID).Scan(
			&b.ID, &b.Name, &b.SKU, &b.Price, &b.CostPrice, &b.FloorPrice, &b.MinMarginPercent,
		)
		if err == nil {
			b.IsVariant = true
			return &b, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("failed to query variant margin bounds", zap.Error(err), zap.String("variant_id", *variantID))
			return nil, err
		}
		// Fallback to parent product if variant not found
	}

	// Query parent product
	query := `
		SELECT p.id, p.name, p.sku, p.price, COALESCE(p.cost_price, 0), COALESCE(p.floor_price, 0),
		       COALESCE(p.min_margin_percent, 10.00)
		FROM products p
		WHERE p.id = $1 AND p.shop_id = $2 AND p.is_active = true
		LIMIT 1;
	`
	var b ItemMarginBounds
	err := r.db.QueryRow(ctx, query, productID, shopID).Scan(
		&b.ID, &b.Name, &b.SKU, &b.Price, &b.CostPrice, &b.FloorPrice, &b.MinMarginPercent,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		r.logger.Error("failed to query product margin bounds", zap.Error(err), zap.String("product_id", productID))
		return nil, err
	}
	b.IsVariant = false
	return &b, nil
}

// ListByShop retrieves all fast-grid tiles for a shop, optionally filtered by time slot.
func (r *HotkeyRepo) ListByShop(ctx context.Context, shopID string, timeSlot string) ([]*model.ShopHotkey, error) {
	timeSlotClause := ""
	args := []interface{}{shopID}

	if timeSlot != "" && !strings.EqualFold(timeSlot, "ALL_DAY") {
		timeSlotClause = " AND (h.time_slot = $2 OR h.time_slot = 'ALL_DAY')"
		args = append(args, timeSlot)
	}

	query := fmt.Sprintf(`
		SELECT h.id, h.shop_id, h.product_id, h.variant_id, h.display_name, h.color_hex,
		       h.sort_order, h.time_slot, h.is_active, h.created_at, h.updated_at,
		       COALESCE(pv.price, p.price) AS price,
		       COALESCE(pv.cost_price, p.cost_price, 0) AS cost_price,
		       COALESCE(pv.floor_price, p.floor_price, 0) AS floor_price,
		       COALESCE(pv.sku, p.sku) AS sku
		FROM shop_hotkeys h
		JOIN products p ON p.id = h.product_id
		LEFT JOIN product_variants pv ON pv.id = h.variant_id
		WHERE h.shop_id = $1 AND h.is_active = true %s
		ORDER BY h.sort_order ASC, h.created_at ASC;
	`, timeSlotClause)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to list shop hotkeys", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	hotkeys := make([]*model.ShopHotkey, 0)
	for rows.Next() {
		h := &model.ShopHotkey{}
		err := rows.Scan(
			&h.ID, &h.ShopID, &h.ProductID, &h.VariantID, &h.DisplayName, &h.ColorHex,
			&h.SortOrder, &h.TimeSlot, &h.IsActive, &h.CreatedAt, &h.UpdatedAt,
			&h.Price, &h.CostPrice, &h.FloorPrice, &h.SKU,
		)
		if err != nil {
			r.logger.Error("failed to scan shop hotkey", zap.Error(err))
			return nil, err
		}
		hotkeys = append(hotkeys, h)
	}
	return hotkeys, nil
}

// Create inserts a new fast-grid tile.
func (r *HotkeyRepo) Create(ctx context.Context, h *model.ShopHotkey) (*model.ShopHotkey, error) {
	if h.ColorHex == "" {
		h.ColorHex = "#2563EB"
	}
	if h.TimeSlot == "" {
		h.TimeSlot = "ALL_DAY"
	}

	query := `
		INSERT INTO shop_hotkeys (
			shop_id, product_id, variant_id, display_name, color_hex, sort_order, time_slot, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`
	err := r.db.QueryRow(
		ctx, query,
		h.ShopID, h.ProductID, h.VariantID, h.DisplayName, h.ColorHex, h.SortOrder, h.TimeSlot, h.IsActive,
	).Scan(&h.ID, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to create shop hotkey", zap.Error(err), zap.String("shop_id", h.ShopID))
		return nil, err
	}
	return h, nil
}

// Delete removes a hotkey tile.
func (r *HotkeyRepo) Delete(ctx context.Context, shopID, id string) error {
	query := `DELETE FROM shop_hotkeys WHERE id = $1 AND shop_id = $2;`
	tag, err := r.db.Exec(ctx, query, id, shopID)
	if err != nil {
		r.logger.Error("failed to delete shop hotkey", zap.Error(err), zap.String("id", id))
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrHotkeyNotFound
	}
	return nil
}

// AutoPopulateFromTopSellers generates fast-grid tiles from bestsellers in the last 14 days.
func (r *HotkeyRepo) AutoPopulateFromTopSellers(ctx context.Context, shopID string, limit int) ([]*model.ShopHotkey, error) {
	if limit <= 0 || limit > 30 {
		limit = 12
	}

	query := `
		WITH top_items AS (
			SELECT pbi.product_id, pbi.variant_id, pbi.product_name, SUM(pbi.quantity) as total_qty
			FROM pos_bill_items pbi
			JOIN pos_bills pb ON pb.id = pbi.bill_id
			WHERE pb.shop_id = $1 AND pb.created_at >= NOW() - INTERVAL '14 days'
			  AND pbi.product_id IS NOT NULL
			GROUP BY pbi.product_id, pbi.variant_id, pbi.product_name
			ORDER BY total_qty DESC
			LIMIT $2
		)
		INSERT INTO shop_hotkeys (shop_id, product_id, variant_id, display_name, color_hex, sort_order, time_slot, is_active)
		SELECT $1, t.product_id, t.variant_id, SUBSTRING(t.product_name, 1, 30), 
		       CASE (ROW_NUMBER() OVER ()) % 4 
		           WHEN 0 THEN '#2563EB' 
		           WHEN 1 THEN '#059669' 
		           WHEN 2 THEN '#D97706' 
		           ELSE '#7C3AED' 
		       END,
		       (ROW_NUMBER() OVER ())::int,
		       'ALL_DAY', true
		FROM top_items t
		WHERE NOT EXISTS (
			SELECT 1 FROM shop_hotkeys existing 
			WHERE existing.shop_id = $1 AND existing.product_id = t.product_id 
			  AND (existing.variant_id = t.variant_id OR (existing.variant_id IS NULL AND t.variant_id IS NULL))
		)
		ON CONFLICT DO NOTHING;
	`
	_, err := r.db.Exec(ctx, query, shopID, limit)
	if err != nil {
		r.logger.Error("failed to auto-populate shop hotkeys", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}

	return r.ListByShop(ctx, shopID, "ALL_DAY")
}

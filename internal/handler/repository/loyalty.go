// Package repository handle db query.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type LoyaltyRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewLoyaltyRepo(db *pgxpool.Pool, logger *zap.Logger) *LoyaltyRepo {
	return &LoyaltyRepo{
		db:     db,
		logger: logger,
	}
}

func (r *LoyaltyRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// RecordReturnWithTx restocks product inventory, deducts earned points from customer, and logs return.
func (r *LoyaltyRepo) RecordReturnWithTx(ctx context.Context, tx pgx.Tx, ret *model.ProductReturn) error {
	query := `
		INSERT INTO product_returns (shop_id, user_id, product_id, reservation_id, quantity, refund_amount, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at
	`
	err := tx.QueryRow(
		ctx,
		query,
		ret.ShopID,
		ret.UserID,
		ret.ProductID,
		ret.ReservationID,
		ret.Quantity,
		ret.RefundAmount,
		strings.TrimSpace(ret.Reason),
	).Scan(&ret.ID, &ret.CreatedAt)
	if err != nil {
		r.logger.Error("failed to insert product return", zap.Error(err))
		return err
	}

	// 1. Restock inventory
	invQuery := `
		UPDATE inventory
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE product_id = $2
	`
	if _, err := tx.Exec(ctx, invQuery, ret.Quantity, ret.ProductID); err != nil {
		r.logger.Error("failed to restock returned item", zap.Error(err), zap.String("product_id", ret.ProductID))
		return err
	}

	// 2. Adjust loyalty points if customer ID is attached
	if ret.UserID != nil && *ret.UserID != "" && ret.RefundAmount > 0 {
		pointsToDeduct := int(ret.RefundAmount / 20)
		if pointsToDeduct > 0 {
			userQuery := `UPDATE users SET loyalty_points = GREATEST(0, loyalty_points - $1) WHERE id = $2`
			_, _ = tx.Exec(ctx, userQuery, pointsToDeduct, *ret.UserID)
		}
	}

	return nil
}

func (r *LoyaltyRepo) ListShopReturns(ctx context.Context, shopID string) ([]*model.ProductReturn, error) {
	query := `
		SELECT r.id, r.shop_id, r.user_id, r.product_id, r.reservation_id, r.quantity, r.refund_amount,
		       COALESCE(r.reason, ''), r.created_at, p.name AS product_name
		FROM product_returns r
		JOIN products p ON r.product_id = p.id
		WHERE r.shop_id = $1
		ORDER BY r.created_at DESC
		LIMIT 50
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to list shop returns", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*model.ProductReturn
	for rows.Next() {
		ret := &model.ProductReturn{}
		err := rows.Scan(
			&ret.ID,
			&ret.ShopID,
			&ret.UserID,
			&ret.ProductID,
			&ret.ReservationID,
			&ret.Quantity,
			&ret.RefundAmount,
			&ret.Reason,
			&ret.CreatedAt,
			&ret.ProductName,
		)
		if err != nil {
			r.logger.Error("failed to scan return row", zap.Error(err))
			return nil, err
		}
		list = append(list, ret)
	}

	return list, nil
}

func (r *LoyaltyRepo) CreateOffer(ctx context.Context, off *model.StoreOffer) (*model.StoreOffer, error) {
	query := `
		INSERT INTO store_offers (shop_id, title, description, discount_text, min_points_required, is_active, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, NOW())
		RETURNING id, shop_id, title, COALESCE(description, ''), discount_text, min_points_required, is_active, expires_at, created_at
	`
	created := &model.StoreOffer{}
	err := r.db.QueryRow(
		ctx,
		query,
		off.ShopID,
		strings.TrimSpace(off.Title),
		strings.TrimSpace(off.Description),
		strings.TrimSpace(off.DiscountText),
		off.MinPointsRequired,
		off.ExpiresAt,
	).Scan(
		&created.ID,
		&created.ShopID,
		&created.Title,
		&created.Description,
		&created.DiscountText,
		&created.MinPointsRequired,
		&created.IsActive,
		&created.ExpiresAt,
		&created.CreatedAt,
	)
	if err != nil {
		r.logger.Error("failed to create store offer", zap.Error(err), zap.String("shop_id", off.ShopID))
		return nil, err
	}

	return created, nil
}

func (r *LoyaltyRepo) ListOffers(ctx context.Context, shopID string, userPoints int) ([]*model.StoreOffer, error) {
	query := `
		SELECT id, shop_id, title, COALESCE(description, ''), discount_text, min_points_required, is_active, expires_at, created_at
		FROM store_offers
		WHERE shop_id = $1 AND is_active = true AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY min_points_required DESC, created_at DESC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to list offers", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var offers []*model.StoreOffer
	for rows.Next() {
		off := &model.StoreOffer{}
		err := rows.Scan(
			&off.ID,
			&off.ShopID,
			&off.Title,
			&off.Description,
			&off.DiscountText,
			&off.MinPointsRequired,
			&off.IsActive,
			&off.ExpiresAt,
			&off.CreatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan offer row", zap.Error(err))
			return nil, err
		}
		off.IsUnlocked = userPoints >= off.MinPointsRequired
		offers = append(offers, off)
	}

	return offers, nil
}

func (r *LoyaltyRepo) GetUserLoyalty(ctx context.Context, userID string) (*model.LoyaltySummary, error) {
	query := `SELECT COALESCE(name, 'Customer'), COALESCE(loyalty_points, 0) FROM users WHERE id = $1`
	var name string
	var points int
	err := r.db.QueryRow(ctx, query, userID).Scan(&name, &points)
	if err != nil {
		return nil, err
	}

	tier := "Bronze"
	nextPoints := 100 - points
	if points >= 300 {
		tier = "Gold VIP"
		nextPoints = 0
	} else if points >= 100 {
		tier = "Silver"
		nextPoints = 300 - points
	}

	return &model.LoyaltySummary{
		UserID:               userID,
		UserName:             name,
		Points:               points,
		Tier:                 tier,
		NextTierPointsNeeded: nextPoints,
	}, nil
}

func (r *LoyaltyRepo) AddLoyaltyPoints(ctx context.Context, tx pgx.Tx, userID string, points int) error {
	query := `UPDATE users SET loyalty_points = loyalty_points + $1 WHERE id = $2`
	_, err := tx.Exec(ctx, query, points, userID)
	return err
}

func (r *LoyaltyRepo) FindProductByBarcode(ctx context.Context, code string) (*model.Product, error) {
	query := `
		SELECT p.id, p.shop_id, p.name, p.slug, COALESCE(p.description, ''), p.sku, p.price, COALESCE(p.compare_price, 0),
		       p.category_id, p.images, COALESCE(p.weight, 0), p.is_active, p.is_featured, p.tags, p.created_at, p.updated_at,
		       COALESCE(i.quantity, 0), COALESCE(i.reserved_quantity, 0), COALESCE(i.low_stock_threshold, 10)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE (p.sku ILIKE $1 OR p.slug = LOWER(TRIM($1))) AND p.is_active = true
		LIMIT 1
	`
	p := &model.Product{}
	var imagesBytes []byte
	inv := &model.Inventory{}

	err := r.db.QueryRow(ctx, query, strings.TrimSpace(code)).Scan(
		&p.ID,
		&p.ShopID,
		&p.Name,
		&p.Slug,
		&p.Description,
		&p.SKU,
		&p.Price,
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
		r.logger.Error("failed to scan product by barcode", zap.Error(err), zap.String("code", code))
		return nil, err
	}

	p.Images = []string{}
	if len(imagesBytes) > 0 {
		_ = json.Unmarshal(imagesBytes, &p.Images)
	}

	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	p.Inventory = inv

	return p, nil
}

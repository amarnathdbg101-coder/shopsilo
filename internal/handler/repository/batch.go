// Package repository handles database queries for inventory batch expiry and auto-markdowns.
package repository

import (
	"context"
	"errors"
	"math"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrBatchNotFound = errors.New("product batch not found")
)

// BatchRepo manages database queries for product batches and expiry radars.
type BatchRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewBatchRepo initializes BatchRepo.
func NewBatchRepo(db *pgxpool.Pool, logger *zap.Logger) *BatchRepo {
	return &BatchRepo{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new inventory batch lot.
func (r *BatchRepo) Create(ctx context.Context, b *model.ProductBatch) (*model.ProductBatch, error) {
	if b.CurrentSellingPrice <= 0 {
		b.CurrentSellingPrice = b.OriginalSellingPrice
	}
	if b.Status == "" {
		b.Status = model.BatchStatusActive
	}

	query := `
		INSERT INTO product_batches (
			shop_id, product_id, variant_id, batch_number, mfg_date, expiry_date,
			quantity, cost_price, original_selling_price, current_selling_price,
			markdown_percent, is_markdown_applied, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`
	err := r.db.QueryRow(ctx, query,
		b.ShopID, b.ProductID, b.VariantID, b.BatchNumber, b.MfgDate, b.ExpiryDate,
		b.Quantity, b.CostPrice, b.OriginalSellingPrice, b.CurrentSellingPrice,
		b.MarkdownPercent, b.IsMarkdownApplied, b.Status,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to insert product batch", zap.Error(err), zap.String("shop_id", b.ShopID))
		return nil, err
	}
	return b, nil
}

// GetByID finds a batch by its UUID.
func (r *BatchRepo) GetByID(ctx context.Context, batchID string) (*model.ProductBatch, error) {
	query := `
		SELECT b.id, b.shop_id, b.product_id, COALESCE(p.name, ''), b.variant_id, b.batch_number,
		       b.mfg_date, b.expiry_date, b.quantity, b.cost_price, b.original_selling_price,
		       b.current_selling_price, b.markdown_percent, b.is_markdown_applied, b.status,
		       b.created_at, b.updated_at
		FROM product_batches b
		JOIN products p ON p.id = b.product_id
		WHERE b.id = $1;
	`
	b := &model.ProductBatch{}
	err := r.db.QueryRow(ctx, query, batchID).Scan(
		&b.ID, &b.ShopID, &b.ProductID, &b.ProductName, &b.VariantID, &b.BatchNumber,
		&b.MfgDate, &b.ExpiryDate, &b.Quantity, &b.CostPrice, &b.OriginalSellingPrice,
		&b.CurrentSellingPrice, &b.MarkdownPercent, &b.IsMarkdownApplied, &b.Status,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBatchNotFound
		}
		return nil, err
	}
	return b, nil
}

// ListNearExpiry finds all active batches expiring within the given days window (e.g. 30 days).
func (r *BatchRepo) ListNearExpiry(ctx context.Context, shopID string, daysWindow int) ([]*model.ProductBatch, error) {
	if daysWindow <= 0 {
		daysWindow = 30
	}

	query := `
		SELECT b.id, b.shop_id, b.product_id, COALESCE(p.name, ''), b.variant_id, b.batch_number,
		       b.mfg_date, b.expiry_date, b.quantity, b.cost_price, b.original_selling_price,
		       b.current_selling_price, b.markdown_percent, b.is_markdown_applied, b.status,
		       b.created_at, b.updated_at
		FROM product_batches b
		JOIN products p ON p.id = b.product_id
		WHERE b.shop_id = $1
		  AND b.quantity > 0
		  AND b.expiry_date <= CURRENT_DATE + ($2 || ' days')::interval
		ORDER BY b.expiry_date ASC;
	`
	rows, err := r.db.Query(ctx, query, shopID, daysWindow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	batches := make([]*model.ProductBatch, 0)
	for rows.Next() {
		b := &model.ProductBatch{}
		err := rows.Scan(
			&b.ID, &b.ShopID, &b.ProductID, &b.ProductName, &b.VariantID, &b.BatchNumber,
			&b.MfgDate, &b.ExpiryDate, &b.Quantity, &b.CostPrice, &b.OriginalSellingPrice,
			&b.CurrentSellingPrice, &b.MarkdownPercent, &b.IsMarkdownApplied, &b.Status,
			&b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, nil
}

// RunAutoMarkdownEngine scans batches, expires past-dated ones, and discounts near-expiry lots.
func (r *BatchRepo) RunAutoMarkdownEngine(ctx context.Context, shopID string, daysThreshold int, defaultMarkdownPercent float64) (scanned int, markdownsApplied int, expiredCount int, capitalProtected float64, updated []*model.ProductBatch, err error) {
	if daysThreshold <= 0 {
		daysThreshold = 30
	}
	if defaultMarkdownPercent <= 0 {
		defaultMarkdownPercent = 20.00
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}
	defer tx.Rollback(ctx)

	// Fetch all candidate batches
	query := `
		SELECT b.id, b.shop_id, b.product_id, COALESCE(p.name, ''), b.variant_id, b.batch_number,
		       b.mfg_date, b.expiry_date, b.quantity, b.cost_price, b.original_selling_price,
		       b.current_selling_price, b.markdown_percent, b.is_markdown_applied, b.status
		FROM product_batches b
		JOIN products p ON p.id = b.product_id
		WHERE b.shop_id = $1 AND b.quantity > 0 AND b.status != 'SOLD_OUT'
		FOR UPDATE;
	`
	rows, err := tx.Query(ctx, query, shopID)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}
	defer rows.Close()

	now := time.Now().Truncate(24 * time.Hour)
	thresholdDate := now.Add(time.Duration(daysThreshold) * 24 * time.Hour)

	var candidates []*model.ProductBatch
	for rows.Next() {
		b := &model.ProductBatch{}
		err := rows.Scan(
			&b.ID, &b.ShopID, &b.ProductID, &b.ProductName, &b.VariantID, &b.BatchNumber,
			&b.MfgDate, &b.ExpiryDate, &b.Quantity, &b.CostPrice, &b.OriginalSellingPrice,
			&b.CurrentSellingPrice, &b.MarkdownPercent, &b.IsMarkdownApplied, &b.Status,
		)
		if err != nil {
			return 0, 0, 0, 0, nil, err
		}
		candidates = append(candidates, b)
	}
	rows.Close()

	scanned = len(candidates)
	updated = make([]*model.ProductBatch, 0)

	for _, b := range candidates {
		exp := b.ExpiryDate.Truncate(24 * time.Hour)

		// 1. Expired Lot
		if exp.Before(now) {
			expiredCount++
			b.Status = model.BatchStatusExpired
			_, err = tx.Exec(ctx, `
				UPDATE product_batches
				SET status = 'EXPIRED', updated_at = NOW()
				WHERE id = $1;
			`, b.ID)
			if err != nil {
				return 0, 0, 0, 0, nil, err
			}
			updated = append(updated, b)
			continue
		}

		// 2. Near-Expiry Lot (within threshold e.g. <= 30 days)
		if !exp.After(thresholdDate) {
			discountPercent := b.MarkdownPercent
			if discountPercent <= 0 {
				discountPercent = defaultMarkdownPercent
			}

			// Apply markdown if not already applied
			if !b.IsMarkdownApplied {
				// Calculate discounted selling price (never below cost price)
				discMultiplier := (100.0 - discountPercent) / 100.0
				discountedPrice := math.Round(b.OriginalSellingPrice*discMultiplier*100) / 100
				if discountedPrice < b.CostPrice {
					discountedPrice = b.CostPrice // Preserve break-even capital
				}

				b.CurrentSellingPrice = discountedPrice
				b.MarkdownPercent = discountPercent
				b.IsMarkdownApplied = true
				b.Status = model.BatchStatusNearExpiry

				_, err = tx.Exec(ctx, `
					UPDATE product_batches
					SET current_selling_price = $1,
					    markdown_percent = $2,
					    is_markdown_applied = true,
					    status = 'NEAR_EXPIRY',
					    updated_at = NOW()
					WHERE id = $3;
				`, discountedPrice, discountPercent, b.ID)
				if err != nil {
					return 0, 0, 0, 0, nil, err
				}

				markdownsApplied++
				capitalProtected += float64(b.Quantity) * b.CostPrice
				updated = append(updated, b)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, 0, nil, err
	}
	return scanned, markdownsApplied, expiredCount, capitalProtected, updated, nil
}


// Package repository handles database operations for damage, leakage, and expiry claims.
package repository

import (
	"context"
	"errors"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrDamageClaimNotFound = errors.New("damage claim record not found")
)

// DamageClaimRepo manages the damage and expiry claims vault.
type DamageClaimRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewDamageClaimRepo initializes DamageClaimRepo.
func NewDamageClaimRepo(db *pgxpool.Pool, logger *zap.Logger) *DamageClaimRepo {
	return &DamageClaimRepo{
		db:     db,
		logger: logger,
	}
}

// Create inserts a newly reported damaged or expired product claim.
func (r *DamageClaimRepo) Create(ctx context.Context, c *model.DamageClaim) (*model.DamageClaim, error) {
	c.TotalLoss = float64(c.Quantity) * c.UnitCost
	if c.Status == "" {
		c.Status = model.ClaimStatusPendingClaim
	}

	query := `
		INSERT INTO damage_claims (
			shop_id, product_id, variant_id, product_name, barcode, batch_number, expiry_date,
			quantity, unit_cost, total_loss, claim_type, distributor_name, distributor_phone,
			sales_rep_name, photo_url, status, resolution_notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`
	err := r.db.QueryRow(ctx, query,
		c.ShopID, c.ProductID, c.VariantID, c.ProductName, c.Barcode, c.BatchNumber, c.ExpiryDate,
		c.Quantity, c.UnitCost, c.TotalLoss, c.ClaimType, c.DistributorName, c.DistributorPhone,
		c.SalesRepName, c.PhotoURL, c.Status, c.ResolutionNotes,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create damage claim", zap.Error(err), zap.String("shop_id", c.ShopID))
		return nil, err
	}
	return c, nil
}

// GetByID retrieves a single claim by UUID.
func (r *DamageClaimRepo) GetByID(ctx context.Context, claimID string) (*model.DamageClaim, error) {
	query := `
		SELECT id, shop_id, product_id, variant_id, product_name, barcode, batch_number, expiry_date,
		       quantity, unit_cost, total_loss, claim_type, distributor_name, distributor_phone,
		       sales_rep_name, photo_url, status, resolution_notes, resolved_at, created_at, updated_at
		FROM damage_claims
		WHERE id = $1;
	`
	c := &model.DamageClaim{}
	err := r.db.QueryRow(ctx, query, claimID).Scan(
		&c.ID, &c.ShopID, &c.ProductID, &c.VariantID, &c.ProductName, &c.Barcode, &c.BatchNumber, &c.ExpiryDate,
		&c.Quantity, &c.UnitCost, &c.TotalLoss, &c.ClaimType, &c.DistributorName, &c.DistributorPhone,
		&c.SalesRepName, &c.PhotoURL, &c.Status, &c.ResolutionNotes, &c.ResolvedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDamageClaimNotFound
		}
		return nil, err
	}
	return c, nil
}

// UpdateStatus changes claim status (e.g. SR settled with replacement or credit note).
func (r *DamageClaimRepo) UpdateStatus(ctx context.Context, claimID string, status model.ClaimStatus, notes string) (*model.DamageClaim, error) {
	var resolvedAt *time.Time
	if status == model.ClaimStatusReplaced || status == model.ClaimStatusCreditNoteIssued || status == model.ClaimStatusRejected {
		now := time.Now()
		resolvedAt = &now
	}

	query := `
		UPDATE damage_claims
		SET status = $1, resolution_notes = $2, resolved_at = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, shop_id, product_id, variant_id, product_name, barcode, batch_number, expiry_date,
		          quantity, unit_cost, total_loss, claim_type, distributor_name, distributor_phone,
		          sales_rep_name, photo_url, status, resolution_notes, resolved_at, created_at, updated_at;
	`
	c := &model.DamageClaim{}
	err := r.db.QueryRow(ctx, query, status, notes, resolvedAt, claimID).Scan(
		&c.ID, &c.ShopID, &c.ProductID, &c.VariantID, &c.ProductName, &c.Barcode, &c.BatchNumber, &c.ExpiryDate,
		&c.Quantity, &c.UnitCost, &c.TotalLoss, &c.ClaimType, &c.DistributorName, &c.DistributorPhone,
		&c.SalesRepName, &c.PhotoURL, &c.Status, &c.ResolutionNotes, &c.ResolvedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDamageClaimNotFound
		}
		return nil, err
	}
	return c, nil
}

// ListByShop queries claims for a shop with optional distributor and status filters.
func (r *DamageClaimRepo) ListByShop(ctx context.Context, shopID, distributorName, status string, limit int) ([]*model.DamageClaim, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, shop_id, product_id, variant_id, product_name, barcode, batch_number, expiry_date,
		       quantity, unit_cost, total_loss, claim_type, distributor_name, distributor_phone,
		       sales_rep_name, photo_url, status, resolution_notes, resolved_at, created_at, updated_at
		FROM damage_claims
		WHERE shop_id = $1
		  AND ($2 = '' OR distributor_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
		ORDER BY created_at DESC
		LIMIT $4;
	`
	rows, err := r.db.Query(ctx, query, shopID, distributorName, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claims := make([]*model.DamageClaim, 0)
	for rows.Next() {
		c := &model.DamageClaim{}
		err := rows.Scan(
			&c.ID, &c.ShopID, &c.ProductID, &c.VariantID, &c.ProductName, &c.Barcode, &c.BatchNumber, &c.ExpiryDate,
			&c.Quantity, &c.UnitCost, &c.TotalLoss, &c.ClaimType, &c.DistributorName, &c.DistributorPhone,
			&c.SalesRepName, &c.PhotoURL, &c.Status, &c.ResolutionNotes, &c.ResolvedAt, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		claims = append(claims, c)
	}
	return claims, nil
}


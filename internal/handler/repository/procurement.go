// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrProcurementItemNotFound = errors.New("procurement item not found")
)

type ProcurementRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewProcurementRepo(db *pgxpool.Pool, logger *zap.Logger) *ProcurementRepo {
	return &ProcurementRepo{
		db:     db,
		logger: logger,
	}
}

// List returns all procurement items for a shop, sorted by completed status and creation date.
func (r *ProcurementRepo) List(ctx context.Context, shopID string) ([]*model.ProcurementItem, error) {
	query := `
		SELECT id, shop_id, name, category, qty, estimated_price, actual_price, dealer_name, dealer_phone, notes, is_completed, created_at, updated_at
		FROM procurement_items
		WHERE shop_id = $1
		ORDER BY is_completed ASC, created_at DESC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to query procurement items: %w", err)
	}
	defer rows.Close()

	var items []*model.ProcurementItem
	for rows.Next() {
		var it model.ProcurementItem
		err := rows.Scan(
			&it.ID, &it.ShopID, &it.Name, &it.Category, &it.Qty,
			&it.EstimatedPrice, &it.ActualPrice, &it.DealerName, &it.DealerPhone,
			&it.Notes, &it.IsCompleted, &it.CreatedAt, &it.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan procurement item: %w", err)
		}
		items = append(items, &it)
	}
	if items == nil {
		items = []*model.ProcurementItem{}
	}
	return items, nil
}

// Create inserts a new procurement item.
func (r *ProcurementRepo) Create(ctx context.Context, shopID string, it *model.ProcurementItem) (*model.ProcurementItem, error) {
	query := `
		INSERT INTO procurement_items (
			shop_id, name, category, qty, estimated_price, actual_price, dealer_name, dealer_phone, notes, is_completed, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, shop_id, name, category, qty, estimated_price, actual_price, dealer_name, dealer_phone, notes, is_completed, created_at, updated_at
	`
	var res model.ProcurementItem
	err := r.db.QueryRow(
		ctx, query,
		shopID, it.Name, it.Category, it.Qty, it.EstimatedPrice, it.ActualPrice, it.DealerName, it.DealerPhone, it.Notes, it.IsCompleted,
	).Scan(
		&res.ID, &res.ShopID, &res.Name, &res.Category, &res.Qty,
		&res.EstimatedPrice, &res.ActualPrice, &res.DealerName, &res.DealerPhone,
		&res.Notes, &res.IsCompleted, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert procurement item: %w", err)
	}
	return &res, nil
}

// Update modifies an existing procurement item.
func (r *ProcurementRepo) Update(ctx context.Context, shopID, id string, it *model.ProcurementItem) (*model.ProcurementItem, error) {
	query := `
		UPDATE procurement_items
		SET name = $1, category = $2, qty = $3, estimated_price = $4, actual_price = $5,
		    dealer_name = $6, dealer_phone = $7, notes = $8, is_completed = $9, updated_at = NOW()
		WHERE id = $10 AND shop_id = $11
		RETURNING id, shop_id, name, category, qty, estimated_price, actual_price, dealer_name, dealer_phone, notes, is_completed, created_at, updated_at
	`
	var res model.ProcurementItem
	err := r.db.QueryRow(
		ctx, query,
		it.Name, it.Category, it.Qty, it.EstimatedPrice, it.ActualPrice,
		it.DealerName, it.DealerPhone, it.Notes, it.IsCompleted, id, shopID,
	).Scan(
		&res.ID, &res.ShopID, &res.Name, &res.Category, &res.Qty,
		&res.EstimatedPrice, &res.ActualPrice, &res.DealerName, &res.DealerPhone,
		&res.Notes, &res.IsCompleted, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update procurement item: %w", err)
	}
	return &res, nil
}

// Toggle flips the is_completed status of a procurement item.
func (r *ProcurementRepo) Toggle(ctx context.Context, shopID, id string) (*model.ProcurementItem, error) {
	query := `
		UPDATE procurement_items
		SET is_completed = NOT is_completed, updated_at = NOW()
		WHERE id = $1 AND shop_id = $2
		RETURNING id, shop_id, name, category, qty, estimated_price, actual_price, dealer_name, dealer_phone, notes, is_completed, created_at, updated_at
	`
	var res model.ProcurementItem
	err := r.db.QueryRow(ctx, query, id, shopID).Scan(
		&res.ID, &res.ShopID, &res.Name, &res.Category, &res.Qty,
		&res.EstimatedPrice, &res.ActualPrice, &res.DealerName, &res.DealerPhone,
		&res.Notes, &res.IsCompleted, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle procurement item: %w", err)
	}
	return &res, nil
}

// Delete removes an item from the procurement list.
func (r *ProcurementRepo) Delete(ctx context.Context, shopID, id string) error {
	res, err := r.db.Exec(ctx, "DELETE FROM procurement_items WHERE id = $1 AND shop_id = $2", id, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete procurement item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrProcurementItemNotFound
	}
	return nil
}

// ClearCompleted removes all completed items from the procurement list.
func (r *ProcurementRepo) ClearCompleted(ctx context.Context, shopID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM procurement_items WHERE shop_id = $1 AND is_completed = true", shopID)
	if err != nil {
		return fmt.Errorf("failed to clear completed procurement items: %w", err)
	}
	return nil
}

// ListWholesalers retrieves all registered wholesalers for a shop sorted alphabetically by name.
func (r *ProcurementRepo) ListWholesalers(ctx context.Context, shopID string) ([]*model.Wholesaler, error) {
	query := `
		SELECT id, shop_id, name, phone, address, gstin, notes, created_at, updated_at
		FROM wholesalers
		WHERE shop_id = $1
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to query wholesalers: %w", err)
	}
	defer rows.Close()

	var list []*model.Wholesaler
	for rows.Next() {
		var w model.Wholesaler
		err := rows.Scan(
			&w.ID, &w.ShopID, &w.Name, &w.Phone, &w.Address,
			&w.GSTIN, &w.Notes, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wholesaler row: %w", err)
		}
		list = append(list, &w)
	}
	return list, nil
}

// CreateWholesaler inserts a new permanent wholesaler for the shop or updates contact info if already present.
func (r *ProcurementRepo) CreateWholesaler(ctx context.Context, w *model.Wholesaler) error {
	query := `
		INSERT INTO wholesalers (shop_id, name, phone, address, gstin, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (shop_id, LOWER(name)) DO UPDATE
		SET phone = CASE WHEN EXCLUDED.phone <> '' THEN EXCLUDED.phone ELSE wholesalers.phone END,
		    address = CASE WHEN EXCLUDED.address <> '' THEN EXCLUDED.address ELSE wholesalers.address END,
		    gstin = CASE WHEN EXCLUDED.gstin <> '' THEN EXCLUDED.gstin ELSE wholesalers.gstin END,
		    notes = CASE WHEN EXCLUDED.notes <> '' THEN EXCLUDED.notes ELSE wholesalers.notes END,
		    updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		w.ShopID, w.Name, w.Phone, w.Address, w.GSTIN, w.Notes,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

// UpdateWholesaler modifies an existing wholesaler record.
func (r *ProcurementRepo) UpdateWholesaler(ctx context.Context, w *model.Wholesaler) error {
	query := `
		UPDATE wholesalers
		SET name = $1, phone = $2, address = $3, gstin = $4, notes = $5, updated_at = NOW()
		WHERE id = $6 AND shop_id = $7
	`
	res, err := r.db.Exec(ctx, query, w.Name, w.Phone, w.Address, w.GSTIN, w.Notes, w.ID, w.ShopID)
	if err != nil {
		return fmt.Errorf("failed to update wholesaler: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("wholesaler not found")
	}
	return nil
}

// DeleteWholesaler removes a wholesaler from the shop directory.
func (r *ProcurementRepo) DeleteWholesaler(ctx context.Context, shopID, id string) error {
	res, err := r.db.Exec(ctx, "DELETE FROM wholesalers WHERE id = $1 AND shop_id = $2", id, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete wholesaler: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("wholesaler not found")
	}
	return nil
}

// Package repository handles database queries for no-bill reverse return lookups.
package repository

import (
	"context"
	"fmt"
	"strings"

	"shopMe/internal/handler/dto"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ReturnLookupRepo manages reverse transaction lookups for returns without receipt.
type ReturnLookupRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewReturnLookupRepo initializes ReturnLookupRepo.
func NewReturnLookupRepo(db *pgxpool.Pool, logger *zap.Logger) *ReturnLookupRepo {
	return &ReturnLookupRepo{
		db:     db,
		logger: logger,
	}
}

// FindPastPurchases searches past bill items matching product or customer identifiers within a given day window.
func (r *ReturnLookupRepo) FindPastPurchases(ctx context.Context, filter dto.ReturnLookupFilter) ([]*dto.ReturnLookupItem, error) {
	days := filter.Days
	if days <= 0 || days > 180 {
		days = 30 // Default 30 days window for retail returns
	}

	whereClauses := []string{
		"b.shop_id = $1",
		fmt.Sprintf("b.created_at >= NOW() - INTERVAL '%d days'", days),
		"b.status != 'cancelled'",
	}
	args := []any{filter.ShopID}
	argIdx := 2

	if filter.CustomerPhone != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("b.customer_phone = $%d", argIdx))
		args = append(args, filter.CustomerPhone)
		argIdx++
	}

	if filter.BillNumber != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("b.bill_number ILIKE $%d", argIdx))
		args = append(args, "%"+filter.BillNumber+"%")
		argIdx++
	}

	if filter.ProductID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("bi.product_id = $%d", argIdx))
		args = append(args, filter.ProductID)
		argIdx++
	}

	if filter.Barcode != "" {
		// Look up product by barcode
		whereClauses = append(whereClauses, fmt.Sprintf("(p.barcode = $%d OR p.sku = $%d)", argIdx, argIdx))
		args = append(args, filter.Barcode)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT b.id, b.bill_number, b.created_at, COALESCE(b.customer_name, ''), COALESCE(b.customer_phone, ''),
		       bi.product_id, bi.variant_id, bi.product_name, bi.quantity, bi.unit_price,
		       COALESCE(bi.unit_cost, 0.00), bi.total_price, b.payment_method, b.status
		FROM pos_bill_items bi
		JOIN pos_bills b ON b.id = bi.bill_id
		LEFT JOIN products p ON p.id = bi.product_id
		WHERE %s
		ORDER BY b.created_at DESC
		LIMIT 50;
	`, strings.Join(whereClauses, " AND "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query past purchases for return lookup", zap.Error(err), zap.String("shop_id", filter.ShopID))
		return nil, err
	}
	defer rows.Close()

	results := make([]*dto.ReturnLookupItem, 0)
	for rows.Next() {
		item := &dto.ReturnLookupItem{}
		err := rows.Scan(
			&item.BillID, &item.BillNumber, &item.BilledAt, &item.CustomerName, &item.CustomerPhone,
			&item.ProductID, &item.VariantID, &item.ProductName, &item.QuantitySold, &item.UnitPriceSold,
			&item.UnitCost, &item.TotalPrice, &item.PaymentMethod, &item.BillStatus,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, nil
}


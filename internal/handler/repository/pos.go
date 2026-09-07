// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/model"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrBillNotFound = errors.New("bill not found")
)

type POSRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPOSRepo(db *pgxpool.Pool, logger *zap.Logger) *POSRepo {
	return &POSRepo{
		db:     db,
		logger: logger,
	}
}

func (r *POSRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *POSRepo) CreateSaleWithTx(ctx context.Context, tx pgx.Tx, bill *model.POSBill) (*model.POSBill, int, error) {
	// 1. Check if customer phone matches a registered user for loyalty points
	var customerUserID *string
	pointsAwarded := 0

	cleanPhone := strings.TrimSpace(bill.CustomerPhone)
	if cleanPhone != "" {
		var uid string
		userQuery := `SELECT id FROM users WHERE phone = $1 OR email = $1 LIMIT 1`
		if err := tx.QueryRow(ctx, userQuery, cleanPhone).Scan(&uid); err == nil && uid != "" {
			customerUserID = &uid
			bill.CustomerUserID = &uid

			// 1 point per Rs 20
			pointsAwarded = int(bill.TotalAmount / 20)
			if pointsAwarded > 0 {
				_, _ = tx.Exec(ctx, `UPDATE users SET loyalty_points = loyalty_points + $1 WHERE id = $2`, pointsAwarded, uid)
			}
		}
	}

	// 2. Insert Bill header
	billQuery := `
		INSERT INTO pos_bills (shop_id, bill_number, customer_phone, customer_user_id, subtotal, discount_amount, total_amount, total_cost, payment_method, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id, created_at
	`
	err := tx.QueryRow(
		ctx,
		billQuery,
		bill.ShopID,
		bill.BillNumber,
		cleanPhone,
		customerUserID,
		bill.Subtotal,
		bill.DiscountAmount,
		bill.TotalAmount,
		bill.TotalCost,
		bill.PaymentMethod,
	).Scan(&bill.ID, &bill.CreatedAt)
	if err != nil {
		r.logger.Error("failed to create pos bill in tx", zap.Error(err))
		return nil, 0, err
	}

	// 3. Insert Bill items and deduct stock
	itemInsertQuery := `
		INSERT INTO pos_bill_items (bill_id, product_id, product_name, product_sku, quantity, unit_price, unit_cost, total_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	stockDeductQuery := `
		UPDATE inventory
		SET quantity = GREATEST(0, quantity - $1), updated_at = NOW()
		WHERE product_id = $2
	`

	for _, item := range bill.Items {
		item.BillID = bill.ID
		err := tx.QueryRow(
			ctx,
			itemInsertQuery,
			item.BillID,
			item.ProductID,
			item.ProductName,
			item.ProductSKU,
			item.Quantity,
			item.UnitPrice,
			item.UnitCost,
			item.TotalPrice,
		).Scan(&item.ID)
		if err != nil {
			r.logger.Error("failed to insert pos bill item", zap.Error(err))
			return nil, 0, err
		}

		// Deduct inventory
		if _, err := tx.Exec(ctx, stockDeductQuery, item.Quantity, item.ProductID); err != nil {
			r.logger.Error("failed to deduct inventory on pos sale", zap.Error(err), zap.String("product_id", item.ProductID))
			return nil, 0, err
		}
	}

	bill.NetProfit = bill.TotalAmount - bill.TotalCost
	return bill, pointsAwarded, nil
}

func (r *POSRepo) GetDailySummary(ctx context.Context, shopID string, date time.Time) (*model.DailySalesSummary, error) {
	query := `
		SELECT 
			COUNT(id) AS total_bills,
			COALESCE(SUM(total_amount), 0.0) AS total_revenue,
			COALESCE(SUM(total_cost), 0.0) AS total_cost,
			COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total_amount ELSE 0 END), 0.0) AS cash_total,
			COALESCE(SUM(CASE WHEN payment_method = 'upi' THEN total_amount ELSE 0 END), 0.0) AS upi_total,
			COALESCE(SUM(CASE WHEN payment_method = 'card' THEN total_amount ELSE 0 END), 0.0) AS card_total
		FROM pos_bills
		WHERE shop_id = $1 AND DATE(created_at) = DATE($2)
	`
	summary := &model.DailySalesSummary{
		Date: date.Format("02-Jan-2006"),
	}

	err := r.db.QueryRow(ctx, query, shopID, date).Scan(
		&summary.TotalBills,
		&summary.TotalRevenue,
		&summary.TotalCost,
		&summary.CashTotal,
		&summary.UPITotal,
		&summary.CardTotal,
	)
	if err != nil {
		r.logger.Error("failed to get daily pos summary", zap.Error(err))
		return nil, err
	}

	summary.TotalProfit = summary.TotalRevenue - summary.TotalCost
	return summary, nil
}

func (r *POSRepo) GetBillByNumber(ctx context.Context, billNumber string) (*model.POSBill, error) {
	billQuery := `
		SELECT b.id, b.shop_id, b.bill_number, COALESCE(b.customer_phone, ''), b.customer_user_id,
		       b.subtotal, b.discount_amount, b.total_amount, b.total_cost, b.payment_method, b.created_at,
		       s.name AS shop_name, s.address AS shop_address, s.phone AS shop_phone, s.whatsapp_number AS shop_whatsapp
		FROM pos_bills b
		JOIN shops s ON b.shop_id = s.id
		WHERE b.bill_number = $1
		LIMIT 1
	`
	bill := &model.POSBill{}
	bill.Shop = &model.Shop{}

	err := r.db.QueryRow(ctx, billQuery, strings.TrimSpace(billNumber)).Scan(
		&bill.ID,
		&bill.ShopID,
		&bill.BillNumber,
		&bill.CustomerPhone,
		&bill.CustomerUserID,
		&bill.Subtotal,
		&bill.DiscountAmount,
		&bill.TotalAmount,
		&bill.TotalCost,
		&bill.PaymentMethod,
		&bill.CreatedAt,
		&bill.Shop.Name,
		&bill.Shop.Address,
		&bill.Shop.Phone,
		&bill.Shop.WhatsAppNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, err
	}

	bill.NetProfit = bill.TotalAmount - bill.TotalCost

	// Fetch items
	itemsQuery := `
		SELECT id, bill_id, product_id, product_name, COALESCE(product_sku, ''), quantity, unit_price, unit_cost, total_price
		FROM pos_bill_items
		WHERE bill_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.Query(ctx, itemsQuery, bill.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := &model.POSBillItem{}
		err := rows.Scan(
			&item.ID,
			&item.BillID,
			&item.ProductID,
			&item.ProductName,
			&item.ProductSKU,
			&item.Quantity,
			&item.UnitPrice,
			&item.UnitCost,
			&item.TotalPrice,
		)
		if err != nil {
			return nil, err
		}
		bill.Items = append(bill.Items, item)
	}

	return bill, nil
}

// Package repository handle db query.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrBillNotFound       = errors.New("bill not found")
	ErrParkedBillNotFound = errors.New("parked bill not found")
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

func (r *POSRepo) CreateSaleWithTx(ctx context.Context, tx pgx.Tx, bill *model.POSBill, pointsToRedeem int) (*model.POSBill, int, int, error) {
	// 1. Check if customer phone matches a registered user for loyalty points
	var customerUserID *string
	pointsAwarded := 0
	actualPointsRedeemed := 0

	cleanPhone := strings.TrimSpace(bill.CustomerPhone)
	if cleanPhone != "" {
		var uid string
		var currentPoints int
		userQuery := `SELECT id, COALESCE(loyalty_points, 0) FROM users WHERE phone = $1 OR email = $1 LIMIT 1`
		if err := tx.QueryRow(ctx, userQuery, cleanPhone).Scan(&uid, &currentPoints); err == nil && uid != "" {
			customerUserID = &uid
			bill.CustomerUserID = &uid

			// Redeem loyalty points if requested
			if pointsToRedeem > 0 {
				if pointsToRedeem > currentPoints {
					actualPointsRedeemed = currentPoints
				} else {
					actualPointsRedeemed = pointsToRedeem
				}
				if actualPointsRedeemed > 0 {
					_, _ = tx.Exec(ctx, `UPDATE users SET loyalty_points = loyalty_points - $1 WHERE id = $2`, actualPointsRedeemed, uid)
				}
			}

			// 1 point awarded per Rs 20 spent
			pointsAwarded = int(bill.TotalAmount / 20)
			if pointsAwarded > 0 {
				_, _ = tx.Exec(ctx, `UPDATE users SET loyalty_points = loyalty_points + $1 WHERE id = $2`, pointsAwarded, uid)
			}
		}
	}

	// 2. Insert Bill header
	billQuery := `
		INSERT INTO pos_bills (
			shop_id, bill_number, customer_phone, customer_user_id,
			subtotal, discount_amount, total_amount, total_cost, payment_method,
			cash_amount, online_amount, khata_amount, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
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
		bill.CashAmount,
		bill.OnlineAmount,
		bill.KhataAmount,
	).Scan(&bill.ID, &bill.CreatedAt)
	if err != nil {
		r.logger.Error("failed to create pos bill in tx", zap.Error(err))
		return nil, 0, 0, err
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

	// 3. Batch insert Bill items and deduct stock in a single network round-trip
	batch := &pgx.Batch{}
	for _, item := range bill.Items {
		item.BillID = bill.ID
		batch.Queue(itemInsertQuery, item.BillID, item.ProductID, item.ProductName, item.ProductSKU, item.Quantity, item.UnitPrice, item.UnitCost, item.TotalPrice)
		batch.Queue(stockDeductQuery, item.Quantity, item.ProductID)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	for _, item := range bill.Items {
		if err := br.QueryRow().Scan(&item.ID); err != nil {
			r.logger.Error("failed to insert pos bill item in batch", zap.Error(err))
			return nil, 0, 0, err
		}
		if _, err := br.Exec(); err != nil {
			r.logger.Error("failed to deduct inventory on pos sale in batch", zap.Error(err), zap.String("product_id", item.ProductID))
			return nil, 0, 0, err
		}
	}

	bill.NetProfit = bill.TotalAmount - bill.TotalCost
	return bill, pointsAwarded, actualPointsRedeemed, nil
}

func (r *POSRepo) GetDailySummary(ctx context.Context, shopID string, date time.Time) (*model.DailySalesSummary, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT 
			COUNT(id) AS total_bills,
			COALESCE(SUM(total_amount), 0.0) AS total_revenue,
			COALESCE(SUM(total_cost), 0.0) AS total_cost,
			COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN total_amount ELSE 0 END), 0.0) AS cash_total,
			COALESCE(SUM(CASE WHEN payment_method = 'upi' THEN total_amount ELSE 0 END), 0.0) AS upi_total,
			COALESCE(SUM(CASE WHEN payment_method = 'card' THEN total_amount ELSE 0 END), 0.0) AS card_total
		FROM pos_bills
		WHERE shop_id = $1 AND created_at >= $2 AND created_at < $3
	`
	summary := &model.DailySalesSummary{
		Date: date.Format("02-Jan-2006"),
	}

	err := r.db.QueryRow(ctx, query, shopID, startOfDay, endOfDay).Scan(
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
		       b.subtotal, b.discount_amount, b.total_amount, b.total_cost, b.payment_method,
		       COALESCE(b.cash_amount, 0), COALESCE(b.online_amount, 0), COALESCE(b.khata_amount, 0),
		       b.created_at,
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
		&bill.CashAmount,
		&bill.OnlineAmount,
		&bill.KhataAmount,
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

// GetDailyCloseReport computes counter sales, khata debt repayments, cash expenses, and exact expected drawer cash.
func (r *POSRepo) GetDailyCloseReport(ctx context.Context, shopID string, targetDate time.Time) (*dto.DailyCloseReport, error) {
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	dateStr := targetDate.Format("2006-01-02")

	query := `
		WITH pos_summary AS (
			SELECT
				COALESCE(SUM(CASE WHEN cash_amount > 0 THEN cash_amount WHEN payment_method = 'cash' THEN total_amount ELSE 0 END), 0) AS cash_sales,
				COALESCE(SUM(CASE WHEN online_amount > 0 THEN online_amount WHEN payment_method IN ('upi', 'online', 'card') THEN total_amount ELSE 0 END), 0) AS upi_sales,
				COALESCE(SUM(CASE WHEN payment_method = 'card' THEN total_amount ELSE 0 END), 0) AS card_sales,
				COALESCE(SUM(CASE WHEN khata_amount > 0 THEN khata_amount WHEN payment_method IN ('credit', 'khata') THEN total_amount ELSE 0 END), 0) AS credit_given,
				COALESCE(SUM(total_amount), 0) AS total_gross_sales,
				COUNT(*) AS total_bills_count
			FROM pos_bills
			WHERE shop_id = $1 AND created_at >= $2 AND created_at < $3
		),
		khata_summary AS (
			SELECT
				COALESCE(SUM(CASE WHEN payment_mode = 'cash' THEN amount ELSE 0 END), 0) AS khata_cash,
				COALESCE(SUM(CASE WHEN payment_mode = 'upi' THEN amount ELSE 0 END), 0) AS khata_upi
			FROM khata_transactions
			WHERE shop_id = $1 AND type = 'RECEIVE_PAYMENT' AND created_at >= $2 AND created_at < $3
		),
		expense_summary AS (
			SELECT
				COALESCE(SUM(CASE WHEN payment_method = 'cash' THEN amount ELSE 0 END), 0) AS cash_exp,
				COALESCE(SUM(amount), 0) AS total_exp
			FROM shop_expenses
			WHERE shop_id = $1 AND expense_date = $4::DATE
		)
		SELECT
			COALESCE(p.cash_sales, 0),
			COALESCE(p.upi_sales, 0),
			COALESCE(p.card_sales, 0),
			COALESCE(p.credit_given, 0),
			COALESCE(p.total_gross_sales, 0),
			COALESCE(p.total_bills_count, 0),
			COALESCE(k.khata_cash, 0),
			COALESCE(k.khata_upi, 0),
			COALESCE(e.cash_exp, 0),
			COALESCE(e.total_exp, 0)
		FROM (SELECT 1) dummy
		LEFT JOIN pos_summary p ON true
		LEFT JOIN khata_summary k ON true
		LEFT JOIN expense_summary e ON true;
	`
	report := &dto.DailyCloseReport{
		Date: dateStr,
	}

	err := r.db.QueryRow(ctx, query, shopID, startOfDay, endOfDay, dateStr).Scan(
		&report.CashSales,
		&report.UPISales,
		&report.CardSales,
		&report.CreditGiven,
		&report.TotalGrossSales,
		&report.TotalBillsCount,
		&report.KhataCashCollected,
		&report.KhataUPICollected,
		&report.CashExpensesPaid,
		&report.TotalExpensesPaid,
	)
	if err != nil {
		r.logger.Error("failed to compute daily close report", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}

	report.TotalKhataCollected = report.KhataCashCollected + report.KhataUPICollected
	report.ExpectedCashInDrawer = report.CashSales + report.KhataCashCollected - report.CashExpensesPaid

	return report, nil
}

// GetMonthlyGSTReport aggregates monthly POS counter sales and calculates GST breakdowns.
func (r *POSRepo) GetMonthlyGSTReport(ctx context.Context, shopID string, year, month int, taxRate float64) (*dto.MonthlyGSTReport, error) {
	query := `
		SELECT 
			COUNT(*) AS total_bills,
			COALESCE(SUM(total_amount), 0) AS total_gross_sales
		FROM pos_bills
		WHERE shop_id = $1
		  AND EXTRACT(YEAR FROM created_at) = $2
		  AND EXTRACT(MONTH FROM created_at) = $3
	`
	var totalBills int
	var totalGross float64
	err := r.db.QueryRow(ctx, query, shopID, year, month).Scan(&totalBills, &totalGross)
	if err != nil {
		r.logger.Error("failed to query monthly gst report", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}

	if taxRate <= 0 {
		taxRate = 18.0
	}

	// Taxable = Gross / (1 + Rate/100)
	taxable := math.Round((totalGross/(1.0+(taxRate/100.0)))*100) / 100
	totalTax := math.Round((totalGross-taxable)*100) / 100
	cgst := math.Round((totalTax/2.0)*100) / 100
	sgst := math.Round((totalTax-cgst)*100) / 100

	slabs := []dto.TaxSlabSummary{
		{
			TaxRatePct:    taxRate,
			TaxableAmount: taxable,
			CGSTAmount:    cgst,
			SGSTAmount:    sgst,
			TotalTax:      totalTax,
			TotalGross:    totalGross,
		},
	}

	monthName := time.Month(month).String()

	return &dto.MonthlyGSTReport{
		ShopID:          shopID,
		Month:           month,
		Year:            year,
		MonthName:       monthName,
		TotalBills:      totalBills,
		TotalGrossSales: totalGross,
		TotalTaxable:    taxable,
		TotalCGST:       cgst,
		TotalSGST:       sgst,
		TotalTax:        totalTax,
		Slabs:           slabs,
		GeneratedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

// ParkBill saves an active counter cart to pos_parked_bills so the cashier can attend to the next customer.
func (r *POSRepo) ParkBill(ctx context.Context, shopID, label, customerPhone string, cart dto.ParkPOSBillRequest, totalAmount float64) (*model.POSParkedBill, error) {
	cartJSON, err := json.Marshal(cart)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parked cart: %w", err)
	}

	query := `
		INSERT INTO pos_parked_bills (shop_id, label, customer_phone, cart_data, total_amount, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, shop_id, COALESCE(label, ''), COALESCE(customer_phone, ''), total_amount, created_at
	`
	parked := &model.POSParkedBill{}
	err = r.db.QueryRow(ctx, query, shopID, strings.TrimSpace(label), strings.TrimSpace(customerPhone), cartJSON, totalAmount).Scan(
		&parked.ID, &parked.ShopID, &parked.Label, &parked.CustomerPhone, &parked.TotalAmount, &parked.CreatedAt,
	)
	if err != nil {
		r.logger.Error("failed to park pos bill", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	parked.CartData = cart
	return parked, nil
}

// ListParkedBills returns all active held carts on the shop's counter.
func (r *POSRepo) ListParkedBills(ctx context.Context, shopID string) ([]*dto.ParkedBillSummaryItem, error) {
	query := `
		SELECT id, COALESCE(label, ''), COALESCE(customer_phone, ''), total_amount, created_at,
		       COALESCE(jsonb_array_length(cart_data->'items'), 0) AS items_count
		FROM pos_parked_bills
		WHERE shop_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to query parked bills", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*dto.ParkedBillSummaryItem
	for rows.Next() {
		var item dto.ParkedBillSummaryItem
		var createdAt time.Time
		if err := rows.Scan(&item.ID, &item.Label, &item.CustomerPhone, &item.TotalAmount, &createdAt, &item.TotalItems); err != nil {
			return nil, err
		}
		item.ParkedAt = createdAt.Format("02-Jan-2006 15:04:05")
		list = append(list, &item)
	}
	if list == nil {
		list = []*dto.ParkedBillSummaryItem{}
	}
	return list, nil
}

// GetParkedBillByID retrieves a parked bill so it can be restored into the active register.
func (r *POSRepo) GetParkedBillByID(ctx context.Context, shopID, id string) (*dto.ParkedBillDetailResponse, error) {
	query := `
		SELECT id, shop_id, COALESCE(label, ''), COALESCE(customer_phone, ''), cart_data, total_amount, created_at
		FROM pos_parked_bills
		WHERE shop_id = $1 AND id = $2
		LIMIT 1
	`
	var resp dto.ParkedBillDetailResponse
	var cartBytes []byte
	var createdAt time.Time

	err := r.db.QueryRow(ctx, query, shopID, id).Scan(
		&resp.ID, &resp.ShopID, &resp.Label, &resp.CustomerPhone, &cartBytes, &resp.TotalAmount, &createdAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrParkedBillNotFound
		}
		return nil, err
	}

	var cart dto.ParkPOSBillRequest
	if err := json.Unmarshal(cartBytes, &cart); err == nil {
		resp.Items = cart.Items
		resp.DiscountAmount = cart.DiscountAmount
		resp.PaymentMethod = cart.PaymentMethod
	}
	resp.ParkedAt = createdAt.Format("02-Jan-2006 15:04:05")
	return &resp, nil
}

// DeleteParkedBill removes a held cart once completed or discarded.
func (r *POSRepo) DeleteParkedBill(ctx context.Context, shopID, id string) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM pos_parked_bills WHERE shop_id = $1 AND id = $2`, shopID, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrParkedBillNotFound
	}
	return nil
}

// GetWeeklyScorecardData computes weekly revenue, growth %, tender breakdown, and net khata cash flow.
func (r *POSRepo) GetWeeklyScorecardData(ctx context.Context, shopID string) (*dto.WeeklyScorecardResponse, error) {
	now := time.Now()
	currStart := now.AddDate(0, 0, -7)
	prevStart := now.AddDate(0, 0, -14)

	res := &dto.WeeklyScorecardResponse{
		ShopID:            shopID,
		CurrentWeekRange:  fmt.Sprintf("%s - %s", currStart.Format("02 Jan"), now.Format("02 Jan 2006")),
		PreviousWeekRange: fmt.Sprintf("%s - %s", prevStart.Format("02 Jan"), currStart.Format("02 Jan 2006")),
		GeneratedAt:       now.Format("02-Jan-2006 15:04"),
	}

	var wg sync.WaitGroup
	var currErr, prevErr, khataErr, topErr, stockErr error

	wg.Add(5)

	// 1. Current 7 days sales & tender breakdown
	go func() {
		defer wg.Done()
		currQuery := `
			SELECT 
				COUNT(*),
				COALESCE(SUM(total_amount), 0),
				COALESCE(SUM(CASE WHEN cash_amount > 0 THEN cash_amount WHEN payment_method = 'cash' THEN total_amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN online_amount > 0 THEN online_amount WHEN payment_method IN ('upi', 'online', 'card') THEN total_amount ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN khata_amount > 0 THEN khata_amount WHEN payment_method IN ('credit', 'khata') THEN total_amount ELSE 0 END), 0)
			FROM pos_bills
			WHERE shop_id = $1 AND created_at >= $2
		`
		currErr = r.db.QueryRow(ctx, currQuery, shopID, currStart).Scan(
			&res.TotalBillsCount,
			&res.CurrentWeekRevenue,
			&res.CashCollected,
			&res.OnlineCollected,
			&res.KhataNewCreditIssued,
		)
		if currErr != nil {
			r.logger.Error("failed to get current week sales", zap.Error(currErr), zap.String("shop_id", shopID))
		} else if res.TotalBillsCount > 0 {
			res.AverageOrderValue = math.Round((res.CurrentWeekRevenue/float64(res.TotalBillsCount))*100) / 100
		}
	}()

	// 2. Previous 7 days revenue for growth %
	go func() {
		defer wg.Done()
		prevQuery := `
			SELECT COALESCE(SUM(total_amount), 0)
			FROM pos_bills
			WHERE shop_id = $1 AND created_at >= $2 AND created_at < $3
		`
		prevErr = r.db.QueryRow(ctx, prevQuery, shopID, prevStart, currStart).Scan(&res.PreviousWeekRevenue)
	}()

	// 3. Khata Cash Recovery for current 7 days
	go func() {
		defer wg.Done()
		khataQuery := `
			SELECT COALESCE(SUM(amount), 0)
			FROM khata_transactions
			WHERE shop_id = $1 AND type = 'RECEIVE_PAYMENT' AND created_at >= $2
		`
		khataErr = r.db.QueryRow(ctx, khataQuery, shopID, currStart).Scan(&res.KhataRecoveredCash)
	}()

	// 4. Top 5 selling products by volume in last 7 days
	var topProducts []*dto.WeeklyTopProductItem
	go func() {
		defer wg.Done()
		topQuery := `
			SELECT i.product_id, i.product_name, SUM(i.quantity) AS units_sold, SUM(i.total_price) AS total_sales
			FROM pos_bill_items i
			JOIN pos_bills b ON i.bill_id = b.id
			WHERE b.shop_id = $1 AND b.created_at >= $2
			GROUP BY i.product_id, i.product_name
			ORDER BY units_sold DESC
			LIMIT 5
		`
		rows, err := r.db.Query(ctx, topQuery, shopID, currStart)
		if err != nil {
			topErr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			var it dto.WeeklyTopProductItem
			if err := rows.Scan(&it.ProductID, &it.ProductName, &it.UnitsSold, &it.TotalSales); err == nil {
				topProducts = append(topProducts, &it)
			}
		}
	}()

	// 5. Critical Out-of-Stock count for items that sold in the last 14 days
	go func() {
		defer wg.Done()
		stockQuery := `
			SELECT COUNT(DISTINCT i.product_id)
			FROM pos_bill_items i
			JOIN pos_bills b ON i.bill_id = b.id
			JOIN inventory inv ON i.product_id = inv.product_id
			WHERE b.shop_id = $1 AND b.created_at >= $2 AND (inv.quantity - inv.reserved_quantity) <= 0
		`
		stockErr = r.db.QueryRow(ctx, stockQuery, shopID, prevStart).Scan(&res.OutOfStockSellersCount)
	}()

	wg.Wait()

	if currErr != nil {
		return nil, currErr
	}

	if prevErr == nil && res.PreviousWeekRevenue > 0 {
		diff := res.CurrentWeekRevenue - res.PreviousWeekRevenue
		res.GrowthPercentage = math.Round((diff/res.PreviousWeekRevenue)*1000) / 10
	} else if res.CurrentWeekRevenue > 0 {
		res.GrowthPercentage = 100.0
	}

	res.NetKhataCashFlow = res.KhataRecoveredCash - res.KhataNewCreditIssued
	if res.NetKhataCashFlow >= 0 {
		res.KhataHealthStatus = "Healthy Cash Recovery (Collected more udhar than issued)"
	} else {
		res.KhataHealthStatus = "Credit Overextension (Issued more udhar than collected - follow up needed)"
	}

	if topProducts == nil {
		topProducts = []*dto.WeeklyTopProductItem{}
	}
	res.TopSellingProducts = topProducts

	_ = khataErr
	_ = topErr
	_ = stockErr

	return res, nil
}

// GetCustomerLastBasket fetches the items purchased on the customer's most recent POS bill with live prices.
func (r *POSRepo) GetCustomerLastBasket(ctx context.Context, shopID, customerPhone string) (*dto.CustomerRecentBasketResponse, error) {
	cleanPhone := strings.TrimSpace(customerPhone)
	billQuery := `
		SELECT id, bill_number, created_at
		FROM pos_bills
		WHERE shop_id = $1 AND customer_phone = $2
		ORDER BY created_at DESC
		LIMIT 1
	`
	var billID, billNumber string
	var createdAt time.Time
	err := r.db.QueryRow(ctx, billQuery, shopID, cleanPhone).Scan(&billID, &billNumber, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("no previous purchases found for this customer")
		}
		return nil, err
	}

	itemsQuery := `
		SELECT i.product_id, i.product_name, COALESCE(p.sku, ''), i.quantity, i.unit_price, p.price,
		       COALESCE(inv.quantity - inv.reserved_quantity, 0) AS available_stock
		FROM pos_bill_items i
		JOIN products p ON i.product_id = p.id
		LEFT JOIN inventory inv ON p.id = inv.product_id
		WHERE i.bill_id = $1
		ORDER BY i.id ASC
	`
	rows, err := r.db.Query(ctx, itemsQuery, billID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var basketItems []*dto.CustomerRecentBasketItem
	var readyCartItems []dto.POSSaleItemRequest
	var estimatedTotal float64

	for rows.Next() {
		var it dto.CustomerRecentBasketItem
		if err := rows.Scan(&it.ProductID, &it.ProductName, &it.SKU, &it.Quantity, &it.LastUnitPrice, &it.CurrentPrice, &it.AvailableStock); err != nil {
			return nil, err
		}
		it.InStock = it.AvailableStock >= it.Quantity
		lineTotal := it.CurrentPrice * float64(it.Quantity)
		estimatedTotal += lineTotal

		basketItems = append(basketItems, &it)
		readyCartItems = append(readyCartItems, dto.POSSaleItemRequest{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	if basketItems == nil {
		basketItems = []*dto.CustomerRecentBasketItem{}
	}

	return &dto.CustomerRecentBasketResponse{
		CustomerPhone:        cleanPhone,
		LastBillNumber:       billNumber,
		LastBillDate:         createdAt.Format("02-Jan-2006 15:04"),
		TotalItemsCount:      len(basketItems),
		EstimatedTotalAmount: math.Round(estimatedTotal*100) / 100,
		Items:                basketItems,
		ReadyCart: &dto.CreatePOSSaleRequest{
			CustomerPhone: cleanPhone,
			PaymentMethod: "cash",
			Items:         readyCartItems,
		},
	}, nil
}

// ProcessPOSReturnWithTx handles in-store product returns: restocks inventory, calculates refund, and records return log.
func (r *POSRepo) ProcessPOSReturnWithTx(ctx context.Context, tx pgx.Tx, shopID string, req dto.ProcessPOSReturnRequest) (*model.POSBill, float64, int, error) {
	bill, err := r.GetBillByNumber(ctx, req.BillNumber)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("bill '%s' not found", req.BillNumber)
	}
	if bill.ShopID != shopID {
		return nil, 0, 0, errors.New("bill does not belong to your shop")
	}

	billItemMap := make(map[string]*model.POSBillItem)
	for _, it := range bill.Items {
		billItemMap[it.ProductID] = it
	}

	var totalRefund float64
	restockedCount := 0

	batch := &pgx.Batch{}
	for _, retItem := range req.Items {
		billedItem, exists := billItemMap[retItem.ProductID]
		if !exists {
			return nil, 0, 0, fmt.Errorf("product %s was not part of bill %s", retItem.ProductID, req.BillNumber)
		}
		if retItem.Quantity > billedItem.Quantity {
			return nil, 0, 0, fmt.Errorf("cannot return quantity %d (only %d originally purchased) for '%s'", retItem.Quantity, billedItem.Quantity, billedItem.ProductName)
		}

		itemRefund := billedItem.UnitPrice * float64(retItem.Quantity)
		totalRefund += itemRefund
		restockedCount += retItem.Quantity

		cleanReason := strings.TrimSpace(req.Reason)
		if cleanReason == "" {
			cleanReason = fmt.Sprintf("Return on Bill %s", req.BillNumber)
		}

		restockQuery := `
			UPDATE inventory
			SET quantity = quantity + $1, updated_at = NOW()
			WHERE product_id = $2
		`
		returnInsertQuery := `
			INSERT INTO product_returns (shop_id, product_id, quantity, refund_amount, reason, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`
		batch.Queue(restockQuery, retItem.Quantity, retItem.ProductID)
		batch.Queue(returnInsertQuery, shopID, retItem.ProductID, retItem.Quantity, itemRefund, cleanReason)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	for range req.Items {
		if _, err := br.Exec(); err != nil {
			r.logger.Error("failed to restock inventory in batch return", zap.Error(err))
			return nil, 0, 0, err
		}
		if _, err := br.Exec(); err != nil {
			r.logger.Error("failed to record product return in batch", zap.Error(err))
			return nil, 0, 0, err
		}
	}

	totalRefund = math.Round(totalRefund*100) / 100
	return bill, totalRefund, restockedCount, nil
}




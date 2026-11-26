// Package repository handle db query.
package repository

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"shopMe/internal/handler/dto"
	"strings"
	"sync"
	"time"
)

// GetShopIntelligence computes industrial-grade retail analytics concurrently in PostgreSQL:
// 1. Period P&L, ABV, MoM/Period Growth & Tender Reconciliation
// 2. Closing Stock Valuation, GMROI, Inventory Turnover & DSI
// 3. Pareto ABC-XYZ Classification (Cumulative Revenue + Coefficient of Variation) & Dead Capital
// 4. Market Basket Association Rules (Support, Confidence, Lift > 1.0)
// 5. IST Peak Hour & Weekday Traffic Heatmap
func (r *ProductRepo) GetShopIntelligence(ctx context.Context, shopID string, fromDate, toDate time.Time) (*dto.ShopIntelligenceResponse, error) {
	periodDays := int(toDate.Sub(fromDate).Hours()/24) + 1
	if periodDays < 1 {
		periodDays = 1
	}
	prevTo := fromDate.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(periodDays - 1))

	res := &dto.ShopIntelligenceResponse{
		FromDate:         fromDate.Format("2006-01-02"),
		ToDate:           toDate.Format("2006-01-02"),
		PeriodDays:       periodDays,
		ABCSummary:       map[string]int{"A": 0, "B": 0, "C": 0},
		DeadCapitalItems: []*dto.ABCXYZItem{},
		TopABCItems:      []*dto.ABCXYZItem{},
		BasketPairs:      []*dto.BasketPairItem{},
		PeakHours:        []*dto.PeakHourBucket{},
		PeakDays:         []*dto.PeakDayBucket{},
	}

	var (
		wg         sync.WaitGroup
		pnlErr     error
		stockErr   error
		abcErr     error
		basketErr  error
		heatmapErr error
	)

	// 1. Period P&L, Tender Reconciliation & Previous Period Growth
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := `
			WITH curr_bills AS (
				SELECT
					COUNT(*)::int AS total_bills,
					COALESCE(SUM(total_amount), 0.0) AS revenue,
					COALESCE(SUM(total_cost), 0.0) AS cost,
					COALESCE(SUM(CASE WHEN LOWER(payment_method) = 'cash' THEN total_amount WHEN LOWER(payment_method) = 'split' THEN COALESCE(cash_amount, 0) ELSE 0 END), 0.0) AS cash_sales,
					COALESCE(SUM(CASE WHEN LOWER(payment_method) IN ('upi', 'online', 'card') THEN total_amount WHEN LOWER(payment_method) = 'split' THEN COALESCE(online_amount, 0) ELSE 0 END), 0.0) AS upi_sales,
					COALESCE(SUM(CASE WHEN LOWER(payment_method) IN ('khata', 'credit', 'udhar') THEN total_amount WHEN LOWER(payment_method) = 'split' THEN COALESCE(khata_amount, 0) ELSE 0 END), 0.0) AS khata_sales
				FROM pos_bills
				WHERE shop_id = $1
				  AND COALESCE(status, '') <> 'cancelled'
				  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $2::date AND $3::date
			),
			curr_exp AS (
				SELECT
					COALESCE(SUM(amount), 0.0) AS total_expenses,
					COALESCE(SUM(CASE WHEN LOWER(COALESCE(payment_method, 'cash')) = 'cash' THEN amount ELSE 0 END), 0.0) AS cash_expenses
				FROM shop_expenses
				WHERE shop_id = $1
				  AND expense_date::date BETWEEN $2::date AND $3::date
			),
			prev_bills AS (
				SELECT COALESCE(SUM(total_amount), 0.0) AS prev_revenue
				FROM pos_bills
				WHERE shop_id = $1
				  AND COALESCE(status, '') <> 'cancelled'
				  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $4::date AND $5::date
			)
			SELECT
				cb.total_bills, cb.revenue, cb.cost, cb.cash_sales, cb.upi_sales, cb.khata_sales,
				ce.total_expenses, ce.cash_expenses, pb.prev_revenue
			FROM curr_bills cb, curr_exp ce, prev_bills pb
		`
		var (
			totalBills   int
			rev, cost    float64
			cashS, upiS  float64
			khataS       float64
			totExp, cExp float64
			prevRev      float64
		)
		err := r.db.QueryRow(ctx, query, shopID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"), prevFrom.Format("2006-01-02"), prevTo.Format("2006-01-02")).
			Scan(&totalBills, &rev, &cost, &cashS, &upiS, &khataS, &totExp, &cExp, &prevRev)
		if err != nil {
			pnlErr = err
			return
		}
		res.TotalBills = totalBills
		res.TotalRevenue = round2(rev)
		res.TotalCost = round2(cost)
		res.GrossProfit = round2(rev - cost)
		res.TotalExpenses = round2(totExp)
		res.NetPocketProfit = round2(res.GrossProfit - totExp)
		if rev > 0 {
			res.AverageMarginPct = round2((res.NetPocketProfit / rev) * 100.0)
		}
		if totalBills > 0 {
			res.AverageBillValue = round2(rev / float64(totalBills))
		}
		res.PrevPeriodRevenue = round2(prevRev)
		if prevRev > 0 {
			res.RevenueGrowthPct = round2(((rev - prevRev) / prevRev) * 100.0)
		}
		res.Tender = dto.TenderReconciliation{
			CashSales:           round2(cashS),
			UPISales:            round2(upiS),
			KhataSales:          round2(khataS),
			CashExpenses:        round2(cExp),
			ExpectedCashInGalla: round2(cashS - cExp),
		}
	}()

	// 2. Closing Stock Valuation
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := `
			SELECT
				COUNT(*)::int AS total_skus,
				COUNT(*) FILTER (WHERE COALESCE(stock, 0) <= 0)::int AS oos_skus,
				COUNT(*) FILTER (WHERE COALESCE(stock, 0) > 0 AND COALESCE(stock, 0) <= COALESCE(NULLIF(low_stock_threshold, 0), 5))::int AS low_skus,
				COUNT(*) FILTER (WHERE COALESCE(stock, 0) > 0 AND COALESCE(cost_price, 0) <= 0)::int AS missing_cost,
				COALESCE(SUM(GREATEST(COALESCE(stock, 0), 0) * CASE WHEN COALESCE(cost_price, 0) > 0 THEN cost_price ELSE price * 0.80 END), 0.0) AS stock_cost_val,
				COALESCE(SUM(GREATEST(COALESCE(stock, 0), 0) * COALESCE(price, 0)), 0.0) AS stock_mrp_val
			FROM products
			WHERE shop_id = $1 AND COALESCE(is_active, true) = true AND deleted_at IS NULL
		`
		var sv dto.StockValuationMetrics
		err := r.db.QueryRow(ctx, query, shopID).Scan(
			&sv.TotalSKUs, &sv.OutOfStockSKUs, &sv.LowStockSKUs, &sv.MissingCostSKUs,
			&sv.ClosingStockCostValue, &sv.ClosingStockMRPValue,
		)
		if err != nil {
			stockErr = err
			return
		}
		sv.ClosingStockCostValue = round2(sv.ClosingStockCostValue)
		sv.ClosingStockMRPValue = round2(sv.ClosingStockMRPValue)
		res.StockValuation = sv
	}()

	// 3. ABC-XYZ 9-Cell Classification + Dead Capital (45d zero sales)
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := `
			WITH prod_sales AS (
				SELECT
					i.product_id,
					SUM(i.quantity)::int AS sold_qty,
					SUM(COALESCE(i.total_price, i.unit_price * i.quantity)) AS revenue,
					SUM(COALESCE(i.total_price, i.unit_price * i.quantity) - (COALESCE(NULLIF(i.unit_cost, 0), p.cost_price, i.unit_price * 0.80) * i.quantity)) AS profit
				FROM pos_bill_items i
				JOIN pos_bills b ON b.id = i.bill_id
				LEFT JOIN products p ON p.id::text = i.product_id::text
				WHERE b.shop_id = $1
				  AND COALESCE(b.status, '') <> 'cancelled'
				  AND (b.created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $2::date AND $3::date
				GROUP BY i.product_id
			),
			daily_qty AS (
				SELECT
					i.product_id,
					(b.created_at AT TIME ZONE 'Asia/Kolkata')::date AS sale_day,
					SUM(i.quantity)::float8 AS day_qty
				FROM pos_bill_items i
				JOIN pos_bills b ON b.id = i.bill_id
				WHERE b.shop_id = $1
				  AND COALESCE(b.status, '') <> 'cancelled'
				  AND (b.created_at AT TIME ZONE 'Asia/Kolkata')::date >= ($3::date - INTERVAL '45 days')
				GROUP BY i.product_id, (b.created_at AT TIME ZONE 'Asia/Kolkata')::date
			),
			xyz_stats AS (
				SELECT
					product_id,
					CASE WHEN AVG(day_qty) > 0 THEN COALESCE(STDDEV_POP(day_qty), 0.0) / AVG(day_qty) ELSE 2.0 END AS cv_score
				FROM daily_qty
				GROUP BY product_id
			),
			abc_ranked AS (
				SELECT
					p.id::text AS product_id,
					p.name,
					COALESCE(p.category, 'General Store') AS category,
					COALESCE(ps.sold_qty, 0) AS sold_qty,
					COALESCE(ps.revenue, 0.0) AS revenue,
					COALESCE(ps.profit, 0.0) AS profit,
					GREATEST(COALESCE(p.stock, 0), 0) AS current_stock,
					GREATEST(COALESCE(p.stock, 0), 0) * CASE WHEN COALESCE(p.cost_price, 0) > 0 THEN p.cost_price ELSE p.price * 0.80 END AS blocked_capital,
					COALESCE(xs.cv_score, 2.0) AS cv_score,
					SUM(COALESCE(ps.revenue, 0.0)) OVER () AS grand_rev,
					SUM(COALESCE(ps.revenue, 0.0)) OVER (ORDER BY COALESCE(ps.revenue, 0.0) DESC, p.name ASC) AS cum_rev
				FROM products p
				LEFT JOIN prod_sales ps ON ps.product_id::text = p.id::text
				LEFT JOIN xyz_stats xs ON xs.product_id::text = p.id::text
				WHERE p.shop_id = $1 AND COALESCE(p.is_active, true) = true AND p.deleted_at IS NULL
			)
			SELECT
				product_id, name, category, sold_qty, revenue, profit, current_stock, blocked_capital, cv_score,
				CASE WHEN grand_rev > 0 THEN (revenue / grand_rev) * 100.0 ELSE 0.0 END AS rev_share_pct,
				CASE
					WHEN revenue <= 0 THEN 'C'
					WHEN grand_rev > 0 AND (cum_rev / grand_rev) <= 0.80 THEN 'A'
					WHEN grand_rev > 0 AND (cum_rev / grand_rev) <= 0.95 THEN 'B'
					ELSE 'C'
				END AS abc_class,
				CASE
					WHEN sold_qty <= 0 THEN 'Z'
					WHEN cv_score < 0.5 THEN 'X'
					WHEN cv_score <= 1.0 THEN 'Y'
					ELSE 'Z'
				END AS xyz_class
			FROM abc_ranked
			ORDER BY revenue DESC, blocked_capital DESC
		`
		rows, err := r.db.Query(ctx, query, shopID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
		if err != nil {
			abcErr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			var it dto.ABCXYZItem
			if scanErr := rows.Scan(
				&it.ProductID, &it.Name, &it.Category, &it.SoldQty, &it.Revenue, &it.GrossProfit,
				&it.CurrentStock, &it.BlockedCapital, &it.CVScore, &it.RevenueSharePct, &it.ABCClass, &it.XYZClass,
			); scanErr != nil {
				continue
			}
			it.Revenue = round2(it.Revenue)
			it.GrossProfit = round2(it.GrossProfit)
			it.BlockedCapital = round2(it.BlockedCapital)
			it.CVScore = round2(it.CVScore)
			it.RevenueSharePct = round2(it.RevenueSharePct)
			if it.Revenue > 0 {
				it.MarginPct = round2((it.GrossProfit / it.Revenue) * 100.0)
			}
			res.ABCSummary[it.ABCClass] = res.ABCSummary[it.ABCClass] + 1
			if it.SoldQty > 0 && len(res.TopABCItems) < 15 {
				res.TopABCItems = append(res.TopABCItems, &it)
			}
			if it.SoldQty == 0 && it.CurrentStock > 0 {
				res.DeadCapitalTotal = round2(res.DeadCapitalTotal + it.BlockedCapital)
				if len(res.DeadCapitalItems) < 15 {
					res.DeadCapitalItems = append(res.DeadCapitalItems, &it)
				}
			}
		}
	}()

	// 4. Market Basket Analysis (Support, Confidence, Lift > 1.05)
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := `
			WITH valid_bills AS (
				SELECT id FROM pos_bills
				WHERE shop_id = $1 AND COALESCE(status, '') <> 'cancelled'
				  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date >= ($2::date - INTERVAL '60 days')
			),
			bill_count AS (
				SELECT GREATEST(COUNT(*), 1)::float8 AS total_tx FROM valid_bills
			),
			prod_tx AS (
				SELECT i.product_id, MAX(i.product_name) AS name, COUNT(DISTINCT i.bill_id)::float8 AS tx_cnt
				FROM pos_bill_items i
				JOIN valid_bills vb ON vb.id = i.bill_id
				WHERE COALESCE(i.product_id::text, '') <> ''
				GROUP BY i.product_id
			),
			pair_tx AS (
				SELECT
					a.product_id AS pid_a,
					b.product_id AS pid_b,
					COUNT(DISTINCT a.bill_id)::int AS pair_cnt
				FROM pos_bill_items a
				JOIN pos_bill_items b ON a.bill_id = b.bill_id AND a.product_id::text < b.product_id::text
				JOIN valid_bills vb ON vb.id = a.bill_id
				GROUP BY a.product_id, b.product_id
				HAVING COUNT(DISTINCT a.bill_id) >= 2
			)
			SELECT
				p.pid_a::text, pa.name, p.pid_b::text, pb.name, p.pair_cnt,
				(p.pair_cnt::float8 / bc.total_tx) * 100.0 AS support_pct,
				(p.pair_cnt::float8 / GREATEST(pa.tx_cnt, 1.0)) * 100.0 AS confidence_pct,
				(p.pair_cnt::float8 * bc.total_tx) / GREATEST(pa.tx_cnt * pb.tx_cnt, 1.0) AS lift
			FROM pair_tx p
			JOIN prod_tx pa ON pa.product_id = p.pid_a
			JOIN prod_tx pb ON pb.product_id = p.pid_b
			CROSS JOIN bill_count bc
			WHERE (p.pair_cnt::float8 * bc.total_tx) / GREATEST(pa.tx_cnt * pb.tx_cnt, 1.0) > 1.05
			ORDER BY lift DESC, pair_cnt DESC
			LIMIT 8
		`
		rows, err := r.db.Query(ctx, query, shopID, toDate.Format("2006-01-02"))
		if err != nil {
			basketErr = err
			return
		}
		defer rows.Close()
		for rows.Next() {
			var b dto.BasketPairItem
			if scanErr := rows.Scan(&b.ProductAID, &b.ProductAName, &b.ProductBID, &b.ProductBName, &b.PairCount, &b.SupportPct, &b.ConfidencePct, &b.Lift); scanErr == nil {
				b.SupportPct = round2(b.SupportPct)
				b.ConfidencePct = round2(b.ConfidencePct)
				b.Lift = round2(b.Lift)
				res.BasketPairs = append(res.BasketPairs, &b)
			}
		}
	}()

	// 5. Peak Hour & Weekday Heatmap (IST)
	wg.Add(1)
	go func() {
		defer wg.Done()
		hourQuery := `
			SELECT
				EXTRACT(HOUR FROM (created_at AT TIME ZONE 'Asia/Kolkata'))::int AS hr,
				COUNT(*)::int AS cnt,
				COALESCE(SUM(total_amount), 0.0) AS rev
			FROM pos_bills
			WHERE shop_id = $1 AND COALESCE(status, '') <> 'cancelled'
			  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $2::date AND $3::date
			GROUP BY hr
			ORDER BY cnt DESC, hr ASC
		`
		hRows, err := r.db.Query(ctx, hourQuery, shopID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
		if err != nil {
			heatmapErr = err
			return
		}
		defer hRows.Close()
		for hRows.Next() {
			var h dto.PeakHourBucket
			if scanErr := hRows.Scan(&h.Hour, &h.BillCount, &h.Revenue); scanErr == nil {
				h.Revenue = round2(h.Revenue)
				res.PeakHours = append(res.PeakHours, &h)
			}
		}

		dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
		dayQuery := `
			SELECT
				EXTRACT(DOW FROM (created_at AT TIME ZONE 'Asia/Kolkata'))::int AS dow,
				COUNT(*)::int AS cnt,
				COALESCE(SUM(total_amount), 0.0) AS rev
			FROM pos_bills
			WHERE shop_id = $1 AND COALESCE(status, '') <> 'cancelled'
			  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $2::date AND $3::date
			GROUP BY dow
			ORDER BY dow ASC
		`
		dRows, err := r.db.Query(ctx, dayQuery, shopID, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02"))
		if err != nil {
			return
		}
		defer dRows.Close()
		for dRows.Next() {
			var d dto.PeakDayBucket
			if scanErr := dRows.Scan(&d.Dow, &d.BillCount, &d.Revenue); scanErr == nil {
				if d.Dow >= 0 && d.Dow < len(dayNames) {
					d.DayName = dayNames[d.Dow]
				}
				d.Revenue = round2(d.Revenue)
				res.PeakDays = append(res.PeakDays, &d)
			}
		}
	}()

	wg.Wait()

	if pnlErr != nil {
		return nil, pnlErr
	}
	_ = stockErr
	_ = abcErr
	_ = basketErr
	_ = heatmapErr

	// Compute GMROI, Inventory Turnover & DSI after both P&L and Stock Valuation finish
	if res.StockValuation.ClosingStockCostValue > 0 {
		res.StockValuation.GMROI = round2(res.GrossProfit / res.StockValuation.ClosingStockCostValue)
		res.StockValuation.InventoryTurnover = round2(res.TotalCost / res.StockValuation.ClosingStockCostValue)
		if res.StockValuation.InventoryTurnover > 0 {
			res.StockValuation.DaysSalesOfInventory = round2(float64(periodDays) / res.StockValuation.InventoryTurnover)
		}
	}

	return res, nil
}

// ExportCACSV generates UTF-8 BOM CSV bytes for Tally / Excel / Chartered Accountant audits.
func (r *ProductRepo) ExportCACSV(ctx context.Context, shopID, exportType string, fromDate, toDate time.Time) ([]byte, string, error) {
	var buf bytes.Buffer
	// Write UTF-8 BOM so Microsoft Excel opens Hindi & ₹ cleanly
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(&buf)

	fromStr := fromDate.Format("2006-01-02")
	toStr := toDate.Format("2006-01-02")
	filename := fmt.Sprintf("shopsilo_%s_%s_to_%s.csv", exportType, fromStr, toStr)

	switch strings.ToLower(strings.TrimSpace(exportType)) {
	case "expenses":
		_ = w.Write([]string{"Date", "Category", "Amount (INR)", "Payment Mode", "Notes"})
		rows, err := r.db.Query(ctx, `
			SELECT expense_date::date::text, category, amount, COALESCE(payment_method, 'cash'), COALESCE(notes, '')
			FROM shop_expenses
			WHERE shop_id = $1 AND expense_date::date BETWEEN $2::date AND $3::date
			ORDER BY expense_date DESC
		`, shopID, fromStr, toStr)
		if err != nil {
			return nil, "", err
		}
		defer rows.Close()
		for rows.Next() {
			var dt, cat, mode, notes string
			var amt float64
			if rows.Scan(&dt, &cat, &amt, &mode, &notes) == nil {
				_ = w.Write([]string{dt, cat, fmt.Sprintf("%.2f", amt), mode, notes})
			}
		}

	case "stock":
		filename = fmt.Sprintf("shopsilo_closing_stock_%s.csv", toStr)
		_ = w.Write([]string{"SKU", "Product Name", "Category", "Stock Qty", "Unit", "Cost Price", "Selling Price", "Stock Value (Cost)", "Stock Value (MRP)"})
		rows, err := r.db.Query(ctx, `
			SELECT COALESCE(sku, ''), name, COALESCE(category, 'General Store'), COALESCE(stock, 0), COALESCE(unit, 'piece'),
			       COALESCE(cost_price, 0), COALESCE(price, 0)
			FROM products
			WHERE shop_id = $1 AND COALESCE(is_active, true) = true AND deleted_at IS NULL
			ORDER BY name ASC
		`, shopID)
		if err != nil {
			return nil, "", err
		}
		defer rows.Close()
		for rows.Next() {
			var sku, name, cat, unit string
			var qty int
			var cost, price float64
			if rows.Scan(&sku, &name, &cat, &qty, &unit, &cost, &price) == nil {
				effCost := cost
				if effCost <= 0 {
					effCost = price * 0.80
				}
				_ = w.Write([]string{
					sku, name, cat, fmt.Sprintf("%d", qty), unit,
					fmt.Sprintf("%.2f", cost), fmt.Sprintf("%.2f", price),
					fmt.Sprintf("%.2f", float64(qty)*effCost), fmt.Sprintf("%.2f", float64(qty)*price),
				})
			}
		}

	case "khata":
		filename = fmt.Sprintf("shopsilo_khata_receivables_%s.csv", toStr)
		_ = w.Write([]string{"Customer Name", "Mobile", "Current Udhar Balance (INR)", "Credit Limit (INR)", "Last Updated"})
		rows, err := r.db.Query(ctx, `
			SELECT customer_name, customer_mobile, COALESCE(current_balance, 0), COALESCE(credit_limit, 0), updated_at::date::text
			FROM customer_khata
			WHERE shop_id = $1
			ORDER BY current_balance DESC
		`, shopID)
		if err != nil {
			return nil, "", err
		}
		defer rows.Close()
		for rows.Next() {
			var name, mobile, upd string
			var bal, lim float64
			if rows.Scan(&name, &mobile, &bal, &lim, &upd) == nil {
				_ = w.Write([]string{name, mobile, fmt.Sprintf("%.2f", bal), fmt.Sprintf("%.2f", lim), upd})
			}
		}

	default: // "sales"
		_ = w.Write([]string{"Date", "Bill Number", "Customer Phone", "Payment Method", "Subtotal", "Discount", "Total Amount", "Status"})
		rows, err := r.db.Query(ctx, `
			SELECT (created_at AT TIME ZONE 'Asia/Kolkata')::date::text, bill_number, COALESCE(customer_phone, ''),
			       payment_method, COALESCE(subtotal, total_amount), COALESCE(discount_amount, 0), total_amount, COALESCE(status, 'completed')
			FROM pos_bills
			WHERE shop_id = $1 AND (created_at AT TIME ZONE 'Asia/Kolkata')::date BETWEEN $2::date AND $3::date
			ORDER BY created_at DESC
		`, shopID, fromStr, toStr)
		if err != nil {
			return nil, "", err
		}
		defer rows.Close()
		for rows.Next() {
			var dt, billNo, phone, method, status string
			var sub, disc, tot float64
			if rows.Scan(&dt, &billNo, &phone, &method, &sub, &disc, &tot, &status) == nil {
				_ = w.Write([]string{dt, billNo, phone, method, fmt.Sprintf("%.2f", sub), fmt.Sprintf("%.2f", disc), fmt.Sprintf("%.2f", tot), status})
			}
		}
	}

	w.Flush()
	return buf.Bytes(), filename, w.Error()
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0.0
	}
	return math.Round(v*100) / 100
}

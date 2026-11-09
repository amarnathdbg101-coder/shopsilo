package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CriticalItem represents a product with dangerously low or zero stock.
type CriticalItem struct {
	Name      string `json:"name"`
	Stock     int    `json:"stock"`
	Threshold int    `json:"threshold"`
}

// OverdueDebtor represents a customer with pending khata balance.
type OverdueDebtor struct {
	Name    string  `json:"name"`
	Mobile  string  `json:"mobile"`
	Balance float64 `json:"balance"`
}

// StoreTelemetrySnapshot captures the live real-time operational state of a store.
type StoreTelemetrySnapshot struct {
	ShopID                 string          `json:"shop_id"`
	TodaySalesAmount       float64         `json:"today_sales_amount"`
	TodaySalesCount        int             `json:"today_sales_count"`
	AverageBasketValue     float64         `json:"average_basket_value"`
	CashSales              float64         `json:"cash_sales"`
	UPISales               float64         `json:"upi_sales"`
	KhataSales             float64         `json:"khata_sales"`
	TodayGrossProfit       float64         `json:"today_gross_profit"`
	TodayExpenses          float64         `json:"today_expenses"`
	LowStockCount          int             `json:"low_stock_count"`
	CriticalItems          []CriticalItem  `json:"critical_items"`
	TotalKhataUdhar        float64         `json:"total_khata_udhar"`
	TotalKhataCustomers    int             `json:"total_khata_customers"`
	TopOverdueDebtors      []OverdueDebtor `json:"top_overdue_debtors"`
	TopMovingProductsToday []string        `json:"top_moving_products_today"`
	CapturedAt             time.Time       `json:"captured_at"`
}

// FormatMarkdown formats the live store telemetry into an unambiguous prompt grounding block.
func (s *StoreTelemetrySnapshot) FormatMarkdown() string {
	var sb strings.Builder
	sb.WriteString("=== LIVE REAL-TIME STORE TELEMETRY (100% FACTUAL GROUND TRUTH) ===\n")
	sb.WriteString(fmt.Sprintf("• Today's Gross Sales: ₹%.2f across %d bills (Avg Basket: ₹%.2f)\n",
		s.TodaySalesAmount, s.TodaySalesCount, s.AverageBasketValue))
	sb.WriteString(fmt.Sprintf("• Payment Split: Cash ₹%.2f | UPI ₹%.2f | Udhar ₹%.2f\n",
		s.CashSales, s.UPISales, s.KhataSales))
	sb.WriteString(fmt.Sprintf("• Est. Gross Profit Today: ₹%.2f | Operating Expenses: ₹%.2f\n",
		s.TodayGrossProfit, s.TodayExpenses))

	if len(s.TopMovingProductsToday) > 0 {
		sb.WriteString(fmt.Sprintf("• Top Moving Items Today: %s\n", strings.Join(s.TopMovingProductsToday, ", ")))
	}

	sb.WriteString(fmt.Sprintf("• Inventory Status: %d items currently low on stock\n", s.LowStockCount))
	if len(s.CriticalItems) > 0 {
		var critStrs []string
		for _, ci := range s.CriticalItems {
			critStrs = append(critStrs, fmt.Sprintf("%s (%d left, min %d)", ci.Name, ci.Stock, ci.Threshold))
		}
		sb.WriteString(fmt.Sprintf("  Critical Alerts: %s\n", strings.Join(critStrs, "; ")))
	}

	sb.WriteString(fmt.Sprintf("• Customer Khata Debt: ₹%.2f pending across %d customer accounts\n",
		s.TotalKhataUdhar, s.TotalKhataCustomers))
	if len(s.TopOverdueDebtors) > 0 {
		var debtStrs []string
		for _, od := range s.TopOverdueDebtors {
			debtStrs = append(debtStrs, fmt.Sprintf("%s (₹%.0f, %s)", od.Name, od.Balance, od.Mobile))
		}
		sb.WriteString(fmt.Sprintf("  Top Debtors: %s\n", strings.Join(debtStrs, "; ")))
	}

	return sb.String()
}

type cachedTelemetry struct {
	snapshot  *StoreTelemetrySnapshot
	expiresAt time.Time
}

// TelemetryProvider aggregates live real-time metrics with 5-second micro-caching.
type TelemetryProvider struct {
	db    *pgxpool.Pool
	mu    sync.RWMutex
	cache map[string]cachedTelemetry
}

// NewTelemetryProvider constructs an initialized TelemetryProvider.
func NewTelemetryProvider(db *pgxpool.Pool) *TelemetryProvider {
	return &TelemetryProvider{
		db:    db,
		cache: make(map[string]cachedTelemetry),
	}
}

// GetSnapshot retrieves the live snapshot from cache or queries PostgreSQL.
func (p *TelemetryProvider) GetSnapshot(ctx context.Context, shopID string) (*StoreTelemetrySnapshot, error) {
	if p.db == nil || strings.TrimSpace(shopID) == "" {
		return &StoreTelemetrySnapshot{
			ShopID:     shopID,
			CapturedAt: time.Now(),
		}, nil
	}

	now := time.Now()

	// Check cache
	p.mu.RLock()
	if entry, ok := p.cache[shopID]; ok && now.Before(entry.expiresAt) {
		p.mu.RUnlock()
		return entry.snapshot, nil
	}
	p.mu.RUnlock()

	// 1. Fetch Today's Sales & Payment Breakdown
	salesQuery := `
		SELECT
			COALESCE(SUM(total_amount), 0),
			COUNT(*),
			COALESCE(SUM(net_profit), 0),
			COALESCE(SUM(cash_amount), 0),
			COALESCE(SUM(online_amount), 0),
			COALESCE(SUM(khata_amount), 0)
		FROM pos_bills
		WHERE shop_id = $1 
		  AND status = 'completed'
		  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date = (NOW() AT TIME ZONE 'Asia/Kolkata')::date
	`
	var (
		totalSales, grossProfit, cashSales, upiSales, khataSales float64
		salesCount                                               int
	)
	_ = p.db.QueryRow(ctx, salesQuery, shopID).Scan(
		&totalSales, &salesCount, &grossProfit, &cashSales, &upiSales, &khataSales,
	)

	avgBasket := 0.0
	if salesCount > 0 {
		avgBasket = totalSales / float64(salesCount)
	}

	// 2. Fetch Today's Shop Expenses
	expenseQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM shop_expenses
		WHERE shop_id = $1 
		  AND (created_at AT TIME ZONE 'Asia/Kolkata')::date = (NOW() AT TIME ZONE 'Asia/Kolkata')::date
	`
	var todayExpenses float64
	_ = p.db.QueryRow(ctx, expenseQuery, shopID).Scan(&todayExpenses)

	// 3. Fetch Low Stock Count & Critical Items
	lowStockQuery := `
		SELECT 
			p.title,
			i.quantity,
			i.low_stock_threshold
		FROM products p
		JOIN inventory i ON p.id = i.product_id
		WHERE p.shop_id = $1 
		  AND p.is_active = true
		  AND i.quantity <= i.low_stock_threshold
		ORDER BY i.quantity ASC
		LIMIT 5
	`
	var criticalItems []CriticalItem
	rows, err := p.db.Query(ctx, lowStockQuery, shopID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ci CriticalItem
			if err := rows.Scan(&ci.Name, &ci.Stock, &ci.Threshold); err == nil {
				criticalItems = append(criticalItems, ci)
			}
		}
	}

	// Total low stock count
	var lowStockCount int
	_ = p.db.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM products p 
		JOIN inventory i ON p.id = i.product_id 
		WHERE p.shop_id = $1 AND p.is_active = true AND i.quantity <= i.low_stock_threshold
	`, shopID).Scan(&lowStockCount)

	// 4. Fetch Khata Debt & Top Debtors
	khataSummaryQuery := `
		SELECT 
			COALESCE(SUM(current_balance), 0),
			COUNT(*)
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance > 0
	`
	var totalKhata float64
	var totalKhataCust int
	_ = p.db.QueryRow(ctx, khataSummaryQuery, shopID).Scan(&totalKhata, &totalKhataCust)

	topDebtorsQuery := `
		SELECT customer_name, customer_mobile, current_balance
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance > 0
		ORDER BY current_balance DESC
		LIMIT 3
	`
	var topDebtors []OverdueDebtor
	dRows, err := p.db.Query(ctx, topDebtorsQuery, shopID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var od OverdueDebtor
			if err := dRows.Scan(&od.Name, &od.Mobile, &od.Balance); err == nil {
				topDebtors = append(topDebtors, od)
			}
		}
	}

	// 5. Fetch Top 3 Moving Products Today
	topMovingQuery := `
		SELECT p.title
		FROM pos_bill_items pbi
		JOIN pos_bills pb ON pbi.bill_id = pb.id
		JOIN products p ON pbi.product_id = p.id
		WHERE pb.shop_id = $1 
		  AND pb.status = 'completed'
		  AND (pb.created_at AT TIME ZONE 'Asia/Kolkata')::date = (NOW() AT TIME ZONE 'Asia/Kolkata')::date
		GROUP BY p.title
		ORDER BY SUM(pbi.quantity) DESC
		LIMIT 3
	`
	var topMoving []string
	mRows, err := p.db.Query(ctx, topMovingQuery, shopID)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var title string
			if err := mRows.Scan(&title); err == nil {
				topMoving = append(topMoving, title)
			}
		}
	}

	snap := &StoreTelemetrySnapshot{
		ShopID:                 shopID,
		TodaySalesAmount:       totalSales,
		TodaySalesCount:        salesCount,
		AverageBasketValue:     avgBasket,
		CashSales:              cashSales,
		UPISales:               upiSales,
		KhataSales:             khataSales,
		TodayGrossProfit:       grossProfit,
		TodayExpenses:          todayExpenses,
		LowStockCount:          lowStockCount,
		CriticalItems:          criticalItems,
		TotalKhataUdhar:        totalKhata,
		TotalKhataCustomers:    totalKhataCust,
		TopOverdueDebtors:      topDebtors,
		TopMovingProductsToday: topMoving,
		CapturedAt:             now,
	}

	// Cache for 5 seconds
	p.mu.Lock()
	p.cache[shopID] = cachedTelemetry{
		snapshot:  snap,
		expiresAt: now.Add(5 * time.Second),
	}
	p.mu.Unlock()

	return snap, nil
}

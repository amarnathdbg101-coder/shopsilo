// Package repository handles database queries for retail intelligence.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrPrescriptionNotFound = errors.New("daily prescription not found")
)

type CopilotRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewCopilotRepo(db *pgxpool.Pool, logger *zap.Logger) *CopilotRepo {
	return &CopilotRepo{
		db:     db,
		logger: logger,
	}
}

// GetDailyPrescriptions retrieves today's unexecuted high-impact prescriptions for the shop.
func (r *CopilotRepo) GetDailyPrescriptions(ctx context.Context, shopID string) ([]*model.DailyPrescription, error) {
	// First check if prescriptions exist for today; if empty, auto-generate them
	query := `
		SELECT id, shop_id, urgency_tier, category, headline, explanation, 
		       financial_impact_inr, executable_action_type, action_payload, 
		       is_executed, executed_at, expires_at, created_at, updated_at
		FROM daily_prescriptions
		WHERE shop_id = $1 AND is_executed = false
		ORDER BY CASE urgency_tier 
			WHEN 'URGENT_CASH' THEN 1 
			WHEN 'STOCKOUT_CRITICAL' THEN 2 
			ELSE 3 
		END, created_at DESC
		LIMIT 3;
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily prescriptions: %w", err)
	}
	defer rows.Close()

	var list []*model.DailyPrescription
	for rows.Next() {
		var p model.DailyPrescription
		var payloadBytes []byte
		err := rows.Scan(
			&p.ID, &p.ShopID, &p.UrgencyTier, &p.Category, &p.Headline, &p.Explanation,
			&p.FinancialImpactINR, &p.ExecutableActionType, &payloadBytes,
			&p.IsExecuted, &p.ExecutedAt, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily prescription: %w", err)
		}
		p.ActionPayload = json.RawMessage(payloadBytes)
		list = append(list, &p)
	}

	if len(list) == 0 {
		// Auto-generate fresh prescriptions from real-time store metrics
		_ = r.GenerateDailyPrescriptions(ctx, shopID)
		// Re-query once
		return r.queryPrescriptions(ctx, shopID)
	}

	return list, nil
}

func (r *CopilotRepo) queryPrescriptions(ctx context.Context, shopID string) ([]*model.DailyPrescription, error) {
	query := `
		SELECT id, shop_id, urgency_tier, category, headline, explanation, 
		       financial_impact_inr, executable_action_type, action_payload, 
		       is_executed, executed_at, expires_at, created_at, updated_at
		FROM daily_prescriptions
		WHERE shop_id = $1 AND is_executed = false
		ORDER BY created_at DESC
		LIMIT 3;
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.DailyPrescription
	for rows.Next() {
		var p model.DailyPrescription
		var payloadBytes []byte
		if err := rows.Scan(
			&p.ID, &p.ShopID, &p.UrgencyTier, &p.Category, &p.Headline, &p.Explanation,
			&p.FinancialImpactINR, &p.ExecutableActionType, &payloadBytes,
			&p.IsExecuted, &p.ExecutedAt, &p.ExpiresAt, &p.CreatedAt, &p.UpdatedAt,
		); err == nil {
			p.ActionPayload = json.RawMessage(payloadBytes)
			list = append(list, &p)
		}
	}
	return list, nil
}

// MarkPrescriptionExecuted marks a prescription card as executed when shopkeeper taps "Apply Now".
func (r *CopilotRepo) MarkPrescriptionExecuted(ctx context.Context, shopID, id string) error {
	now := time.Now()
	res, err := r.db.Exec(ctx, `
		UPDATE daily_prescriptions
		SET is_executed = true, executed_at = $1, updated_at = $1
		WHERE id = $2 AND shop_id = $3
	`, now, id, shopID)
	if err != nil {
		return fmt.Errorf("failed to mark prescription executed: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrPrescriptionNotFound
	}
	return nil
}

// GetDeadCapitalSummary returns aggregate dead stock metrics and carrying cost losses for the merchant.
func (r *CopilotRepo) GetDeadCapitalSummary(ctx context.Context, shopID string) (*model.DeadCapitalSummary, error) {
	query := `
		SELECT 
			COALESCE(COUNT(pi.id), 0) AS total_items,
			COALESCE(SUM(inv.quantity * p.cost_price), 0.0) AS blocked_capital,
			COALESCE(SUM(inv.quantity * p.cost_price * 0.015), 0.0) AS carrying_loss,
			COALESCE(SUM(inv.quantity * GREATEST(p.floor_price, p.cost_price)), 0.0) AS recoverable_cash
		FROM product_intelligence pi
		JOIN products p ON pi.product_id = p.id
		JOIN inventory inv ON p.id = inv.product_id
		WHERE pi.shop_id = $1 AND pi.is_dead_stock = true AND inv.quantity > 0;
	`
	var summary model.DeadCapitalSummary
	err := r.db.QueryRow(ctx, query, shopID).Scan(
		&summary.TotalDeadStockItems,
		&summary.TotalBlockedCapitalINR,
		&summary.MonthlyCarryingCostLoss,
		&summary.RecoverableCashINR,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to aggregate dead capital: %w", err)
	}

	// Fetch top dead stock items for the detailed list
	itemsQuery := `
		SELECT pi.id, pi.product_id, pi.shop_id, p.name, p.price, p.cost_price, 
		       COALESCE(p.floor_price, p.cost_price * 1.15), inv.quantity,
		       COALESCE(c.name, 'General'), pi.attributes, pi.abc_tier, pi.xyz_tier,
		       pi.velocity_daily, pi.days_of_supply, pi.days_since_last_sale,
		       pi.is_dead_stock, pi.carrying_cost_monthly, pi.clearance_stage,
		       pi.recommended_action, pi.recommended_min_price, pi.created_at, pi.updated_at
		FROM product_intelligence pi
		JOIN products p ON pi.product_id = p.id
		JOIN inventory inv ON p.id = inv.product_id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE pi.shop_id = $1 AND pi.is_dead_stock = true AND inv.quantity > 0
		ORDER BY pi.carrying_cost_monthly DESC, pi.days_since_last_sale DESC
		LIMIT 20;
	`
	rows, err := r.db.Query(ctx, itemsQuery, shopID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it model.ProductIntelligence
			var rawAttr []byte
			if scanErr := rows.Scan(
				&it.ID, &it.ProductID, &it.ShopID, &it.ProductName, &it.ProductPrice, &it.CostPrice,
				&it.FloorPrice, &it.CurrentStock, &it.CategoryName, &rawAttr, &it.ABCTier, &it.XYZTier,
				&it.VelocityDaily, &it.DaysOfSupply, &it.DaysSinceLastSale,
				&it.IsDeadStock, &it.CarryingCostMonthly, &it.ClearanceStage,
				&it.RecommendedAction, &it.RecommendedMinPrice, &it.CreatedAt, &it.UpdatedAt,
			); scanErr == nil {
				it.Attributes = json.RawMessage(rawAttr)
				summary.DeadStockItems = append(summary.DeadStockItems, &it)
			}
		}
	}

	return &summary, nil
}

// GetProductIntelligence fetches intelligence metrics for a specific product.
func (r *CopilotRepo) GetProductIntelligence(ctx context.Context, shopID, productID string) (*model.ProductIntelligence, error) {
	query := `
		SELECT pi.id, pi.product_id, pi.shop_id, p.name, p.price, p.cost_price, 
		       COALESCE(p.floor_price, p.cost_price * 1.15), inv.quantity,
		       COALESCE(c.name, 'General'), pi.attributes, pi.abc_tier, pi.xyz_tier,
		       pi.velocity_daily, pi.days_of_supply, pi.days_since_last_sale,
		       pi.is_dead_stock, pi.carrying_cost_monthly, pi.clearance_stage,
		       pi.recommended_action, pi.recommended_min_price, pi.created_at, pi.updated_at
		FROM products p
		JOIN inventory inv ON p.id = inv.product_id
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN product_intelligence pi ON p.id = pi.product_id
		WHERE p.id = $1 AND p.shop_id = $2;
	`
	var it model.ProductIntelligence
	var rawAttr []byte
	var piID, piShopID *string
	var abcTier, xyzTier *string
	var velDaily *float64
	var daysSupply, daysLastSale *int
	var isDead *bool
	var carryCost *float64
	var clearStage *int
	var recAction *string
	var recMinPrice *float64
	var createdAt, updatedAt *time.Time

	err := r.db.QueryRow(ctx, query, productID, shopID).Scan(
		&piID, &it.ProductID, &piShopID, &it.ProductName, &it.ProductPrice, &it.CostPrice,
		&it.FloorPrice, &it.CurrentStock, &it.CategoryName, &rawAttr, &abcTier, &xyzTier,
		&velDaily, &daysSupply, &daysLastSale,
		&isDead, &carryCost, &clearStage,
		&recAction, &recMinPrice, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product intelligence: %w", err)
	}

	if piID != nil {
		it.ID = *piID
		it.ShopID = *piShopID
		if abcTier != nil {
			it.ABCTier = *abcTier
		}
		if xyzTier != nil {
			it.XYZTier = *xyzTier
		}
		if velDaily != nil {
			it.VelocityDaily = *velDaily
		}
		if daysSupply != nil {
			it.DaysOfSupply = *daysSupply
		}
		if daysLastSale != nil {
			it.DaysSinceLastSale = *daysLastSale
		}
		if isDead != nil {
			it.IsDeadStock = *isDead
		}
		if carryCost != nil {
			it.CarryingCostMonthly = *carryCost
		}
		if clearStage != nil {
			it.ClearanceStage = *clearStage
		}
		if recAction != nil {
			it.RecommendedAction = *recAction
		}
		if recMinPrice != nil {
			it.RecommendedMinPrice = *recMinPrice
		}
		if createdAt != nil {
			it.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			it.UpdatedAt = *updatedAt
		}
	} else {
		// Sensible defaults if not yet indexed
		it.ShopID = shopID
		it.ABCTier = "C"
		it.XYZTier = "Z"
		it.FloorPrice = math.Max(it.CostPrice*1.15, it.CostPrice+10.0)
	}

	if len(rawAttr) > 0 {
		it.Attributes = json.RawMessage(rawAttr)
	} else {
		it.Attributes = json.RawMessage("{}")
	}

	return &it, nil
}

// GetCustomerPsychology returns RFM and bargaining records for a phone number.
func (r *CopilotRepo) GetCustomerPsychology(ctx context.Context, shopID, phone string) (*model.CustomerPsychology, error) {
	cleanPhone := strings.TrimSpace(phone)
	if cleanPhone == "" {
		return &model.CustomerPsychology{
			ShopID:               shopID,
			RFMSegment:           "NEW_WALKIN",
			BargainTendencyScore: 0.50,
		}, nil
	}

	query := `
		SELECT id, shop_id, customer_phone, customer_name, rfm_segment,
		       total_lifetime_spend, total_orders, avg_order_value,
		       bargain_tendency_score, max_discount_affinity,
		       preferred_categories, preferred_fabrics_or_brands,
		       last_purchase_date, created_at, updated_at
		FROM customer_psychology
		WHERE shop_id = $1 AND customer_phone = $2;
	`
	var cp model.CustomerPsychology
	err := r.db.QueryRow(ctx, query, shopID, cleanPhone).Scan(
		&cp.ID, &cp.ShopID, &cp.CustomerPhone, &cp.CustomerName, &cp.RFMSegment,
		&cp.TotalLifetimeSpend, &cp.TotalOrders, &cp.AvgOrderValue,
		&cp.BargainTendencyScore, &cp.MaxDiscountAffinity,
		&cp.PreferredCategories, &cp.PreferredFabricsOrBrands,
		&cp.LastPurchaseDate, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Infer from pos_bills if not yet in customer_psychology
			return r.inferCustomerFromBills(ctx, shopID, cleanPhone)
		}
		return nil, err
	}
	return &cp, nil
}

func (r *CopilotRepo) inferCustomerFromBills(ctx context.Context, shopID, phone string) (*model.CustomerPsychology, error) {
	query := `
		SELECT 
			COALESCE(COUNT(id), 0) AS total_orders,
			COALESCE(SUM(total_amount), 0.0) AS total_spend,
			COALESCE(AVG(total_amount), 0.0) AS avg_spend,
			MAX(created_at) AS last_order
		FROM pos_bills
		WHERE shop_id = $1 AND customer_phone = $2;
	`
	var orders int
	var totalSpend, avgSpend float64
	var lastOrder *time.Time

	_ = r.db.QueryRow(ctx, query, shopID, phone).Scan(&orders, &totalSpend, &avgSpend, &lastOrder)

	segment := "NEW_WALKIN"
	bargainScore := 0.50
	if orders >= 5 && totalSpend >= 5000 {
		segment = "VIP_GOLD"
		bargainScore = 0.20
	} else if orders >= 2 {
		segment = "LOYAL_SILVER"
		bargainScore = 0.40
	}

	return &model.CustomerPsychology{
		ShopID:               shopID,
		CustomerPhone:        phone,
		RFMSegment:           segment,
		TotalLifetimeSpend:   totalSpend,
		TotalOrders:          orders,
		AvgOrderValue:        avgSpend,
		BargainTendencyScore: bargainScore,
		LastPurchaseDate:     lastOrder,
	}, nil
}

// RecalculateABCXYZ runs the nightly or on-demand ABC-XYZ demand matrix calculation.
func (r *CopilotRepo) RecalculateABCXYZ(ctx context.Context, shopID string) error {
	// 1. Calculate sales velocity and revenue over past 30 days
	query := `
		WITH sales_30d AS (
			SELECT 
				bi.product_id,
				COALESCE(SUM(bi.quantity), 0) AS total_qty,
				COALESCE(SUM(bi.total_price), 0.0) AS total_rev,
				COUNT(DISTINCT DATE(b.created_at)) AS active_sales_days,
				COALESCE(STDDEV_POP(bi.quantity), 0.0) AS qty_stddev
			FROM pos_bill_items bi
			JOIN pos_bills b ON bi.bill_id = b.id
			WHERE b.shop_id = $1 AND b.created_at >= NOW() - INTERVAL '30 days'
			GROUP BY bi.product_id
		),
		total_shop_rev AS (
			SELECT COALESCE(SUM(total_rev), 1.0) AS grand_total FROM sales_30d
		)
		SELECT 
			p.id,
			inv.quantity,
			p.cost_price,
			p.price,
			COALESCE(p.floor_price, p.cost_price * 1.15),
			COALESCE(s.total_qty, 0) AS qty_sold,
			COALESCE(s.total_rev, 0.0) AS rev_generated,
			COALESCE(s.qty_stddev, 0.0) AS stddev,
			(SELECT grand_total FROM total_shop_rev) AS grand_revenue,
			COALESCE(EXTRACT(DAY FROM NOW() - MAX(b.created_at)), 999)::INT AS days_since_last
		FROM products p
		JOIN inventory inv ON p.id = inv.product_id
		LEFT JOIN sales_30d s ON p.id = s.product_id
		LEFT JOIN pos_bill_items bi ON p.id = bi.product_id
		LEFT JOIN pos_bills b ON bi.bill_id = b.id AND b.shop_id = $1
		WHERE p.shop_id = $1 AND p.is_active = true
		GROUP BY p.id, inv.quantity, p.cost_price, p.price, p.floor_price, s.total_qty, s.total_rev, s.qty_stddev;
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		return fmt.Errorf("failed to query ABC-XYZ calculation data: %w", err)
	}
	defer rows.Close()

	type productMetric struct {
		id            string
		stock         int
		cost          float64
		price         float64
		floor         float64
		qtySold       int
		rev           float64
		stddev        float64
		grandRev      float64
		daysSinceLast int
	}

	var metrics []productMetric
	for rows.Next() {
		var m productMetric
		if err := rows.Scan(
			&m.id, &m.stock, &m.cost, &m.price, &m.floor,
			&m.qtySold, &m.rev, &m.stddev, &m.grandRev, &m.daysSinceLast,
		); err == nil {
			metrics = append(metrics, m)
		}
	}

	// 2. Perform ABC-XYZ scoring
	now := time.Now()
	for _, m := range metrics {
		revShare := 0.0
		if m.grandRev > 0 && !math.IsNaN(m.grandRev) && !math.IsInf(m.grandRev, 0) {
			revShare = m.rev / m.grandRev
		}
		abcTier := "C"
		if revShare >= 0.05 {
			abcTier = "A" // Top revenue
		} else if revShare >= 0.01 {
			abcTier = "B"
		}

		dailyVel := float64(m.qtySold) / 30.0
		if math.IsNaN(dailyVel) || math.IsInf(dailyVel, 0) || dailyVel < 0 {
			dailyVel = 0.0
		}

		cv := 1.5 // Default high variance
		if dailyVel > 0 && !math.IsNaN(m.stddev) && !math.IsInf(m.stddev, 0) {
			cv = m.stddev / dailyVel
		}
		if math.IsNaN(cv) || math.IsInf(cv, 0) {
			cv = 1.5
		}

		xyzTier := "Z"
		if dailyVel > 0.5 && cv < 0.6 {
			xyzTier = "X" // Highly predictable & steady
		} else if dailyVel > 0.1 && cv < 1.1 {
			xyzTier = "Y" // Seasonally fluctuating
		}

		daysSupply := 999
		if dailyVel > 0 {
			rawSupply := float64(m.stock) / dailyVel
			if !math.IsNaN(rawSupply) && !math.IsInf(rawSupply, 0) && rawSupply >= 0 {
				daysSupply = int(math.Min(999, rawSupply))
			}
		}

		isDeadStock := false
		if (m.daysSinceLast > 60 || m.qtySold == 0) && m.stock > 0 {
			isDeadStock = true
		}

		monthlyCarryingCost := float64(m.stock) * m.cost * 0.015 // 1.5% monthly blocked capital interest

		recAction := "Steady stock"
		clearStage := 0
		recMinPrice := math.Max(m.floor, m.cost*1.15)

		if isDeadStock {
			recAction = "Dead Capital: Launch Clearance Combo Deal"
			clearStage = 2
			recMinPrice = math.Max(m.cost*1.05, m.cost+10.0) // Near-cost liquidator
		} else if daysSupply < 7 && abcTier == "A" {
			recAction = "Rocket Item: Reorder from Wholesaler Immediately"
		}

		upsertQuery := `
			INSERT INTO product_intelligence (
				product_id, shop_id, abc_tier, xyz_tier, velocity_daily,
				days_of_supply, days_since_last_sale, is_dead_stock,
				carrying_cost_monthly, clearance_stage, recommended_action,
				recommended_min_price, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)
			ON CONFLICT (product_id) DO UPDATE SET
				abc_tier = EXCLUDED.abc_tier,
				xyz_tier = EXCLUDED.xyz_tier,
				velocity_daily = EXCLUDED.velocity_daily,
				days_of_supply = EXCLUDED.days_of_supply,
				days_since_last_sale = EXCLUDED.days_since_last_sale,
				is_dead_stock = EXCLUDED.is_dead_stock,
				carrying_cost_monthly = EXCLUDED.carrying_cost_monthly,
				clearance_stage = EXCLUDED.clearance_stage,
				recommended_action = EXCLUDED.recommended_action,
				recommended_min_price = EXCLUDED.recommended_min_price,
				updated_at = EXCLUDED.updated_at;
		`
		_, _ = r.db.Exec(ctx, upsertQuery,
			m.id, shopID, abcTier, xyzTier, dailyVel,
			daysSupply, m.daysSinceLast, isDeadStock,
			monthlyCarryingCost, clearStage, recAction,
			recMinPrice, now,
		)
	}

	return nil
}

// GenerateDailyPrescriptions synthesizes exactly 3 high-impact actionable cards for store owner.
func (r *CopilotRepo) GenerateDailyPrescriptions(ctx context.Context, shopID string) error {
	// First purge any unexecuted prescriptions older than 24 hours to prevent stale cards
	_, _ = r.db.Exec(ctx, `
		DELETE FROM daily_prescriptions 
		WHERE shop_id = $1 AND is_executed = false AND created_at < NOW() - INTERVAL '24 hours';
	`, shopID)

	// Card 1: Dead Stock Cash Recovery
	var deadStockCount int
	var blockedCapital, carryingLoss float64
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(inv.quantity * p.cost_price), 0), COALESCE(SUM(pi.carrying_cost_monthly), 0)
		FROM product_intelligence pi
		JOIN products p ON pi.product_id = p.id
		JOIN inventory inv ON p.id = inv.product_id
		WHERE pi.shop_id = $1 AND pi.is_dead_stock = true AND inv.quantity > 0;
	`, shopID).Scan(&deadStockCount, &blockedCapital, &carryingLoss)

	if deadStockCount > 0 {
		headline := fmt.Sprintf("%d Dead Stock Items Blocking ₹%.0f Cash", deadStockCount, blockedCapital)
		explanation := fmt.Sprintf("In items par aapka har mahine ₹%.0f ka interest aur shelf rent nuksan ho raha hai. Launch 1-tap clearance combo to recover liquid cash.", carryingLoss)
		actionPayload, _ := json.Marshal(map[string]any{
			"type":            "DEAD_STOCK_CLEARANCE",
			"suggested_off":   "25%",
			"target_products": deadStockCount,
		})
		_, _ = r.db.Exec(ctx, `
			INSERT INTO daily_prescriptions (
				shop_id, urgency_tier, category, headline, explanation, 
				financial_impact_inr, executable_action_type, action_payload
			) VALUES ($1, 'URGENT_CASH', 'DEAD_STOCK', $2, $3, $4, 'CLEAR_STOCK', $5);
		`, shopID, headline, explanation, blockedCapital, actionPayload)
	}

	// Card 2: Imminent Stockout on Top Selling 'A' Items
	var stockoutCount int
	var topProductName string
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(MAX(p.name), '')
		FROM product_intelligence pi
		JOIN products p ON pi.product_id = p.id
		WHERE pi.shop_id = $1 AND pi.abc_tier = 'A' AND pi.days_of_supply <= 5;
	`, shopID).Scan(&stockoutCount, &topProductName)

	if stockoutCount > 0 {
		headline := fmt.Sprintf("Stockout Alert: %s & %d Top Sellers Running Out", topProductName, stockoutCount)
		explanation := "Ye aapke dukan ke top 80% bikri wale items hain. Agle 3 din me stockout ho jayega. Wholesaler purchase order turant dispatch karein."
		actionPayload, _ := json.Marshal(map[string]any{
			"type":         "WHOLESALE_REORDER",
			"product_name": topProductName,
		})
		_, _ = r.db.Exec(ctx, `
			INSERT INTO daily_prescriptions (
				shop_id, urgency_tier, category, headline, explanation, 
				financial_impact_inr, executable_action_type, action_payload
			) VALUES ($1, 'STOCKOUT_CRITICAL', 'REORDER', $2, $3, 12000.0, 'SEND_PO', $4);
		`, shopID, headline, explanation, actionPayload)
	}

	// Card 3: Overdue Khata Recovery
	var overdueCustomers int
	var overdueAmount float64
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(current_balance), 0.0)
		FROM khata_customers
		WHERE shop_id = $1 AND current_balance > 0 AND last_payment_date < NOW() - INTERVAL '30 days';
	`, shopID).Scan(&overdueCustomers, &overdueAmount)

	if overdueCustomers > 0 {
		headline := fmt.Sprintf("%d Customers with 30+ Days Overdue Khata (₹%.0f)", overdueCustomers, overdueAmount)
		explanation := "Udhari recovery me der hone par bad-debt risk badh jata hai. Automated WhatsApp payment reminder bhejein."
		actionPayload, _ := json.Marshal(map[string]any{
			"type":              "KHATA_REMINDER",
			"customer_count":    overdueCustomers,
			"recoverable_total": overdueAmount,
		})
		_, _ = r.db.Exec(ctx, `
			INSERT INTO daily_prescriptions (
				shop_id, urgency_tier, category, headline, explanation, 
				financial_impact_inr, executable_action_type, action_payload
			) VALUES ($1, 'KHATA_RECOVERY', 'KHATA', $2, $3, $4, 'SEND_REMINDER', $5);
		`, shopID, headline, explanation, overdueAmount, actionPayload)
	}

	return nil
}

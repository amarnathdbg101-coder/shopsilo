package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// StartAnomalyWorker runs the continuous background Anomaly & Fraud Detection Engine.
// It actively scans POS transactions, cash registers, and Khata default risks across shops,
// generating high-priority proactive briefings to safeguard store profitability.
func StartAnomalyWorker(
	ctx context.Context,
	db *pgxpool.Pool,
	brainRepo *repository.AIBrainRepo,
	logger *zap.Logger,
) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("ai anomaly worker recovered from panic", zap.Any("panic", r))
		}
	}()

	logger.Info("ai anomaly worker initialized (Autonomous Fraud & Leakage Radar)")

	// Scan every 20 minutes
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()

	// Initial run shortly after startup (after 10 seconds)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
			runAnomalyPass(ctx, db, brainRepo, logger)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("ai anomaly worker stopped cleanly")
			return

		case <-ticker.C:
			runAnomalyPass(ctx, db, brainRepo, logger)
		}
	}
}

func runAnomalyPass(
	ctx context.Context,
	db *pgxpool.Pool,
	brainRepo *repository.AIBrainRepo,
	logger *zap.Logger,
) {
	if db == nil || brainRepo == nil {
		return
	}

	scanCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			logger.Error("ai anomaly scan caught panic during run", zap.Any("panic", r))
		}
	}()

	// Scan 1: Cash drawer pilferage & unauthorized manual opens
	scanCashDrawerAnomalies(scanCtx, db, brainRepo, logger)

	// Scan 2: Abnormal high-discount bills (>25% discount)
	scanDiscountAnomalies(scanCtx, db, brainRepo, logger)

	// Scan 3: Bill cancellation / void spike
	scanVoidSpikeAnomalies(scanCtx, db, brainRepo, logger)

	// Scan 4: Stale high-value Khata debtors
	scanKhataOverdueAnomalies(scanCtx, db, brainRepo, logger)
}

func hasRecentBriefing(ctx context.Context, db *pgxpool.Pool, shopID, briefingType, titleSubstr string) bool {
	var count int
	query := `
		SELECT COUNT(*)
		FROM ai_proactive_briefings
		WHERE shop_id = $1 AND briefing_type = $2 AND is_dismissed = false
		  AND title LIKE '%' || $3 || '%'
		  AND created_at >= NOW() - INTERVAL '12 hours';
	`
	_ = db.QueryRow(ctx, query, shopID, briefingType, titleSubstr).Scan(&count)
	return count > 0
}

func scanCashDrawerAnomalies(ctx context.Context, db *pgxpool.Pool, brainRepo *repository.AIBrainRepo, logger *zap.Logger) {
	query := `
		SELECT s.shop_id, s.shift_name, s.discrepancy_amount, s.expected_cash, COALESCE(s.actual_cash, 0),
		       COUNT(CASE WHEN e.event_type = 'MANUAL_OPEN' THEN 1 END) as manual_opens
		FROM cash_drawer_sessions s
		LEFT JOIN cash_drawer_events e ON e.session_id = s.id
		WHERE s.opened_at >= NOW() - INTERVAL '12 hours'
		GROUP BY s.id, s.shop_id, s.shift_name, s.discrepancy_amount, s.expected_cash, s.actual_cash
		HAVING s.discrepancy_amount < -50.0 OR COUNT(CASE WHEN e.event_type = 'MANUAL_OPEN' THEN 1 END) >= 3;
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var shopID, shiftName string
		var discrepancy, expected, actual float64
		var manualOpens int

		if err := rows.Scan(&shopID, &shiftName, &discrepancy, &expected, &actual, &manualOpens); err != nil {
			continue
		}

		if hasRecentBriefing(ctx, db, shopID, model.BriefingAnomalyTheft, "Cash Register") {
			continue
		}

		summary := fmt.Sprintf(
			"🚨 **Cash Register Anomaly Alert** (Shift: %s):\n- **Expected Cash**: ₹%.2f\n- **Counted Cash**: ₹%.2f\n- **Shortage**: ₹%.2f\n- **Manual Drawer Opens (No Bill)**: %d times\n\nKripya cashier se petty cash expenses aur counter slips verify karein.",
			shiftName, expected, actual, -discrepancy, manualOpens,
		)

		actionItems, _ := json.Marshal([]map[string]string{
			{"title": "Verify Drawer Slips", "action": "open_cash_audit"},
			{"title": "Ask AI CA Munim", "action": "chat_ca_munim"},
		})

		_ = brainRepo.CreateBriefing(ctx, &model.AIProactiveBriefing{
			ShopID:          shopID,
			BriefingType:    model.BriefingAnomalyTheft,
			Title:           "🚨 Cash Register Anomaly Detected",
			SummaryMarkdown: summary,
			ActionItems:     actionItems,
		})
		logger.Warn("proactive cash anomaly briefing dispatched", zap.String("shop_id", shopID))
	}
}

func scanDiscountAnomalies(ctx context.Context, db *pgxpool.Pool, brainRepo *repository.AIBrainRepo, logger *zap.Logger) {
	query := `
		SELECT shop_id, bill_number, subtotal, discount_amount, total_amount
		FROM pos_bills
		WHERE status = 'completed' AND created_at >= NOW() - INTERVAL '6 hours'
		  AND discount_amount > 50.0 AND (discount_amount / (subtotal + 0.0001)) >= 0.25
		ORDER BY discount_amount DESC
		LIMIT 5;
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var shopID, billNumber string
		var subtotal, discount, total float64

		if err := rows.Scan(&shopID, &billNumber, &subtotal, &discount, &total); err != nil {
			continue
		}

		if hasRecentBriefing(ctx, db, shopID, model.BriefingAnomalyTheft, "Discount Spike") {
			continue
		}

		pct := (discount / subtotal) * 100
		summary := fmt.Sprintf(
			"⚠️ **Unusual Discount Spike Detected**:\nBill **#%s** par **%.1f%% ka discount** (₹%.2f) diya gaya hai (Subtotal: ₹%.2f, Bill Total: ₹%.2f).\n\nCheck karein ki kya ye discount shop policy ke anurup approve hua tha ya cashier mistake hai.",
			billNumber, pct, discount, subtotal, total,
		)

		actionItems, _ := json.Marshal([]map[string]string{
			{"title": fmt.Sprintf("Audit Bill #%s", billNumber), "action": "view_bill"},
		})

		_ = brainRepo.CreateBriefing(ctx, &model.AIProactiveBriefing{
			ShopID:          shopID,
			BriefingType:    model.BriefingAnomalyTheft,
			Title:           fmt.Sprintf("⚠️ High Discount Alert (Bill #%s)", billNumber),
			SummaryMarkdown: summary,
			ActionItems:     actionItems,
		})
		logger.Warn("proactive discount anomaly briefing dispatched", zap.String("shop_id", shopID), zap.String("bill_number", billNumber))
	}
}

func scanVoidSpikeAnomalies(ctx context.Context, db *pgxpool.Pool, brainRepo *repository.AIBrainRepo, logger *zap.Logger) {
	query := `
		SELECT shop_id, COUNT(*)
		FROM pos_bills
		WHERE status = 'cancelled' AND cancelled_at >= NOW() - INTERVAL '6 hours'
		GROUP BY shop_id
		HAVING COUNT(*) >= 2;
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var shopID string
		var cancelCount int

		if err := rows.Scan(&shopID, &cancelCount); err != nil {
			continue
		}

		if hasRecentBriefing(ctx, db, shopID, model.BriefingAnomalyTheft, "Bill Cancellation Spike") {
			continue
		}

		summary := fmt.Sprintf(
			"🚨 **Frequent Bill Cancellations**:\nPichle 6 ghante me counter par **%d bills void/cancel** kiye gaye hain. Retail me ye cashier cash diversion ya fake return ka sanket ho sakta hai. Cancelled bills ke audit trails check karein.",
			cancelCount,
		)

		_ = brainRepo.CreateBriefing(ctx, &model.AIProactiveBriefing{
			ShopID:          shopID,
			BriefingType:    model.BriefingAnomalyTheft,
			Title:           "🚨 Frequent Bill Cancellation Spike",
			SummaryMarkdown: summary,
			ActionItems:     json.RawMessage(`[{"title":"Audit Cancelled Bills","action":"view_cancelled_bills"}]`),
		})
		logger.Warn("proactive bill cancellation spike briefing dispatched", zap.String("shop_id", shopID))
	}
}

func scanKhataOverdueAnomalies(ctx context.Context, db *pgxpool.Pool, brainRepo *repository.AIBrainRepo, logger *zap.Logger) {
	query := `
		SELECT shop_id, customer_name, customer_mobile, current_balance
		FROM customer_khata
		WHERE current_balance >= 2000.0 AND updated_at <= NOW() - INTERVAL '30 days'
		ORDER BY current_balance DESC
		LIMIT 1;
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var shopID, customerName, customerMobile string
		var balance float64

		if err := rows.Scan(&shopID, &customerName, &customerMobile, &balance); err != nil {
			continue
		}

		if hasRecentBriefing(ctx, db, shopID, model.BriefingAnomalyTheft, "Khata Default Risk") {
			continue
		}

		summary := fmt.Sprintf(
			"📋 **High-Risk Khata Stagnation Alert**:\nCustomer **%s** (Mob: %s) ka **₹%.2f** ka udhar 30 se adhik dino se stagnant hai bina kisi payment ke.\n\nBad debt banne se pehle unhein AI WhatsApp gentle payment reminder bhej kar settle karein.",
			customerName, customerMobile, balance,
		)

		actionItems, _ := json.Marshal([]map[string]string{
			{"title": "Send WhatsApp Nudge", "action": "send_nudge", "mobile": customerMobile},
		})

		_ = brainRepo.CreateBriefing(ctx, &model.AIProactiveBriefing{
			ShopID:          shopID,
			BriefingType:    model.BriefingAnomalyTheft,
			Title:           fmt.Sprintf("📋 Khata Default Risk (%s)", customerName),
			SummaryMarkdown: summary,
			ActionItems:     actionItems,
		})
		logger.Info("proactive khata overdue briefing dispatched", zap.String("shop_id", shopID), zap.String("customer", customerName))
	}
}

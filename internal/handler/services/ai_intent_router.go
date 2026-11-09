package services

import (
	"fmt"
	"regexp"
	"strings"

	"shopMe/internal/handler/model"
)

var (
	// Action-forcing verbs that require ReAct Tool Calling / State Modifications (must not bypass LLM)
	reActionVerbs = regexp.MustCompile(`(?i)\b(kardo|kar do|likh do|likhdo|update|badha do|ghata do|add|restock|propose|bhejo|change|set)\b`)

	// Intent Regex Matchers for sub-2ms deterministic instant resolution
	reSalesQuery  = regexp.MustCompile(`(?i)\b(sale|sales|bikri|kamai|aaj ka hisab|aaj kitna|total business|turnover)\b`)
	reStockQuery  = regexp.MustCompile(`(?i)\b(stock|maal|khatam|low stock|out of stock|inventory|reorder|khali)\b`)
	reKhataQuery  = regexp.MustCompile(`(?i)\b(khata|udhar|udhari|baki|debt|overdue|recovery|kiska baki)\b`)
	reProfitQuery = regexp.MustCompile(`(?i)\b(profit|munafa|margin|net profit|gross profit|bachat|income)\b`)
	reCashQuery   = regexp.MustCompile(`(?i)\b(cash|galla|drawer|counter cash|upi|online cash|petty cash)\b`)
)

// AIIntentRouter provides sub-2ms deterministic intent resolution for high-frequency factual queries.
type AIIntentRouter struct{}

// NewAIIntentRouter initializes the intent routing fast-path.
func NewAIIntentRouter() *AIIntentRouter {
	return &AIIntentRouter{}
}

// RouteFastPath attempts to resolve a user query locally using live store telemetry without incurring LLM latency.
// Returns formatted answer and true if the query was resolved deterministically with 100% confidence.
func (r *AIIntentRouter) RouteFastPath(
	userMessage string,
	persona string,
	snap *StoreTelemetrySnapshot,
) (string, bool) {
	if snap == nil {
		return "", false
	}

	clean := strings.TrimSpace(userMessage)
	if clean == "" {
		return "", false
	}

	// 1. If message contains state-mutating action verbs (e.g. "price 48 kardo"), delegate to ReAct Tool Loop!
	if reActionVerbs.MatchString(clean) {
		return "", false
	}

	// 2. Factual Query Intent 1: Today's Sales & Counter Performance
	if reSalesQuery.MatchString(clean) && !reStockQuery.MatchString(clean) && !reKhataQuery.MatchString(clean) {
		return r.formatSalesIntentResponse(persona, snap), true
	}

	// 3. Factual Query Intent 2: Inventory Health & Low Stock Radar
	if reStockQuery.MatchString(clean) {
		return r.formatStockIntentResponse(persona, snap), true
	}

	// 4. Factual Query Intent 3: Customer Khata & Overdue Debt Radar
	if reKhataQuery.MatchString(clean) {
		return r.formatKhataIntentResponse(persona, snap), true
	}

	// 5. Factual Query Intent 4: Net Profit & Operational Margins
	if reProfitQuery.MatchString(clean) {
		return r.formatProfitIntentResponse(persona, snap), true
	}

	// 6. Factual Query Intent 5: Cash vs UPI Counter Drawer Breakdown
	if reCashQuery.MatchString(clean) {
		return r.formatCashDrawerIntentResponse(persona, snap), true
	}

	// Unmatched intent -> Fall through to ReAct Gemini loop for deep multi-turn reasoning
	return "", false
}

func (r *AIIntentRouter) formatSalesIntentResponse(persona string, snap *StoreTelemetrySnapshot) string {
	avg := snap.AverageBasketValue
	switch persona {
	case model.PersonaCA:
		return fmt.Sprintf("📊 **CA Sales Audit**: Aaj ab tak total **₹%.2f** ki billing hui hai across **%d bills** (Average Basket: ₹%.2f). Payment split: Physical Cash ₹%.2f, UPI/QR ₹%.2f, aur Udhar Khata ₹%.2f.",
			snap.TodaySalesAmount, snap.TodaySalesCount, avg, snap.CashSales, snap.UPISales, snap.KhataSales)

	case model.PersonaAnalyst:
		topMoving := "Grocery & Essentials"
		if len(snap.TopMovingProductsToday) > 0 {
			topMoving = strings.Join(snap.TopMovingProductsToday, ", ")
		}
		return fmt.Sprintf("📈 **Sales Velocity Radar**: Aaj total revenue **₹%.2f** (%d bills punch hue). Average basket size ₹%.2f hai. Top moving velocity items: **%s**.",
			snap.TodaySalesAmount, snap.TodaySalesCount, avg, topMoving)

	case model.PersonaStaff:
		return fmt.Sprintf("⚡ **Counter Status**: Aaj **%d bills** complete hue hain with total collection of **₹%.2f** (Cash: ₹%.2f, UPI: ₹%.2f). Counter queue smooth hai.",
			snap.TodaySalesCount, snap.TodaySalesAmount, snap.CashSales, snap.UPISales)

	default: // COO / Partner
		return fmt.Sprintf("Bhai, aaj ab tak **₹%.2f** ki total sales ho chuki hai across **%d bills** (Average ticket size: ₹%.2f). Cash collection ₹%.2f hai aur UPI ₹%.2f. Sales positive trend me hai!",
			snap.TodaySalesAmount, snap.TodaySalesCount, avg, snap.CashSales, snap.UPISales)
	}
}

func (r *AIIntentRouter) formatStockIntentResponse(persona string, snap *StoreTelemetrySnapshot) string {
	if snap.LowStockCount == 0 {
		return "✅ **Inventory Healthy**: Aapki dukan ka sabhi essential stock buffer threshold se upar hai. Zero out-of-stock items detected."
	}

	var critStrs []string
	for _, ci := range snap.CriticalItems {
		critStrs = append(critStrs, fmt.Sprintf("• **%s**: sirf %d units bache hain (Min buffer: %d)", ci.Name, ci.Stock, ci.Threshold))
	}

	return fmt.Sprintf("🚨 **Low Stock Radar**: Total **%d items** par stock critical chal raha hai:\n%s\n\nKahiye to distributor ko restock order bhejne ke liye action proposal create kar dun?",
		snap.LowStockCount, strings.Join(critStrs, "\n"))
}

func (r *AIIntentRouter) formatKhataIntentResponse(persona string, snap *StoreTelemetrySnapshot) string {
	if snap.TotalKhataUdhar <= 0 {
		return "🎉 **Khata All Clear**: Badhai ho! Shop ka customer udhar balance zero hai, sabhi khata accounts fully settled hain."
	}

	var debtStrs []string
	for _, od := range snap.TopOverdueDebtors {
		debtStrs = append(debtStrs, fmt.Sprintf("• **%s**: ₹%.0f (Mob: %s)", od.Name, od.Balance, od.Mobile))
	}

	return fmt.Sprintf("📋 **Customer Khata Radar**: Market me kul **₹%.2f** ka udhar baki hai across **%d customers**.\nTop pending accounts:\n%s\n\nKisi bhi customer ko WhatsApp payment reminder bhejne ke liye batayein.",
		snap.TotalKhataUdhar, snap.TotalKhataCustomers, strings.Join(debtStrs, "\n"))
}

func (r *AIIntentRouter) formatProfitIntentResponse(persona string, snap *StoreTelemetrySnapshot) string {
	netProfit := snap.TodayGrossProfit - snap.TodayExpenses
	marginPct := 0.0
	if snap.TodaySalesAmount > 0 {
		marginPct = (netProfit / snap.TodaySalesAmount) * 100
	}

	return fmt.Sprintf("💰 **P&L Breakdown Today**:\n- **Gross Sales**: ₹%.2f\n- **Gross Profit**: ₹%.2f\n- **Operating Expenses**: ₹%.2f\n- **Estimated Net Profit**: **₹%.2f** (Net Margin: %.1f%%)\n\nExpenses controlled hain aur margin safe zone me hai.",
		snap.TodaySalesAmount, snap.TodayGrossProfit, snap.TodayExpenses, netProfit, marginPct)
}

func (r *AIIntentRouter) formatCashDrawerIntentResponse(persona string, snap *StoreTelemetrySnapshot) string {
	return fmt.Sprintf("💵 **Counter Cash & Payment Breakdown**:\n- **Physical Cash in Register**: ₹%.2f\n- **Online UPI / QR**: ₹%.2f\n- **Khata / Udhar Given**: ₹%.2f\n- **Total Today**: ₹%.2f\n\nPhysical cash ko sham ko drawer se tally karna mat bhooliye.",
		snap.CashSales, snap.UPISales, snap.KhataSales, snap.TodaySalesAmount)
}

// Package dto handles request and response struct.
package dto

type MonthlyProfitResponse struct {
	Month               string  `json:"month"`
	Year                int     `json:"year"`
	TotalCompletedSales int     `json:"total_completed_sales"`
	TotalItemsSold      int     `json:"total_items_sold"`
	TotalRevenue        float64            `json:"total_revenue"`
	TotalCost           float64            `json:"total_cost"`
	GrossProfit         float64            `json:"gross_profit"`
	NetProfit           float64            `json:"net_profit"` // Sales Net Profit (Revenue - Cost)
	TotalExpenses       float64            `json:"total_expenses"`
	NetPocketProfit     float64            `json:"net_pocket_profit"` // Final Pocket Profit (NetProfit - TotalExpenses)
	ExpenseBreakdown    map[string]float64 `json:"expense_breakdown"`
	AverageMarginPct    float64            `json:"average_margin_pct"`
}

type ProductPerformanceItem struct {
	ProductID       string  `json:"product_id"`
	Name            string  `json:"name"`
	SKU             string  `json:"sku"`
	Price           float64 `json:"price"`
	CostPrice       float64 `json:"cost_price"`
	UnitProfit      float64 `json:"unit_profit"`
	ProfitMarginPct float64 `json:"profit_margin_pct"`
	CurrentStock    int     `json:"current_stock"`
	TotalSoldQty    int     `json:"total_sold_qty"`
	TotalProfit     float64 `json:"total_profit"`
	DaysInStock     int     `json:"days_in_stock"`
}

type ProductMatrixResponse struct {
	BestProfitable []*ProductPerformanceItem `json:"best_profitable"`
	WorstProfitable []*ProductPerformanceItem `json:"worst_profitable"`
	OldDeadStock    []*ProductPerformanceItem `json:"old_dead_stock"`
	NewArrivals     []*ProductPerformanceItem `json:"new_arrivals"`
}

// TenderReconciliation breaks down revenue and cash drawer flow by payment mode.
type TenderReconciliation struct {
	CashSales           float64 `json:"cash_sales"`
	UPISales            float64 `json:"upi_sales"`
	KhataSales          float64 `json:"khata_sales"`
	CashExpenses        float64 `json:"cash_expenses"`
	ExpectedCashInGalla float64 `json:"expected_cash_in_galla"`
}

// StockValuationMetrics holds closing stock valuation and retail capital efficiency KPIs.
type StockValuationMetrics struct {
	ClosingStockCostValue float64 `json:"closing_stock_cost_value"`
	ClosingStockMRPValue  float64 `json:"closing_stock_mrp_value"`
	TotalSKUs             int     `json:"total_skus"`
	OutOfStockSKUs        int     `json:"out_of_stock_skus"`
	LowStockSKUs          int     `json:"low_stock_skus"`
	MissingCostSKUs       int     `json:"missing_cost_skus"`
	GMROI                 float64 `json:"gmroi"`
	InventoryTurnover     float64 `json:"inventory_turnover"`
	DaysSalesOfInventory  float64 `json:"days_sales_of_inventory"`
}

// ABCXYZItem represents a product classified by Pareto ABC value and XYZ demand predictability.
type ABCXYZItem struct {
	ProductID       string  `json:"product_id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	SoldQty         int     `json:"sold_qty"`
	Revenue         float64 `json:"revenue"`
	GrossProfit     float64 `json:"gross_profit"`
	MarginPct       float64 `json:"margin_pct"`
	CurrentStock    int     `json:"current_stock"`
	BlockedCapital  float64 `json:"blocked_capital"`
	ABCClass        string  `json:"abc_class"` // A (Top 80%), B (Next 15%), C (Bottom 5%)
	XYZClass        string  `json:"xyz_class"` // X (CV<0.5), Y (0.5..1.0), Z (CV>1.0)
	CVScore         float64 `json:"cv_score"`
	RevenueSharePct float64 `json:"revenue_share_pct"`
}

// BasketPairItem represents a Market Basket Association rule (A => B) with Support, Confidence, and Lift.
type BasketPairItem struct {
	ProductAID    string  `json:"product_a_id"`
	ProductAName  string  `json:"product_a_name"`
	ProductBID    string  `json:"product_b_id"`
	ProductBName  string  `json:"product_b_name"`
	PairCount     int     `json:"pair_count"`
	SupportPct    float64 `json:"support_pct"`
	ConfidencePct float64 `json:"confidence_pct"`
	Lift          float64 `json:"lift"`
}

// PeakHourBucket aggregates counter traffic by IST hour (0-23).
type PeakHourBucket struct {
	Hour      int     `json:"hour"`
	BillCount int     `json:"bill_count"`
	Revenue   float64 `json:"revenue"`
}

// PeakDayBucket aggregates counter traffic by weekday (0=Sun..6=Sat).
type PeakDayBucket struct {
	Dow       int     `json:"dow"`
	DayName   string  `json:"day_name"`
	BillCount int     `json:"bill_count"`
	Revenue   float64 `json:"revenue"`
}

// ShopIntelligenceResponse is the unified industrial-grade analytics & CA intelligence payload.
type ShopIntelligenceResponse struct {
	FromDate            string                `json:"from_date"`
	ToDate              string                `json:"to_date"`
	PeriodDays          int                   `json:"period_days"`
	TotalBills          int                   `json:"total_bills"`
	TotalRevenue        float64               `json:"total_revenue"`
	TotalCost           float64               `json:"total_cost"`
	GrossProfit         float64               `json:"gross_profit"`
	TotalExpenses       float64               `json:"total_expenses"`
	NetPocketProfit     float64               `json:"net_pocket_profit"`
	AverageMarginPct    float64               `json:"average_margin_pct"`
	AverageBillValue    float64               `json:"average_bill_value"`
	PrevPeriodRevenue   float64               `json:"prev_period_revenue"`
	RevenueGrowthPct    float64               `json:"revenue_growth_pct"`
	Tender              TenderReconciliation  `json:"tender"`
	StockValuation      StockValuationMetrics `json:"stock_valuation"`
	DeadCapitalTotal    float64               `json:"dead_capital_total"`
	DeadCapitalItems    []*ABCXYZItem         `json:"dead_capital_items"`
	TopABCItems         []*ABCXYZItem         `json:"top_abc_items"`
	ABCSummary          map[string]int        `json:"abc_summary"`
	BasketPairs         []*BasketPairItem     `json:"basket_pairs"`
	PeakHours           []*PeakHourBucket     `json:"peak_hours"`
	PeakDays            []*PeakDayBucket      `json:"peak_days"`
}

// PulseDayPoint represents one pre-bucketed day in the 7-day trend chart.
type PulseDayPoint struct {
	Date    string  `json:"date"`
	Label   string  `json:"label"`
	Revenue float64 `json:"revenue"`
	Profit  float64 `json:"profit"`
	Bills   int     `json:"bills"`
}

// PulseInsight represents a plain-language actionable alert computed by the backend.
type PulseInsight struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"` // "high" | "medium" | "info"
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	ActionLabel string `json:"action_label"`
	ActionRoute string `json:"action_route"`
}

// PulseHeadline holds pre-computed today and monthly headline KPIs with period-over-period deltas.
type PulseHeadline struct {
	TodayRevenue     float64 `json:"today_revenue"`
	TodayProfit      float64 `json:"today_profit"`
	TodayBills       int     `json:"today_bills"`
	YesterdayRevenue float64 `json:"yesterday_revenue"`
	TodayDeltaPct    float64 `json:"today_delta_pct"`
	TodayTrend       string  `json:"today_trend"` // "up" | "down" | "flat"

	MonthRevenue     float64 `json:"month_revenue"`
	MonthGrossProfit float64 `json:"month_gross_profit"`
	MonthExpenses    float64 `json:"month_expenses"`
	MonthNetProfit   float64 `json:"month_net_profit"`
	PrevMonthRevenue float64 `json:"prev_month_revenue"`
	MonthDeltaPct    float64 `json:"month_delta_pct"`
	MonthTrend       string  `json:"month_trend"` // "up" | "down" | "flat"
}

// PulseOperations holds summarized inventory and udhar risk counts.
type PulseOperations struct {
	LowStockCount    int     `json:"low_stock_count"`
	OutOfStockCount  int     `json:"out_of_stock_count"`
	DeadStockCount   int     `json:"dead_stock_count"`
	KhataOutstanding float64 `json:"khata_outstanding"`
	KhataCustomers   int     `json:"khata_customers"`
}

// DukaanPulseResponse is the unified, plain-language executive summary for the merchant dashboard.
type DukaanPulseResponse struct {
	Headline        PulseHeadline   `json:"headline"`
	SevenDayTrend   []PulseDayPoint `json:"seven_day_trend"`
	BestDayLabel    string          `json:"best_day_label"`
	BestDayRevenue  float64         `json:"best_day_revenue"`
	AvgDailyRevenue float64         `json:"avg_daily_revenue"`
	ChartMaxY       float64         `json:"chart_max_y"`
	Operations      PulseOperations `json:"operations"`
	HealthScore     int             `json:"health_score"`
	HealthStatus    string          `json:"health_status"` // "Healthy" | "Needs Attention" | "Action Required"
	PlainSummary    string          `json:"plain_summary"`
	Insights        []PulseInsight  `json:"insights"`
	GeneratedAt     string          `json:"generated_at"`
}

// AdvisorItemDecision holds CA-style stock sizing, 4-quadrant classification, and plain recommendation.
type AdvisorItemDecision struct {
	ProductID        string  `json:"product_id"`
	Name             string  `json:"name"`
	Category         string  `json:"category"`
	SKU              string  `json:"sku"`
	CurrentStock     int     `json:"current_stock"`
	AvgDailySales    float64 `json:"avg_daily_sales"`
	DaysOfSupply     float64 `json:"days_of_supply"`
	RecommendedStock int     `json:"recommended_stock"`
	SuggestedOrder   int     `json:"suggested_order_qty"`
	Price            float64 `json:"price"`
	CostPrice        float64 `json:"cost_price"`
	MarginPct        float64 `json:"margin_pct"`
	Sold30D          int     `json:"sold_30d"`
	Revenue30D       float64 `json:"revenue_30d"`
	Profit30D        float64 `json:"profit_30d"`
	BlockedCapital   float64 `json:"blocked_capital"`
	Quadrant         string  `json:"quadrant"`     // "star" | "crowd_puller" | "hidden_gem" | "cash_trap"
	StockAction      string  `json:"stock_action"` // "reorder_now" | "keep_stocked" | "overstocked" | "drop_and_clear"
	WhySellingTag    string  `json:"why_selling_tag"`
	Recommendation   string  `json:"recommendation"`
}

// CapitalSummary summarizes working capital locked in inventory and khata vs recoverable cash.
type CapitalSummary struct {
	InventoryCapital     float64 `json:"inventory_capital"`
	OverstockCapital     float64 `json:"overstock_capital"`
	DeadStockCapital     float64 `json:"dead_stock_capital"`
	KhataReceivables     float64 `json:"khata_receivables"`
	CashReleasePotential float64 `json:"cash_release_potential"`
}

// SeasonalHint highlights products or categories showing significant YoY or MoM demand shifts.
type SeasonalHint struct {
	ProductID      string  `json:"product_id"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	CurrentQty     int     `json:"current_qty"`
	ComparisonQty  int     `json:"comparison_qty"`
	GrowthMultiple float64 `json:"growth_multiple"`
	ComparisonType string  `json:"comparison_type"` // "yoy" | "mom"
	Hint           string  `json:"hint"`
}

// SmartComboSuggestion pairs a fast-moving anchor product with a slow/cash-trap item to liquidate blocked capital.
type SmartComboSuggestion struct {
	AnchorProductID   string  `json:"anchor_product_id"`
	AnchorName        string  `json:"anchor_name"`
	SlowProductID     string  `json:"slow_product_id"`
	SlowName          string  `json:"slow_name"`
	RegularTotal      float64 `json:"regular_total"`
	BundlePrice       float64 `json:"bundle_price"`
	CustomerSavings   float64 `json:"customer_savings"`
	CapitalToUnlock   float64 `json:"capital_to_unlock"`
	Pitch             string  `json:"pitch"`
}

// ShrinkageAlert flags unusual bill cancellations or manual counter discounts for owner review.
type ShrinkageAlert struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // "high" | "medium"
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

// BusinessAdvisorResponse provides CA & Business Analyst decisions on what sells, why, how much to stock, and what to drop.
type BusinessAdvisorResponse struct {
	Capital          CapitalSummary         `json:"capital"`
	Stars            []AdvisorItemDecision  `json:"stars"`
	CrowdPullers     []AdvisorItemDecision  `json:"crowd_pullers"`
	HiddenGems       []AdvisorItemDecision  `json:"hidden_gems"`
	CashTraps        []AdvisorItemDecision  `json:"cash_traps"`
	ReorderNow       []AdvisorItemDecision  `json:"reorder_now"`
	SeasonalHints    []SeasonalHint         `json:"seasonal_hints"`
	SmartCombos      []SmartComboSuggestion `json:"smart_combos"`
	ShrinkageAlerts  []ShrinkageAlert       `json:"shrinkage_alerts"`
	ExecutiveVerdict string                 `json:"executive_verdict"`
	GeneratedAt      string                 `json:"generated_at"`
}

// EveningDigestResponse holds a ready-to-share end-of-day summary for the shop owner.
type EveningDigestResponse struct {
	Date         string   `json:"date"`
	ShopName     string   `json:"shop_name"`
	TodayBills   int      `json:"today_bills"`
	TodaySales   float64  `json:"today_sales"`
	CashSales    float64  `json:"cash_sales"`
	UPISales     float64  `json:"upi_sales"`
	KhataSales   float64  `json:"khata_sales"`
	TodayExpense float64  `json:"today_expenses"`
	NetProfit    float64  `json:"net_profit"`
	TopReorders  []string `json:"top_reorders"`
	WhatsAppText string   `json:"whatsapp_text"`
	GeneratedAt  string   `json:"generated_at"`
}

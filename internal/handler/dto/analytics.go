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

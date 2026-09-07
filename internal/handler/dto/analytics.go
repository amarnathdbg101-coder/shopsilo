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

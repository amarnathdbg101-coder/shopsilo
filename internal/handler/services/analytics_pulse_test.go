package services

import (
	"testing"
	"time"

	"shopMe/internal/handler/dto"
)

func TestComputeTrendDelta(t *testing.T) {
	cases := []struct {
		name      string
		current   float64
		previous  float64
		wantPct   float64
		wantTrend string
	}{
		{name: "up from positive", current: 1120, previous: 1000, wantPct: 12.0, wantTrend: "up"},
		{name: "down from positive", current: 850, previous: 1000, wantPct: -15.0, wantTrend: "down"},
		{name: "flat within 1 pct", current: 1005, previous: 1000, wantPct: 0.5, wantTrend: "flat"},
		{name: "zero previous with sales", current: 500, previous: 0, wantPct: 100.0, wantTrend: "up"},
		{name: "both zero", current: 0, previous: 0, wantPct: 0.0, wantTrend: "flat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pct, trend := computeTrendDelta(tc.current, tc.previous)
			if pct != tc.wantPct || trend != tc.wantTrend {
				t.Fatalf("got (%v, %q), want (%v, %q)", pct, trend, tc.wantPct, tc.wantTrend)
			}
		})
	}
}

func TestBuildDukaanPulseResponse_HealthyAndActionable(t *testing.T) {
	now := time.Date(2026, 11, 27, 15, 0, 0, 0, time.Local)
	trend := []dto.PulseDayPoint{
		{Date: "2026-11-21", Label: "Sat ", Revenue: 5200, Profit: 900, Bills: 18},
		{Date: "2026-11-22", Label: "Sun", Revenue: 3100, Profit: 600, Bills: 10},
	}

	// Case 1: Clean store -> Healthy score 100, positive fallback insight
	healthy := buildDukaanPulseResponse(
		&dto.PulseHeadline{
			TodayRevenue:     4250,
			TodayProfit:      850,
			TodayBills:       14,
			YesterdayRevenue: 3800,
			MonthRevenue:     92000,
			MonthGrossProfit: 18000,
			PrevMonthRevenue: 80000,
		},
		trend,
		&dto.PulseOperations{},
		4000,
		now,
	)

	if healthy.HealthScore != 100 || healthy.HealthStatus != "Healthy" {
		t.Fatalf("expected Healthy (100), got %s (%d)", healthy.HealthStatus, healthy.HealthScore)
	}
	if healthy.Headline.MonthNetProfit != 14000 {
		t.Fatalf("expected MonthNetProfit 14000, got %v", healthy.Headline.MonthNetProfit)
	}
	if healthy.BestDayLabel != "Sat" || healthy.BestDayRevenue != 5200 {
		t.Fatalf("expected best day Sat (5200), got %q (%v)", healthy.BestDayLabel, healthy.BestDayRevenue)
	}
	if healthy.ChartMaxY < 5200 {
		t.Fatalf("expected ChartMaxY >= 5200, got %v", healthy.ChartMaxY)
	}
	if len(healthy.Insights) != 1 || healthy.Insights[0].ID != "all_good" {
		t.Fatalf("expected all_good insight, got %+v", healthy.Insights)
	}

	// Case 2: Store with stock & khata issues -> prioritized insights & reduced score
	risk := buildDukaanPulseResponse(
		&dto.PulseHeadline{TodayRevenue: 1200, TodayBills: 4, MonthGrossProfit: 2000},
		trend,
		&dto.PulseOperations{
			OutOfStockCount:  4,
			LowStockCount:    5,
			DeadStockCount:   3,
			KhataOutstanding: 30000,
			KhataCustomers:   7,
		},
		5000,
		now,
	)

	if risk.HealthScore >= 75 || risk.HealthStatus == "Healthy" {
		t.Fatalf("expected degraded health status, got %s (%d)", risk.HealthStatus, risk.HealthScore)
	}
	if len(risk.Insights) != 4 || risk.Insights[0].ID != "out_of_stock" {
		t.Fatalf("expected 4 prioritized insights starting with out_of_stock, got %+v", risk.Insights)
	}
}

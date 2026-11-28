package services

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
)

func TestAnalyticsAndAdvisor_2000RandomizedStressCases(t *testing.T) {
	rng := rand.New(rand.NewSource(2026))
	now := time.Date(2026, 11, 28, 9, 30, 0, 0, time.UTC)

	for i := 0; i < 2000; i++ {
		prodCount := rng.Intn(40)
		raw := make([]repository.AdvisorRawProduct, 0, prodCount)
		for j := 0; j < prodCount; j++ {
			price := float64(rng.Intn(2000))
			if rng.Intn(10) == 0 {
				price = 0 // unpredictable zero price
			}
			cost := float64(rng.Intn(2000))
			if rng.Intn(8) == 0 {
				cost = 0 // missing cost price
			}
			raw = append(raw, repository.AdvisorRawProduct{
				ProductID:    fmt.Sprintf("p-%d-%d", i, j),
				Name:         fmt.Sprintf("Product %d", j),
				Price:        price,
				CostPrice:    cost,
				CurrentStock: rng.Intn(200),
				Sold30D:      rng.Intn(150),
				Sold45D:      rng.Intn(200),
				CoBillCount:  rng.Intn(30),
				AgeDays:      rng.Intn(365),
			})
		}

		capital := dto.CapitalSummary{
			InventoryCapital: float64(rng.Intn(500000)),
			KhataReceivables: float64(rng.Intn(100000)),
		}

		adv := buildBusinessAdvisorResponse(raw, capital, now)
		if adv == nil {
			t.Fatalf("iteration %d: nil advisor response", i)
		}
		if adv.Capital.CashReleasePotential < 0 {
			t.Fatalf("iteration %d: negative CashReleasePotential %v", i, adv.Capital.CashReleasePotential)
		}
		if len(adv.Stars) > 10 || len(adv.CrowdPullers) > 10 || len(adv.HiddenGems) > 10 || len(adv.CashTraps) > 10 || len(adv.ReorderNow) > 10 {
			t.Fatalf("iteration %d: advisor bucket exceeded cap of 10", i)
		}

		trend := make([]dto.PulseDayPoint, 7)
		for d := 0; d < 7; d++ {
			trend[d] = dto.PulseDayPoint{
				Date:    fmt.Sprintf("2026-11-%02d", 22+d),
				Label:   "Mon",
				Revenue: float64(rng.Intn(50000)),
				Profit:  float64(rng.Intn(15000) - 3000),
				Bills:   rng.Intn(120),
			}
		}

		pulse := buildDukaanPulseResponse(
			&dto.PulseHeadline{
				TodayRevenue:     float64(rng.Intn(50000)),
				YesterdayRevenue: float64(rng.Intn(50000)),
				MonthRevenue:     float64(rng.Intn(500000)),
				PrevMonthRevenue: float64(rng.Intn(500000)),
				MonthGrossProfit: float64(rng.Intn(100000) - 20000),
			},
			trend,
			&dto.PulseOperations{
				LowStockCount:    rng.Intn(50),
				OutOfStockCount:  rng.Intn(50),
				DeadStockCount:   rng.Intn(50),
				KhataOutstanding: float64(rng.Intn(100000)),
				KhataCustomers:   rng.Intn(80),
			},
			float64(rng.Intn(30000)),
			now,
		)
		if pulse.HealthScore < 0 || pulse.HealthScore > 100 {
			t.Fatalf("iteration %d: HealthScore out of [0,100]: %d", i, pulse.HealthScore)
		}
		if pulse.ChartMaxY <= 0 {
			t.Fatalf("iteration %d: ChartMaxY must be > 0, got %v", i, pulse.ChartMaxY)
		}

		digest := buildEveningDigestResponse(
			"",
			rng.Intn(200),
			float64(rng.Intn(50000)),
			float64(rng.Intn(20000)),
			float64(rng.Intn(20000)),
			float64(rng.Intn(10000)),
			float64(rng.Intn(10000)-2000),
			float64(rng.Intn(3000)),
			nil,
			[]string{"hi", "en"}[rng.Intn(2)],
			now,
		)
		if digest.WhatsAppText == "" {
			t.Fatalf("iteration %d: empty digest WhatsAppText", i)
		}
	}
}

// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"sort"
	"strings"
	"sync"
	"time"
)

type AnalyticsService struct {
	productRepo *repository.ProductRepo
	shopRepo    *repository.ShopRepo
	expenseRepo *repository.ExpenseRepo
}

func NewAnalyticsService(productRepo *repository.ProductRepo, shopRepo *repository.ShopRepo, expenseRepo *repository.ExpenseRepo) *AnalyticsService {
	return &AnalyticsService{
		productRepo: productRepo,
		shopRepo:    shopRepo,
		expenseRepo: expenseRepo,
	}
}

// GetMonthlyProfit returns the revenue, wholesale cost, and net profit for the shop in a given month.
func (s *AnalyticsService) GetMonthlyProfit(ctx context.Context, shopOwnerUserID string, year, month int) (*dto.MonthlyProfitResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if year <= 0 {
		year = time.Now().Year()
	}
	if month <= 0 || month > 12 {
		month = int(time.Now().Month())
	}

	var (
		profit    *dto.MonthlyProfitResponse
		profitErr error
		totalExp  float64
		breakdown map[string]float64
		expErr    error
		wg        sync.WaitGroup
	)

	wg.Add(2)

	// Goroutine 1: Sales revenue, cost, and item volume analytics
	go func() {
		defer wg.Done()
		profit, profitErr = s.productRepo.GetMonthlyProfitAnalytics(ctx, shop.ID, year, month)
	}()

	// Goroutine 2: Store operational expenses and category breakdown
	go func() {
		defer wg.Done()
		if s.expenseRepo != nil {
			totalExp, breakdown, expErr = s.expenseRepo.GetTotalExpensesForMonth(ctx, shop.ID, year, month)
		}
	}()

	wg.Wait()

	if profitErr != nil {
		return nil, profitErr
	}

	if expErr == nil && profit != nil {
		profit.TotalExpenses = totalExp
		profit.ExpenseBreakdown = breakdown
		profit.GrossProfit = profit.NetProfit
		profit.NetPocketProfit = profit.GrossProfit - totalExp
	}

	return profit, nil
}

// GetProductMatrix returns top profitable, lowest margin, old dead stock, and new arrival items.
func (s *AnalyticsService) GetProductMatrix(ctx context.Context, shopOwnerUserID string) (*dto.ProductMatrixResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.productRepo.GetProductPerformanceMatrix(ctx, shop.ID)
}

// GetShopIntelligence returns industrial-grade retail KPIs (GMROI, Turnover, ABC-XYZ, Market Basket, Peak Hours & Tender Reconciliation).
func (s *AnalyticsService) GetShopIntelligence(ctx context.Context, shopOwnerUserID string, fromDate, toDate time.Time) (*dto.ShopIntelligenceResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.productRepo.GetShopIntelligence(ctx, shop.ID, fromDate, toDate)
}

// ExportCACSV generates UTF-8 BOM CSV bytes for sales, expenses, stock valuation, or khata receivables.
func (s *AnalyticsService) ExportCACSV(ctx context.Context, shopOwnerUserID, exportType string, fromDate, toDate time.Time) ([]byte, string, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, "", ErrShopNotFound
		}
		return nil, "", err
	}

	return s.productRepo.ExportCACSV(ctx, shop.ID, exportType, fromDate, toDate)
}

type pulseCacheEntry struct {
	payload   *dto.DukaanPulseResponse
	expiresAt time.Time
}

var (
	pulseCacheMu sync.RWMutex
	pulseCache   = make(map[string]pulseCacheEntry)
)

const pulseCacheTTL = 2 * time.Minute

// GetDukaanPulse returns a pre-digested, plain-language executive summary for the merchant dashboard
// with a 2-minute per-shop cache and parallel SQL aggregation.
func (s *AnalyticsService) GetDukaanPulse(ctx context.Context, shopOwnerUserID string) (*dto.DukaanPulseResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	now := time.Now()
	pulseCacheMu.RLock()
	if entry, ok := pulseCache[shop.ID]; ok && now.Before(entry.expiresAt) {
		pulseCacheMu.RUnlock()
		return entry.payload, nil
	}
	pulseCacheMu.RUnlock()

	var (
		headline *dto.PulseHeadline
		trend    []dto.PulseDayPoint
		ops      *dto.PulseOperations
		metricsErr error
		totalExp float64
		expErr   error
		wg       sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		headline, trend, ops, metricsErr = s.productRepo.GetDukaanPulseMetrics(ctx, shop.ID, now)
	}()
	go func() {
		defer wg.Done()
		if s.expenseRepo != nil {
			totalExp, _, expErr = s.expenseRepo.GetTotalExpensesForMonth(ctx, shop.ID, now.Year(), int(now.Month()))
		}
	}()
	wg.Wait()

	if metricsErr != nil {
		return nil, metricsErr
	}
	if expErr != nil {
		totalExp = 0
	}

	res := buildDukaanPulseResponse(headline, trend, ops, totalExp, now)

	pulseCacheMu.Lock()
	if len(pulseCache) > 500 {
		for k, v := range pulseCache {
			if now.After(v.expiresAt) {
				delete(pulseCache, k)
			}
		}
	}
	pulseCache[shop.ID] = pulseCacheEntry{
		payload:   res,
		expiresAt: now.Add(pulseCacheTTL),
	}
	pulseCacheMu.Unlock()

	return res, nil
}

func computeTrendDelta(current, previous float64) (float64, string) {
	if previous <= 0 {
		if current > 0 {
			return 100.0, "up"
		}
		return 0.0, "flat"
	}
	pct := math.Round(((current-previous)/previous)*1000) / 10
	if pct >= 1.0 {
		return pct, "up"
	}
	if pct <= -1.0 {
		return pct, "down"
	}
	return pct, "flat"
}

func buildDukaanPulseResponse(
	headline *dto.PulseHeadline,
	trend []dto.PulseDayPoint,
	ops *dto.PulseOperations,
	monthExpenses float64,
	now time.Time,
) *dto.DukaanPulseResponse {
	if headline == nil {
		headline = &dto.PulseHeadline{}
	}
	if ops == nil {
		ops = &dto.PulseOperations{}
	}

	headline.TodayDeltaPct, headline.TodayTrend = computeTrendDelta(headline.TodayRevenue, headline.YesterdayRevenue)
	headline.MonthExpenses = monthExpenses
	headline.MonthNetProfit = headline.MonthGrossProfit - monthExpenses
	headline.MonthDeltaPct, headline.MonthTrend = computeTrendDelta(headline.MonthRevenue, headline.PrevMonthRevenue)

	var bestLabel string
	var bestRev, sumRev, maxRev float64
	for i := range trend {
		trend[i].Label = strings.TrimSpace(trend[i].Label)
		sumRev += trend[i].Revenue
		if trend[i].Revenue > bestRev {
			bestRev = trend[i].Revenue
			bestLabel = trend[i].Label
		}
		if trend[i].Revenue > maxRev {
			maxRev = trend[i].Revenue
		}
	}

	avgDaily := 0.0
	if len(trend) > 0 {
		avgDaily = math.Round((sumRev/float64(len(trend)))*100) / 100
	}
	chartMaxY := 1000.0
	if maxRev > 0 {
		step := 500.0
		if maxRev > 10000 {
			step = 2000.0
		}
		chartMaxY = math.Ceil((maxRev*1.15)/step) * step
	}

	score := 100
	insights := make([]dto.PulseInsight, 0, 4)

	if ops.OutOfStockCount > 0 {
		score -= minInt(ops.OutOfStockCount*4, 25)
		insights = append(insights, dto.PulseInsight{
			ID:          "out_of_stock",
			Severity:    "high",
			Title:       fmt.Sprintf("%d items out of stock hain", ops.OutOfStockCount),
			Subtitle:    "Selling items khatam hone se billing aur customer miss ho sakte hain.",
			ActionLabel: "Restock Karein",
			ActionRoute: "/inventory/low-stock",
		})
	}

	if ops.LowStockCount > 0 {
		score -= minInt(ops.LowStockCount*2, 15)
		insights = append(insights, dto.PulseInsight{
			ID:          "low_stock",
			Severity:    "medium",
			Title:       fmt.Sprintf("%d items low stock par hain", ops.LowStockCount),
			Subtitle:    "Mandi order sheet check karke jaldi reorder kar lein.",
			ActionLabel: "View Low Stock",
			ActionRoute: "/inventory/low-stock",
		})
	}

	if ops.KhataOutstanding > 0 {
		if ops.KhataOutstanding >= 25000 {
			score -= 15
		} else if ops.KhataOutstanding >= 5000 {
			score -= 8
		}
		insights = append(insights, dto.PulseInsight{
			ID:          "khata_outstanding",
			Severity:    "medium",
			Title:       fmt.Sprintf("₹%.0f Udhar %d customers par pending hai", ops.KhataOutstanding, ops.KhataCustomers),
			Subtitle:    "Pending udhar par WhatsApp reminder bhejne se cashflow fast hota hai.",
			ActionLabel: "Khata Dekhein",
			ActionRoute: "/khata",
		})
	}

	if ops.DeadStockCount > 0 {
		score -= minInt(ops.DeadStockCount*2, 15)
		if len(insights) < 4 {
			insights = append(insights, dto.PulseInsight{
				ID:          "dead_stock",
				Severity:    "info",
				Title:       fmt.Sprintf("%d items 45+ din se nahi bike", ops.DeadStockCount),
				Subtitle:    "Dead stock par chhota discount ya combo offer lagakar cash free karein.",
				ActionLabel: "Dead Stock Dekhein",
				ActionRoute: "/analytics/matrix",
			})
		}
	}

	if headline.MonthNetProfit < 0 {
		score -= 15
	}
	if score < 0 {
		score = 0
	}

	healthStatus := "Healthy"
	if score < 45 {
		healthStatus = "Action Required"
	} else if score < 75 {
		healthStatus = "Needs Attention"
	}

	if len(insights) == 0 {
		insights = append(insights, dto.PulseInsight{
			ID:          "all_good",
			Severity:    "info",
			Title:       "Dukaan bilkul smooth chal rahi hai",
			Subtitle:    "Stock aur udhar control me hain — aaj ki billing par focus karein.",
			ActionLabel: "POS Kholein",
			ActionRoute: "/pos",
		})
	}

	summary := fmt.Sprintf("Aaj ki bikri ₹%.0f (%d bills) aur is mahine ka net profit ₹%.0f hai.",
		headline.TodayRevenue, headline.TodayBills, headline.MonthNetProfit)
	if bestLabel != "" && bestRev > 0 {
		summary += fmt.Sprintf(" Pichle 7 din me %s sabse best (₹%.0f) raha.", bestLabel, bestRev)
	}

	return &dto.DukaanPulseResponse{
		Headline:        *headline,
		SevenDayTrend:   trend,
		BestDayLabel:    bestLabel,
		BestDayRevenue:  bestRev,
		AvgDailyRevenue: avgDaily,
		ChartMaxY:       chartMaxY,
		Operations:      *ops,
		HealthScore:     score,
		HealthStatus:    healthStatus,
		PlainSummary:    summary,
		Insights:        insights,
		GeneratedAt:     now.Format(time.RFC3339),
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type advisorCacheEntry struct {
	payload   *dto.BusinessAdvisorResponse
	expiresAt time.Time
}

type seasonalCacheEntry struct {
	hints     []dto.SeasonalHint
	expiresAt time.Time
}

var (
	advisorCacheMu  sync.RWMutex
	advisorCache    = make(map[string]advisorCacheEntry)
	seasonalCacheMu sync.RWMutex
	seasonalCache   = make(map[string]seasonalCacheEntry)
)

const (
	advisorCacheTTL  = 3 * time.Minute
	seasonalCacheTTL = 15 * time.Minute
)

// InvalidateAnalyticsCache clears per-shop pulse and advisor caches immediately after mutations
// (e.g. POS sale, bill cancel, expense entry, or stock adjustment).
func InvalidateAnalyticsCache(shopID string) {
	if shopID == "" {
		return
	}
	pulseCacheMu.Lock()
	delete(pulseCache, shopID)
	pulseCacheMu.Unlock()

	advisorCacheMu.Lock()
	delete(advisorCache, shopID)
	advisorCacheMu.Unlock()
}

// GetBusinessAdvisor returns CA-grade product classification (Stars, Crowd Pullers, Hidden Gems, Cash Traps),
// dynamic reorder quantities, seasonal hints, and working-capital release opportunities.
func (s *AnalyticsService) GetBusinessAdvisor(ctx context.Context, shopOwnerUserID string) (*dto.BusinessAdvisorResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	now := time.Now()
	advisorCacheMu.RLock()
	if entry, ok := advisorCache[shop.ID]; ok && now.Before(entry.expiresAt) {
		advisorCacheMu.RUnlock()
		return entry.payload, nil
	}
	advisorCacheMu.RUnlock()

	var cachedSeasonal []dto.SeasonalHint
	seasonalHit := false
	seasonalCacheMu.RLock()
	if sEntry, ok := seasonalCache[shop.ID]; ok && now.Before(sEntry.expiresAt) {
		cachedSeasonal = sEntry.hints
		seasonalHit = true
	}
	seasonalCacheMu.RUnlock()

	var (
		raw        []repository.AdvisorRawProduct
		capital    dto.CapitalSummary
		statsErr   error
		hints      []dto.SeasonalHint
		hintsErr   error
		shrinkSig  repository.ShrinkageSignals
		shrinkErr  error
		wg         sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		raw, capital, statsErr = s.productRepo.GetAdvisorProductStats(ctx, shop.ID, now)
	}()
	go func() {
		defer wg.Done()
		shrinkSig, shrinkErr = s.productRepo.GetShrinkageSignals(ctx, shop.ID, now)
	}()

	if !seasonalHit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hints, hintsErr = s.productRepo.GetSeasonalDemandHints(ctx, shop.ID, now)
		}()
	} else {
		hints = cachedSeasonal
	}

	wg.Wait()
	if statsErr != nil {
		return nil, statsErr
	}
	if !seasonalHit && hintsErr == nil {
		seasonalCacheMu.Lock()
		if len(seasonalCache) > 500 {
			for k, v := range seasonalCache {
				if now.After(v.expiresAt) {
					delete(seasonalCache, k)
				}
			}
		}
		seasonalCache[shop.ID] = seasonalCacheEntry{
			hints:     hints,
			expiresAt: now.Add(seasonalCacheTTL),
		}
		seasonalCacheMu.Unlock()
	}

	res := buildBusinessAdvisorResponse(raw, capital, now)
	if hints == nil {
		hints = []dto.SeasonalHint{}
	}
	res.SeasonalHints = hints
	if shrinkErr == nil {
		res.ShrinkageAlerts = buildShrinkageAlerts(shrinkSig)
	} else {
		res.ShrinkageAlerts = []dto.ShrinkageAlert{}
	}

	advisorCacheMu.Lock()
	if len(advisorCache) > 500 {
		for k, v := range advisorCache {
			if now.After(v.expiresAt) {
				delete(advisorCache, k)
			}
		}
	}
	advisorCache[shop.ID] = advisorCacheEntry{
		payload:   res,
		expiresAt: now.Add(advisorCacheTTL),
	}
	advisorCacheMu.Unlock()

	return res, nil
}

func buildBusinessAdvisorResponse(raw []repository.AdvisorRawProduct, capital dto.CapitalSummary, now time.Time) *dto.BusinessAdvisorResponse {
	var stars, pullers, gems, traps, reorderNow []dto.AdvisorItemDecision

	for _, it := range raw {
		avgDaily := math.Round((float64(it.Sold30D)/30.0)*100) / 100
		revenue30 := float64(it.Sold30D) * it.Price
		unitProfit := it.Price - it.CostPrice
		if it.CostPrice <= 0 {
			unitProfit = 0
		}
		profit30 := float64(it.Sold30D) * unitProfit

		var marginPct float64
		if it.Price > 0 && it.CostPrice > 0 {
			marginPct = math.Round(((it.Price-it.CostPrice)/it.Price)*1000) / 10
		}

		daysOfSupply := 0.0
		if avgDaily > 0 {
			daysOfSupply = math.Round((float64(it.CurrentStock)/avgDaily)*10) / 10
		} else if it.CurrentStock > 0 {
			daysOfSupply = 999
		}

		blockedCap := math.Round((float64(it.CurrentStock)*it.CostPrice)*100) / 100
		isDeadStock := it.Sold45D == 0 && it.AgeDays >= 45 && it.CurrentStock > 0
		isHighVelocity := avgDaily >= 0.5
		isHighMargin := marginPct >= 12.0

		d := dto.AdvisorItemDecision{
			ProductID:      it.ProductID,
			Name:           it.Name,
			Category:       it.Category,
			SKU:            it.SKU,
			CurrentStock:   it.CurrentStock,
			AvgDailySales:  avgDaily,
			DaysOfSupply:   daysOfSupply,
			Price:          it.Price,
			CostPrice:      it.CostPrice,
			MarginPct:      marginPct,
			Sold30D:        it.Sold30D,
			Revenue30D:     math.Round(revenue30*100) / 100,
			Profit30D:      math.Round(profit30*100) / 100,
			BlockedCapital: blockedCap,
		}

		switch {
		case isDeadStock || (!isHighVelocity && !isHighMargin && daysOfSupply > 45 && it.CurrentStock > 0):
			d.Quadrant = "cash_trap"
			d.StockAction = "drop_and_clear"
			d.RecommendedStock = 0
			d.SuggestedOrder = 0
			if isDeadStock {
				capital.DeadStockCapital += blockedCap
				d.WhySellingTag = "45+ din se zero bikri — cash block hai"
			} else {
				capital.OverstockCapital += blockedCap
				d.WhySellingTag = "Slow movement + kam margin"
			}
			d.Recommendation = fmt.Sprintf("15%% discount ya combo me clear karein aur dubara na rakhein (₹%.0f cash free hoga).", blockedCap)
			traps = append(traps, d)

		case isHighVelocity && isHighMargin:
			d.Quadrant = "star"
			d.RecommendedStock = maxInt(int(math.Ceil(avgDaily*10)), 3)
			if it.CoBillCount >= 5 {
				d.WhySellingTag = fmt.Sprintf("Fast mover + %.1f%% margin + %d combo bills", marginPct, it.CoBillCount)
			} else {
				d.WhySellingTag = fmt.Sprintf("High profit star (%.1f%% margin, %.1f/day)", marginPct, avgDaily)
			}
			applyStockSizing(&d, &capital)
			stars = append(stars, d)

		case isHighVelocity && !isHighMargin:
			d.Quadrant = "crowd_puller"
			d.RecommendedStock = maxInt(int(math.Ceil(avgDaily*10)), 5)
			if it.CoBillCount >= 3 {
				d.WhySellingTag = fmt.Sprintf("Traffic anchor: %d multi-item bills me saath bika", it.CoBillCount)
			} else {
				d.WhySellingTag = fmt.Sprintf("High-volume daily staple (%.1f/day)", avgDaily)
			}
			applyStockSizing(&d, &capital)
			pullers = append(pullers, d)

		default:
			d.Quadrant = "hidden_gem"
			d.RecommendedStock = maxInt(int(math.Ceil(avgDaily*7)), 2)
			if it.Sold30D == 0 {
				d.WhySellingTag = "Naya/niche item — visibility badhayein"
			} else {
				d.WhySellingTag = fmt.Sprintf("Achha margin (%.1f%%), kam volume — upsell karein", marginPct)
			}
			applyStockSizing(&d, &capital)
			gems = append(gems, d)
		}

		if d.StockAction == "reorder_now" {
			reorderNow = append(reorderNow, d)
		}
	}

	sort.Slice(stars, func(i, j int) bool { return stars[i].Profit30D > stars[j].Profit30D })
	sort.Slice(pullers, func(i, j int) bool { return pullers[i].Sold30D > pullers[j].Sold30D })
	sort.Slice(gems, func(i, j int) bool { return gems[i].MarginPct > gems[j].MarginPct })
	sort.Slice(traps, func(i, j int) bool { return traps[i].BlockedCapital > traps[j].BlockedCapital })
	sort.Slice(reorderNow, func(i, j int) bool { return reorderNow[i].DaysOfSupply < reorderNow[j].DaysOfSupply })

	stars = capAdvisorSlice(stars, 10)
	pullers = capAdvisorSlice(pullers, 10)
	gems = capAdvisorSlice(gems, 10)
	traps = capAdvisorSlice(traps, 10)
	reorderNow = capAdvisorSlice(reorderNow, 10)

	capital.InventoryCapital = math.Round(capital.InventoryCapital*100) / 100
	capital.OverstockCapital = math.Round(capital.OverstockCapital*100) / 100
	capital.DeadStockCapital = math.Round(capital.DeadStockCapital*100) / 100
	capital.KhataReceivables = math.Round(capital.KhataReceivables*100) / 100
	capital.CashReleasePotential = math.Round((capital.DeadStockCapital+capital.OverstockCapital+(capital.KhataReceivables*0.5))*100) / 100

	verdict := fmt.Sprintf("Total ₹%.0f stock me aur ₹%.0f udhar me laga hai.", capital.InventoryCapital, capital.KhataReceivables)
	if capital.CashReleasePotential > 0 {
		verdict += fmt.Sprintf(" Dead/overstock clearance aur udhar recovery se ₹%.0f cash free ho sakta hai.", capital.CashReleasePotential)
	}
	if len(reorderNow) > 0 {
		verdict += fmt.Sprintf(" %d fast-moving items turant reorder karne hain.", len(reorderNow))
	}

	return &dto.BusinessAdvisorResponse{
		Capital:          capital,
		Stars:            stars,
		CrowdPullers:     pullers,
		HiddenGems:       gems,
		CashTraps:        traps,
		ReorderNow:       reorderNow,
		SmartCombos:      buildSmartCombos(stars, pullers, traps),
		ShrinkageAlerts:  []dto.ShrinkageAlert{},
		ExecutiveVerdict: verdict,
		GeneratedAt:      now.Format(time.RFC3339),
	}
}

func buildSmartCombos(stars, pullers, traps []dto.AdvisorItemDecision) []dto.SmartComboSuggestion {
	if len(traps) == 0 {
		return []dto.SmartComboSuggestion{}
	}
	anchors := append([]dto.AdvisorItemDecision{}, pullers...)
	anchors = append(anchors, stars...)
	if len(anchors) == 0 {
		return []dto.SmartComboSuggestion{}
	}

	limit := minInt(len(traps), 3)
	combos := make([]dto.SmartComboSuggestion, 0, limit)
	for i := 0; i < limit; i++ {
		slow := traps[i]
		anchor := anchors[i%len(anchors)]
		if slow.Price <= 0 || anchor.Price <= 0 {
			continue
		}
		regular := math.Round((anchor.Price+slow.Price)*100) / 100
		savings := math.Round(slow.Price * 0.15)
		if savings < 5 {
			savings = 5
		}
		if savings >= slow.Price {
			savings = math.Round(slow.Price * 0.10)
		}
		bundle := math.Round((regular-savings)*100) / 100
		combos = append(combos, dto.SmartComboSuggestion{
			AnchorProductID: anchor.ProductID,
			AnchorName:      anchor.Name,
			SlowProductID:   slow.ProductID,
			SlowName:        slow.Name,
			RegularTotal:    regular,
			BundlePrice:     bundle,
			CustomerSavings: savings,
			CapitalToUnlock: slow.BlockedCapital,
			Pitch:           fmt.Sprintf("%s ke saath %s combo ₹%.0f me dein (₹%.0f bachao) — ₹%.0f fasa cash niklega.", anchor.Name, slow.Name, bundle, savings, slow.BlockedCapital),
		})
	}
	return combos
}

func buildShrinkageAlerts(sig repository.ShrinkageSignals) []dto.ShrinkageAlert {
	alerts := make([]dto.ShrinkageAlert, 0, 2)
	if sig.TodayCancelledCount >= 2 || (sig.TodayCancelledCount >= 1 && sig.Avg7DCancelledCount > 0 && float64(sig.TodayCancelledCount) >= sig.Avg7DCancelledCount*2.0) {
		alerts = append(alerts, dto.ShrinkageAlert{
			ID:       "cancelled_bills_spike",
			Severity: "high",
			Title:    fmt.Sprintf("Aaj %d bills cancel hue (₹%.0f)", sig.TodayCancelledCount, sig.TodayCancelledAmount),
			Detail:   "Counter par bill cancellation reason aur stock wapas aaya ya nahi zaroor check karein.",
		})
	}
	if sig.TodaySubtotal > 0 && sig.TodayDiscountAmount >= 100 {
		discPct := (sig.TodayDiscountAmount / sig.TodaySubtotal) * 100
		if discPct >= 8.0 {
			alerts = append(alerts, dto.ShrinkageAlert{
				ID:       "discount_spike",
				Severity: "medium",
				Title:    fmt.Sprintf("Aaj %.1f%% extra discount (₹%.0f) diya gaya", discPct, sig.TodayDiscountAmount),
				Detail:   "Normal se zyada manual discount diya gaya hai — margin bachane ke liye verify karein.",
			})
		}
	}
	return alerts
}

func applyStockSizing(d *dto.AdvisorItemDecision, capital *dto.CapitalSummary) {
	if d.AvgDailySales >= 0.3 && d.DaysOfSupply <= 4.0 {
		d.StockAction = "reorder_now"
		d.SuggestedOrder = maxInt(d.RecommendedStock-d.CurrentStock, 1)
		d.Recommendation = fmt.Sprintf("Abhi %d pcs mangwayein (sirf %.1f din ka stock bacha hai).", d.SuggestedOrder, d.DaysOfSupply)
		return
	}
	if d.DaysOfSupply > 45 && d.CurrentStock > d.RecommendedStock && d.RecommendedStock > 0 {
		d.StockAction = "overstocked"
		excessCap := float64(d.CurrentStock-d.RecommendedStock) * d.CostPrice
		capital.OverstockCapital += excessCap
		d.Recommendation = fmt.Sprintf("Abhi order na karein — %.0f din ka stock already hai (ideal: %d pcs).", d.DaysOfSupply, d.RecommendedStock)
		return
	}
	d.StockAction = "keep_stocked"
	d.Recommendation = fmt.Sprintf("Stock sahi hai (ideal: %d pcs, ~10 din cover).", d.RecommendedStock)
}

func capAdvisorSlice(items []dto.AdvisorItemDecision, limit int) []dto.AdvisorItemDecision {
	if items == nil {
		return []dto.AdvisorItemDecision{}
	}
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetEveningDigest compiles today's sales, cash/UPI/khata split, expenses, net profit,
// and top 3 morning reorder items into a ready-to-share WhatsApp owner digest.
func (s *AnalyticsService) GetEveningDigest(ctx context.Context, shopOwnerUserID, lang string) (*dto.EveningDigestResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	now := time.Now()
	var (
		bills      int
		sales      float64
		cash       float64
		upi        float64
		khata      float64
		grossProf  float64
		todayExp   float64
		digestErr  error
		advisorRes *dto.BusinessAdvisorResponse
		wg         sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		bills, sales, cash, upi, khata, grossProf, todayExp, digestErr = s.productRepo.GetTodayCashflowDigest(ctx, shop.ID, now)
	}()
	go func() {
		defer wg.Done()
		advisorRes, _ = s.GetBusinessAdvisor(ctx, shopOwnerUserID)
	}()
	wg.Wait()

	if digestErr != nil {
		return nil, digestErr
	}

	var topReorders []string
	if advisorRes != nil {
		for i, item := range advisorRes.ReorderNow {
			if i >= 3 {
				break
			}
			topReorders = append(topReorders, fmt.Sprintf("%s (+%d pcs)", item.Name, item.SuggestedOrder))
		}
	}

	return buildEveningDigestResponse(shop.Name, bills, sales, cash, upi, khata, grossProf-todayExp, todayExp, topReorders, lang, now), nil
}

func buildEveningDigestResponse(
	shopName string,
	bills int,
	sales, cash, upi, khata, netProfit, todayExp float64,
	topReorders []string,
	lang string,
	now time.Time,
) *dto.EveningDigestResponse {
	if strings.TrimSpace(shopName) == "" {
		shopName = "ShopSilo Store"
	}
	if topReorders == nil {
		topReorders = []string{}
	}

	dateStr := now.Format("02 Jan 2006")
	var b strings.Builder
	if strings.HasPrefix(strings.ToLower(lang), "hi") {
		b.WriteString(fmt.Sprintf("📊 *%s — आज की शाम की रिपोर्ट*\n", shopName))
		b.WriteString(fmt.Sprintf("📅 %s\n\n", dateStr))
		b.WriteString(fmt.Sprintf("🧾 *कुल बिल:* %d\n", bills))
		b.WriteString(fmt.Sprintf("💰 *कुल बिक्री:* ₹%.0f\n", sales))
		b.WriteString(fmt.Sprintf("   • कैश (गल्ला): ₹%.0f\n", cash))
		b.WriteString(fmt.Sprintf("   • UPI / ऑनलाइन: ₹%.0f\n", upi))
		b.WriteString(fmt.Sprintf("   • उधार (खाता): ₹%.0f\n", khata))
		b.WriteString(fmt.Sprintf("💸 *आज का खर्च:* ₹%.0f\n", todayExp))
		b.WriteString(fmt.Sprintf("🏆 *शुद्ध बचत (Net Profit):* ₹%.0f\n", netProfit))
		if len(topReorders) > 0 {
			b.WriteString("\n📦 *कल सुबह मंगाने वाले टॉप आइटम:*\n")
			for i, r := range topReorders {
				b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r))
			}
		}
	} else {
		b.WriteString(fmt.Sprintf("📊 *%s — Evening Owner Digest*\n", shopName))
		b.WriteString(fmt.Sprintf("📅 %s\n\n", dateStr))
		b.WriteString(fmt.Sprintf("🧾 *Total Bills:* %d\n", bills))
		b.WriteString(fmt.Sprintf("💰 *Today's Sales:* ₹%.0f\n", sales))
		b.WriteString(fmt.Sprintf("   • Cash in Galla: ₹%.0f\n", cash))
		b.WriteString(fmt.Sprintf("   • UPI / Online: ₹%.0f\n", upi))
		b.WriteString(fmt.Sprintf("   • Khata (Credit): ₹%.0f\n", khata))
		b.WriteString(fmt.Sprintf("💸 *Today's Expenses:* ₹%.0f\n", todayExp))
		b.WriteString(fmt.Sprintf("🏆 *Net Profit:* ₹%.0f\n", netProfit))
		if len(topReorders) > 0 {
			b.WriteString("\n📦 *Top Morning Reorders:*\n")
			for i, r := range topReorders {
				b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r))
			}
		}
	}

	return &dto.EveningDigestResponse{
		Date:         now.Format("2006-01-02"),
		ShopName:     shopName,
		TodayBills:   bills,
		TodaySales:   math.Round(sales*100) / 100,
		CashSales:    math.Round(cash*100) / 100,
		UPISales:     math.Round(upi*100) / 100,
		KhataSales:   math.Round(khata*100) / 100,
		TodayExpense: math.Round(todayExp*100) / 100,
		NetProfit:    math.Round(netProfit*100) / 100,
		TopReorders:  topReorders,
		WhatsAppText: b.String(),
		GeneratedAt:  now.Format(time.RFC3339),
	}
}

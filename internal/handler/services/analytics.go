// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
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

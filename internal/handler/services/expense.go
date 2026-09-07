// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
	"time"
)

type ExpenseService struct {
	expenseRepo *repository.ExpenseRepo
	shopRepo    *repository.ShopRepo
}

func NewExpenseService(expenseRepo *repository.ExpenseRepo, shopRepo *repository.ShopRepo) *ExpenseService {
	return &ExpenseService{
		expenseRepo: expenseRepo,
		shopRepo:    shopRepo,
	}
}

// CreateExpense records an operational store expense.
func (s *ExpenseService) CreateExpense(ctx context.Context, shopOwnerUserID string, input dto.CreateExpenseRequest) (*model.ShopExpense, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	dateStr := strings.TrimSpace(input.ExpenseDate)
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	exp := &model.ShopExpense{
		ShopID:        shop.ID,
		Category:      strings.TrimSpace(input.Category),
		Amount:        input.Amount,
		Notes:         strings.TrimSpace(input.Notes),
		PaymentMethod: strings.TrimSpace(input.PaymentMethod),
		ExpenseDate:   dateStr,
	}

	return s.expenseRepo.CreateExpense(ctx, shop.ID, exp)
}

// ListExpenses returns monthly expenses with total and category breakdown.
func (s *ExpenseService) ListExpenses(ctx context.Context, shopOwnerUserID string, year, month int, category string) (*dto.ExpenseListResponse, error) {
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

	expenses, total, catTotals, err := s.expenseRepo.ListExpenses(ctx, shop.ID, year, month, strings.TrimSpace(category))
	if err != nil {
		return nil, err
	}

	if expenses == nil {
		expenses = []model.ShopExpense{}
	}

	return &dto.ExpenseListResponse{
		Expenses:      expenses,
		TotalAmount:   total,
		CategoryTotal: catTotals,
		Month:         month,
		Year:          year,
	}, nil
}

// DeleteExpense deletes an expense.
func (s *ExpenseService) DeleteExpense(ctx context.Context, shopOwnerUserID, expenseID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	return s.expenseRepo.DeleteExpense(ctx, shop.ID, strings.TrimSpace(expenseID))
}

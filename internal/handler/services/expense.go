// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
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

// validateCreateExpenseInput validates and normalizes operational expense fields.
func validateCreateExpenseInput(input *dto.CreateExpenseRequest) error {
	input.Category = strings.TrimSpace(input.Category)
	if input.Category == "" {
		return errors.New("expense category is required")
	}
	if math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) || input.Amount <= 0 {
		return errors.New("expense amount must be greater than zero")
	}

	dateStr := strings.TrimSpace(input.ExpenseDate)
	if dateStr == "" {
		dateStr = utils.IndianNow().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return errors.New("invalid expense_date format; expected YYYY-MM-DD")
	}
	input.ExpenseDate = dateStr

	payMethod := strings.ToLower(strings.TrimSpace(input.PaymentMethod))
	if payMethod == "" {
		payMethod = "cash"
	}
	switch payMethod {
	case "cash", "upi", "bank", "card", "online", "other":
	default:
		return errors.New("invalid payment_method")
	}
	input.PaymentMethod = payMethod
	input.Notes = strings.TrimSpace(input.Notes)
	return nil
}

// CreateExpense records an operational store expense.
func (s *ExpenseService) CreateExpense(ctx context.Context, shopOwnerUserID string, input dto.CreateExpenseRequest) (*model.ShopExpense, error) {
	if err := validateCreateExpenseInput(&input); err != nil {
		return nil, err
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	exp := &model.ShopExpense{
		ShopID:        shop.ID,
		Category:      input.Category,
		Amount:        input.Amount,
		Notes:         input.Notes,
		PaymentMethod: input.PaymentMethod,
		ExpenseDate:   input.ExpenseDate,
	}

	created, err := s.expenseRepo.CreateExpense(ctx, shop.ID, exp)
	if err != nil {
		return nil, err
	}
	InvalidateAnalyticsCache(shop.ID)
	return created, nil
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

	nowIST := utils.IndianNow()
	if year <= 0 {
		year = nowIST.Year()
	}
	if month <= 0 || month > 12 {
		month = int(nowIST.Month())
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
	cleanID := strings.TrimSpace(expenseID)
	if cleanID == "" {
		return errors.New("expense_id is required")
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	if err := s.expenseRepo.DeleteExpense(ctx, shop.ID, cleanID); err != nil {
		return err
	}
	InvalidateAnalyticsCache(shop.ID)
	return nil
}

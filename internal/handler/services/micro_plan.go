// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"errors"
	"math"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// MicroPlanService manages customer micro-installment repayment schedules.
type MicroPlanService struct {
	planRepo  *repository.MicroPlanRepo
	shopRepo  *repository.ShopRepo
	userRepo  *repository.UserRepo
	khataRepo *repository.KhataRepo
	logger    *zap.Logger
}

// NewMicroPlanService initializes MicroPlanService.
func NewMicroPlanService(
	planRepo *repository.MicroPlanRepo,
	shopRepo *repository.ShopRepo,
	userRepo *repository.UserRepo,
	khataRepo *repository.KhataRepo,
	logger *zap.Logger,
) *MicroPlanService {
	return &MicroPlanService{
		planRepo:  planRepo,
		shopRepo:  shopRepo,
		userRepo:  userRepo,
		khataRepo: khataRepo,
		logger:    logger,
	}
}

// CreatePlan sets up a structured micro-repayment plan for an existing debtor.
func (s *MicroPlanService) CreatePlan(ctx context.Context, shopOwnerUserID string, req dto.CreateMicroPlanRequest) (*model.KhataMicroPlan, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	phone := strings.TrimSpace(req.CustomerPhone)
	if phone == "" {
		return nil, errors.New("customer phone number is required to start a micro-installment plan")
	}

	if req.TotalDebt <= 0 {
		return nil, errors.New("total debt must be greater than zero")
	}
	if req.DailyAmount <= 0 {
		return nil, errors.New("daily micro amount must be greater than zero")
	}

	custName := strings.TrimSpace(req.CustomerName)
	if custName == "" {
		custName = "Khata Customer"
	}

	var customerID *string
	if s.userRepo != nil {
		if user, err := s.userRepo.FindByPhone(ctx, phone); err == nil && user != nil {
			customerID = &user.ID
			if custName == "Khata Customer" && user.FullName != "" {
				custName = user.FullName
			}
		}
	}

	freq := strings.ToUpper(strings.TrimSpace(req.Frequency))
	if freq != "WEEKLY" && freq != "MONTHLY" {
		freq = "DAILY"
	}

	totalInstallments := int(math.Ceil(req.TotalDebt / req.DailyAmount))

	plan := &model.KhataMicroPlan{
		ShopID:            shop.ID,
		CustomerID:        customerID,
		CustomerPhone:     phone,
		CustomerName:      custName,
		TotalDebt:         req.TotalDebt,
		DailyAmount:       req.DailyAmount,
		Frequency:         freq,
		TotalInstallments: totalInstallments,
		PaidInstallments:  0,
		TotalPaidAmount:   0.00,
		RemainingBalance:  req.TotalDebt,
		Status:            model.PlanStatusActive,
	}

	return s.planRepo.CreatePlan(ctx, plan)
}

// RecordPayment logs a single installment receipt and updates the plan status.
func (s *MicroPlanService) RecordPayment(ctx context.Context, shopOwnerUserID, planID string, req dto.RecordMicroPaymentRequest) (*dto.MicroPlanSummaryResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	plan, err := s.planRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.ShopID != shop.ID {
		return nil, errors.New("unauthorized to record payment on this plan")
	}

	payment := &model.KhataMicroPayment{
		PlanID:               planID,
		ShopID:               shop.ID,
		Amount:               req.Amount,
		PaymentMode:          req.PaymentMode,
		TransactionReference: strings.TrimSpace(req.TransactionReference),
		Notes:                strings.TrimSpace(req.Notes),
	}

	updatedPlan, err := s.planRepo.RecordPayment(ctx, payment)
	if err != nil {
		return nil, err
	}

	// Also record settlement payment entry in khata if available
	if s.khataRepo != nil && plan.CustomerPhone != "" {
		_, _ = s.khataRepo.RecordTransaction(ctx, shop.ID, plan.CustomerPhone, plan.CustomerName, "payment", req.Amount, "Micro Installment Paid: "+req.PaymentMode, "", req.PaymentMode, "", "")
	}

	return s.GetPlanSummary(ctx, shopOwnerUserID, updatedPlan.ID)
}

// GetPlanSummary returns full plan metrics, progress percentage, and payments log.
func (s *MicroPlanService) GetPlanSummary(ctx context.Context, shopOwnerUserID, planID string) (*dto.MicroPlanSummaryResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	plan, err := s.planRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.ShopID != shop.ID {
		return nil, errors.New("unauthorized to view this plan")
	}

	payments, err := s.planRepo.ListPayments(ctx, planID)
	if err != nil {
		return nil, err
	}

	progress := 0.0
	if plan.TotalDebt > 0 {
		progress = math.Round((plan.TotalPaidAmount/plan.TotalDebt)*10000) / 100
		if progress > 100.0 {
			progress = 100.0
		}
	}

	return &dto.MicroPlanSummaryResponse{
		Plan:            plan,
		Payments:        payments,
		ProgressPercent: progress,
	}, nil
}

// ListPlans lists all micro repayment plans.
func (s *MicroPlanService) ListPlans(ctx context.Context, shopOwnerUserID, status string, limit int) ([]*model.KhataMicroPlan, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	return s.planRepo.ListPlans(ctx, shop.ID, status, limit)
}

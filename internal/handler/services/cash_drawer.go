// Package services handles business logic for cash drawer telemetry and staff shifts.
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

// CashDrawerService coordinates counter shift sessions, safe drops, and petty cash accountability.
type CashDrawerService struct {
	shopRepo   *repository.ShopRepo
	drawerRepo *repository.CashDrawerRepo
	logger     *zap.Logger
}

// NewCashDrawerService initializes CashDrawerService.
func NewCashDrawerService(
	shopRepo *repository.ShopRepo,
	drawerRepo *repository.CashDrawerRepo,
	logger *zap.Logger,
) *CashDrawerService {
	return &CashDrawerService{
		shopRepo:   shopRepo,
		drawerRepo: drawerRepo,
		logger:     logger,
	}
}

// GetCurrentStatus computes live drawer metrics, safe drop alerts, and recent events.
func (s *CashDrawerService) GetCurrentStatus(ctx context.Context, staffUserID string) (*dto.CurrentDrawerStatusResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	session, err := s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveDrawerSession) {
			// Return empty state indicating shift is closed
			return &dto.CurrentDrawerStatusResponse{
				Session:              nil,
				CurrentDrawerCash:    0,
				SafeDropRecommended:  false,
				RecentEvents:         []*model.CashDrawerEvent{},
			}, nil
		}
		return nil, err
	}

	currentCash := session.ExpectedCash
	safeDropRecommended := false
	recommendedDrop := 0.0

	// Trigger theft-protection alert when cash exceeds threshold (e.g. ₹10,000)
	if currentCash > session.SafeDropThreshold && session.SafeDropThreshold > 0 {
		safeDropRecommended = true
		// Retain a safe float (e.g. 40% of threshold or ₹3,000) and drop the rest
		retainedFloat := math.Min(3000.0, session.SafeDropThreshold*0.4)
		recommendedDrop = math.Floor((currentCash-retainedFloat)/500.0) * 500.0 // Round to neat ₹500 denominations
		if recommendedDrop <= 0 {
			recommendedDrop = currentCash - retainedFloat
		}
	}

	events, err := s.drawerRepo.ListSessionEvents(ctx, session.ID, 20)
	if err != nil {
		events = []*model.CashDrawerEvent{}
	}

	return &dto.CurrentDrawerStatusResponse{
		Session:                   session,
		CurrentDrawerCash:         currentCash,
		SafeDropRecommended:       safeDropRecommended,
		SafeDropRecommendedAmount: recommendedDrop,
		RecentEvents:              events,
	}, nil
}

// OpenSession begins a new shift with the given opening float.
func (s *CashDrawerService) OpenSession(ctx context.Context, staffUserID string, req dto.OpenDrawerSessionRequest) (*model.CashDrawerSession, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	threshold := 10000.00
	if req.SafeDropThreshold != nil && *req.SafeDropThreshold > 0 {
		threshold = *req.SafeDropThreshold
	}

	session := &model.CashDrawerSession{
		ShopID:            shop.ID,
		StaffUserID:       staffUserID,
		ShiftName:         strings.TrimSpace(req.ShiftName),
		OpeningCash:       req.OpeningCash,
		SafeDropThreshold: threshold,
	}

	return s.drawerRepo.OpenSession(ctx, session)
}

// RecordSafeDrop transfers surplus cash to safe vault.
func (s *CashDrawerService) RecordSafeDrop(ctx context.Context, staffUserID string, req dto.RecordSafeDropRequest) (*model.CashDrawerSession, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	session, err := s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
	if err != nil {
		return nil, err
	}

	if req.Amount > session.ExpectedCash {
		return nil, errors.New("cannot safe-drop more cash than currently present in drawer")
	}

	err = s.drawerRepo.ApplySafeDrop(ctx, session.ID, shop.ID, staffUserID, req.Amount, strings.TrimSpace(req.Notes))
	if err != nil {
		return nil, err
	}

	return s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
}

// RecordPettyExpense logs store expenses paid with drawer cash.
func (s *CashDrawerService) RecordPettyExpense(ctx context.Context, staffUserID string, req dto.RecordPettyExpenseRequest) (*model.CashDrawerSession, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	session, err := s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
	if err != nil {
		return nil, err
	}

	if req.Amount > session.ExpectedCash {
		return nil, errors.New("cannot pay expense: drawer has insufficient cash")
	}

	err = s.drawerRepo.ApplyPettyExpense(
		ctx, session.ID, shop.ID, staffUserID, req.Amount,
		strings.TrimSpace(req.Category), strings.TrimSpace(req.Reason),
	)
	if err != nil {
		return nil, err
	}

	return s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
}

// CloseSession closes the shift and calculates discrepancy amount.
func (s *CashDrawerService) CloseSession(ctx context.Context, staffUserID string, req dto.CloseDrawerSessionRequest) (*model.CashDrawerSession, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	session, err := s.drawerRepo.GetActiveSession(ctx, shop.ID, staffUserID)
	if err != nil {
		return nil, err
	}

	return s.drawerRepo.CloseSession(ctx, session.ID, shop.ID, staffUserID, req.ActualCash, strings.TrimSpace(req.ClosingNotes))
}

// ListHistory retrieves past drawer shifts.
func (s *CashDrawerService) ListHistory(ctx context.Context, staffUserID string) (*dto.DrawerSessionsHistoryResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, staffUserID)
	if err != nil {
		return nil, err
	}

	sessions, err := s.drawerRepo.ListHistory(ctx, shop.ID, 30)
	if err != nil {
		return nil, err
	}

	return &dto.DrawerSessionsHistoryResponse{
		Sessions: sessions,
		Count:    len(sessions),
	}, nil
}


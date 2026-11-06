// Package services handles business logic for margin shield and counter speed hotkeys.
package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// MarginService orchestrates live bargaining floor validation and POS fast-grid hotkeys.
type MarginService struct {
	shopRepo   *repository.ShopRepo
	hotkeyRepo *repository.HotkeyRepo
	logger     *zap.Logger
}

// NewMarginService initializes the margin and counter speed service.
func NewMarginService(
	shopRepo *repository.ShopRepo,
	hotkeyRepo *repository.HotkeyRepo,
	logger *zap.Logger,
) *MarginService {
	return &MarginService{
		shopRepo:   shopRepo,
		hotkeyRepo: hotkeyRepo,
		logger:     logger,
	}
}

// CheckMargin calculates real-time bargaining safety and returns color-coded stealth badges.
func (s *MarginService) CheckMargin(ctx context.Context, shopOwnerUserID string, req dto.MarginCheckRequest) (*dto.MarginCheckResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	bounds, err := s.hotkeyRepo.GetItemMarginBounds(ctx, shop.ID, req.ProductID, req.VariantID)
	if err != nil {
		return nil, err
	}

	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	costPrice := bounds.CostPrice
	floorPrice := bounds.FloorPrice
	normalPrice := bounds.Price
	minMarginPct := bounds.MinMarginPercent
	if minMarginPct <= 0 {
		minMarginPct = 10.00
	}

	// Calculate absolute minimum floor:
	// Maximum of explicit floorPrice and (costPrice + minimum shop margin)
	costWithMargin := costPrice * (1.0 + (minMarginPct / 100.0))
	effectiveFloor := math.Max(floorPrice, costWithMargin)

	// If no cost or floor is configured, default to normalPrice
	if effectiveFloor <= 0 {
		effectiveFloor = normalPrice
	}

	res := &dto.MarginCheckResponse{
		FloorPrice:    effectiveFloor,
		CostPrice:     costPrice,
		NormalPrice:   normalPrice,
		ProposedPrice: req.ProposedPrice,
	}

	// Check if proposed price causes a financial loss
	if req.ProposedPrice < effectiveFloor {
		lossPerUnit := effectiveFloor - req.ProposedPrice
		res.IsAllowed = false
		res.UnitProfit = req.ProposedPrice - costPrice
		res.TotalProfit = res.UnitProfit * float64(qty)
		if req.ProposedPrice > 0 {
			res.ProfitMarginPct = ((req.ProposedPrice - costPrice) / req.ProposedPrice) * 100.0
		}
		res.IndicatorColor = "RED"
		res.StealthBadge = "🔴 LOSS ALERT"
		res.WarningMessage = fmt.Sprintf(
			"Loss Prevention Alert: Minimum safe price is ₹%.2f (Wholesale Cost: ₹%.2f). Selling at ₹%.2f causes a shortfall of ₹%.2f per unit.",
			effectiveFloor, costPrice, req.ProposedPrice, lossPerUnit,
		)
		return res, nil
	}

	// Allowed pricing
	res.IsAllowed = true
	profit := req.ProposedPrice - costPrice
	res.UnitProfit = profit
	res.TotalProfit = profit * float64(qty)

	if req.ProposedPrice > 0 {
		res.ProfitMarginPct = math.Round((profit/req.ProposedPrice)*10000) / 100
	} else {
		res.ProfitMarginPct = 0
	}

	if res.ProfitMarginPct < 15.0 {
		res.IndicatorColor = "ORANGE"
		res.StealthBadge = "🟠 TIGHT MARGIN"
		res.WarningMessage = fmt.Sprintf(
			"Tight Margin Warning: Profit is ₹%.2f per unit (%.1f%%). Consider suggesting a matching accessory or cosmetic.",
			profit, res.ProfitMarginPct,
		)
	} else {
		res.IndicatorColor = "GREEN"
		res.StealthBadge = "🟢 SAFE BARGAIN"
		res.WarningMessage = fmt.Sprintf(
			"Safe Margin: Profit is ₹%.2f per unit (%.1f%%). Total profit for %d units is ₹%.2f.",
			profit, res.ProfitMarginPct, qty, res.TotalProfit,
		)
	}

	return res, nil
}

// ListHotkeys retrieves fast-grid tiles for cashier screen.
func (s *MarginService) ListHotkeys(ctx context.Context, shopOwnerUserID, timeSlot string) (*dto.ShopHotkeysListResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	hotkeys, err := s.hotkeyRepo.ListByShop(ctx, shop.ID, timeSlot)
	if err != nil {
		return nil, err
	}

	return &dto.ShopHotkeysListResponse{
		Hotkeys: hotkeys,
		Count:   len(hotkeys),
	}, nil
}

// CreateHotkey adds a new quick tile to the cashier grid.
func (s *MarginService) CreateHotkey(ctx context.Context, shopOwnerUserID string, req dto.CreateShopHotkeyRequest) (*model.ShopHotkey, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	hotkey := &model.ShopHotkey{
		ShopID:      shop.ID,
		ProductID:   req.ProductID,
		VariantID:   req.VariantID,
		DisplayName: strings.TrimSpace(req.DisplayName),
		ColorHex:    strings.TrimSpace(req.ColorHex),
		SortOrder:   req.SortOrder,
		TimeSlot:    strings.ToUpper(strings.TrimSpace(req.TimeSlot)),
		IsActive:    true,
	}

	return s.hotkeyRepo.Create(ctx, hotkey)
}

// DeleteHotkey removes a tile from the cashier grid.
func (s *MarginService) DeleteHotkey(ctx context.Context, shopOwnerUserID, id string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return err
	}

	return s.hotkeyRepo.Delete(ctx, shop.ID, id)
}

// AutoPopulateHotkeys scans recent POS bills and automatically populates the top 12 bestsellers.
func (s *MarginService) AutoPopulateHotkeys(ctx context.Context, shopOwnerUserID string) (*dto.ShopHotkeysListResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	hotkeys, err := s.hotkeyRepo.AutoPopulateFromTopSellers(ctx, shop.ID, 16)
	if err != nil {
		return nil, err
	}

	return &dto.ShopHotkeysListResponse{
		Hotkeys: hotkeys,
		Count:   len(hotkeys),
	}, nil
}


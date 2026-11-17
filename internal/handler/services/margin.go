// Package services handles business logic for margin shield and counter speed hotkeys.
package services

import (
	"context"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// MarginService orchestrates POS fast-grid hotkeys.
type MarginService struct {
	shopRepo   *repository.ShopRepo
	hotkeyRepo *repository.HotkeyRepo
	logger     *zap.Logger
}

// NewMarginService initializes the counter speed service.
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


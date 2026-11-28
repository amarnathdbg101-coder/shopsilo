// Package services handles business logic for margin shield and counter speed hotkeys.
package services

import (
	"context"
	"errors"
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

// validateCreateHotkeyInput validates product ID and display name for a cashier hotkey.
func validateCreateHotkeyInput(req *dto.CreateShopHotkeyRequest) error {
	req.ProductID = strings.TrimSpace(req.ProductID)
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		return errors.New("display_name is required")
	}
	return nil
}

// CreateHotkey adds a new quick tile to the cashier grid.
func (s *MarginService) CreateHotkey(ctx context.Context, shopOwnerUserID string, req dto.CreateShopHotkeyRequest) (*model.ShopHotkey, error) {
	if err := validateCreateHotkeyInput(&req); err != nil {
		return nil, err
	}
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	hotkey := &model.ShopHotkey{
		ShopID:      shop.ID,
		ProductID:   req.ProductID,
		VariantID:   req.VariantID,
		DisplayName: req.DisplayName,
		ColorHex:    strings.TrimSpace(req.ColorHex),
		SortOrder:   req.SortOrder,
		TimeSlot:    strings.ToUpper(strings.TrimSpace(req.TimeSlot)),
		IsActive:    true,
	}

	return s.hotkeyRepo.Create(ctx, hotkey)
}

// DeleteHotkey removes a tile from the cashier grid.
func (s *MarginService) DeleteHotkey(ctx context.Context, shopOwnerUserID, id string) error {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return errors.New("hotkey id is required")
	}
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return err
	}

	return s.hotkeyRepo.Delete(ctx, shop.ID, cleanID)
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


// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
)

type InventoryService struct {
	productRepo *repository.ProductRepo
	shopRepo    *repository.ShopRepo
}

func NewInventoryService(productRepo *repository.ProductRepo, shopRepo *repository.ShopRepo) *InventoryService {
	return &InventoryService{
		productRepo: productRepo,
		shopRepo:    shopRepo,
	}
}

// AdjustStock allows shop owner to record wholesale stock arrival (+qty) or audited adjustments.
func (s *InventoryService) AdjustStock(ctx context.Context, shopOwnerUserID string, input dto.AdjustStockRequest) (*model.Inventory, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	inv, err := s.productRepo.AdjustInventoryStock(ctx, shop.ID, input.ProductID, input.Adjustment, input.LowStockThreshold)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errors.New("product not found in your shop")
		}
		return nil, err
	}

	return inv, nil
}

// GetLowStockAlerts returns all products that are at or below the low stock threshold.
func (s *InventoryService) GetLowStockAlerts(ctx context.Context, shopOwnerUserID string) (*dto.LowStockResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetLowStockProducts(ctx, shop.ID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*dto.LowStockProduct{}
	}

	return &dto.LowStockResponse{
		TotalLowStockItems: len(items),
		Items:              items,
	}, nil
}

// GenerateReorderSheetPDF creates a printable wholesale re-order sheet in PDF.
func (s *InventoryService) GenerateReorderSheetPDF(ctx context.Context, shopOwnerUserID string) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetLowStockProducts(ctx, shop.ID)
	if err != nil {
		return nil, err
	}

	return utils.GenerateWholesaleReorderPDF(shop, items)
}

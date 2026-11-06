// Package services implements business logic for shop management and operations.
package services

import (
	"context"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// ReturnLookupService handles finding historical transactions for customers returning goods without paper receipt.
type ReturnLookupService struct {
	returnRepo *repository.ReturnLookupRepo
	shopRepo   *repository.ShopRepo
	logger     *zap.Logger
}

// NewReturnLookupService initializes ReturnLookupService.
func NewReturnLookupService(
	returnRepo *repository.ReturnLookupRepo,
	shopRepo *repository.ShopRepo,
	logger *zap.Logger,
) *ReturnLookupService {
	return &ReturnLookupService{
		returnRepo: returnRepo,
		shopRepo:   shopRepo,
		logger:     logger,
	}
}

// LookupReturn finds past sales matching customer phone, barcode, product ID, or bill number.
func (s *ReturnLookupService) LookupReturn(ctx context.Context, shopOwnerUserID string, filter dto.ReturnLookupFilter) (*dto.ReturnLookupResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	filter.ShopID = shop.ID
	items, err := s.returnRepo.FindPastPurchases(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.ReturnLookupResponse{
		Count:   len(items),
		Results: items,
	}, nil
}


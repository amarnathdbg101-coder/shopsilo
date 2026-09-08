// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"time"
)

type LoyaltyService struct {
	loyaltyRepo *repository.LoyaltyRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
}

func NewLoyaltyService(
	loyaltyRepo *repository.LoyaltyRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
) *LoyaltyService {
	return &LoyaltyService{
		loyaltyRepo: loyaltyRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
	}
}

// ProcessReturn handles in-store customer returns: restocks inventory, adjusts loyalty points, and logs return.
func (s *LoyaltyService) ProcessReturn(ctx context.Context, shopOwnerUserID string, input dto.CreateReturnRequest) (*model.ProductReturn, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	prod, err := s.productRepo.FindByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.New("product not found")
	}
	if prod.ShopID != shop.ID {
		return nil, errors.New("product does not belong to your shop")
	}

	ret := &model.ProductReturn{
		ShopID:        shop.ID,
		ProductID:     input.ProductID,
		ProductName:   prod.Name,
		ReservationID: input.ReservationID,
		Quantity:      input.Quantity,
		RefundAmount:  input.RefundAmount,
		Reason:        input.Reason,
	}

	tx, err := s.loyaltyRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	if err := s.loyaltyRepo.RecordReturnWithTx(ctx, tx, ret); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit product return")
	}

	return ret, nil
}

func (s *LoyaltyService) ListShopReturns(ctx context.Context, shopOwnerUserID string) ([]*model.ProductReturn, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.loyaltyRepo.ListShopReturns(ctx, shop.ID)
}

// CreateOffer allows a shopkeeper to create a new in-store offer/deal (can set min loyalty points for VIPs).
func (s *LoyaltyService) CreateOffer(ctx context.Context, shopOwnerUserID string, input dto.CreateOfferRequest) (*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	var expiresAt *time.Time
	if input.ExpiresInDays > 0 {
		exp := time.Now().AddDate(0, 0, input.ExpiresInDays)
		expiresAt = &exp
	}

	offer := &model.StoreOffer{
		ShopID:            shop.ID,
		Title:             input.Title,
		Description:       input.Description,
		DiscountText:      input.DiscountText,
		MinPointsRequired: input.MinPointsRequired,
		ExpiresAt:         expiresAt,
	}

	return s.loyaltyRepo.CreateOffer(ctx, offer)
}

// ListShopOffers lists active store offers and marks whether customer has unlocked VIP deals.
func (s *LoyaltyService) ListShopOffers(ctx context.Context, slug string, customerUserID string) ([]*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	userPoints := 0
	if customerUserID != "" {
		summary, err := s.loyaltyRepo.GetUserLoyalty(ctx, customerUserID)
		if err == nil && summary != nil {
			userPoints = summary.Points
		}
	}

	return s.loyaltyRepo.ListOffers(ctx, shop.ID, userPoints)
}

func (s *LoyaltyService) ListAllActiveOffers(ctx context.Context, category string) ([]*model.StoreOffer, error) {
	return s.loyaltyRepo.ListAllActiveOffers(ctx, category, 30)
}

func (s *LoyaltyService) GetUserLoyalty(ctx context.Context, userID string) (*model.LoyaltySummary, error) {
	return s.loyaltyRepo.GetUserLoyalty(ctx, userID)
}

// ScanProduct performs instant barcode/SKU/QR lookup for physical camera scanning.
func (s *LoyaltyService) ScanProduct(ctx context.Context, code string) (*dto.ScanProductResponse, error) {
	prod, err := s.loyaltyRepo.FindProductByBarcode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("no active product matches scanned code '%s'", code)
	}

	// 1 point reward per Rs 20
	pointsReward := int(prod.Price / 20)
	if pointsReward < 1 {
		pointsReward = 1
	}

	return &dto.ScanProductResponse{
		Product:      prod,
		PointsReward: pointsReward,
	}, nil
}

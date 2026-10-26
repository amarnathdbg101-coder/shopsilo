// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
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

	discType := strings.ToLower(strings.TrimSpace(input.DiscountType))
	if discType != "flat" && discType != "percentage" {
		discType = "percentage"
	}

	appDays := strings.ToLower(strings.TrimSpace(input.ApplicableDays))
	if appDays == "" {
		appDays = "all"
	}

	bannerTag := strings.TrimSpace(input.BannerTag)
	if bannerTag == "" {
		bannerTag = "🔥 Special Offer"
	}

	offer := &model.StoreOffer{
		ShopID:            shop.ID,
		Title:             input.Title,
		Description:       input.Description,
		DiscountText:      input.DiscountText,
		DiscountType:      discType,
		DiscountValue:     input.DiscountValue,
		MinOrderAmount:    input.MinOrderAmount,
		MaxDiscount:       input.MaxDiscount,
		ApplicableDays:    appDays,
		BannerTag:         bannerTag,
		PromoCode:         strings.ToUpper(strings.TrimSpace(input.PromoCode)),
		MinPointsRequired: input.MinPointsRequired,
		ExpiresAt:         expiresAt,
	}

	created, err := s.loyaltyRepo.CreateOffer(ctx, offer)
	if err != nil {
		return nil, err
	}

	// Record Audit Track for Creation
	_ = s.loyaltyRepo.RecordOfferAudit(ctx, &dto.OfferAuditRecord{
		OfferID:         created.ID,
		ShopID:          shop.ID,
		Action:          "CREATED",
		NewTitle:        created.Title,
		NewDiscountText: created.DiscountText,
		NewDescription:  created.Description,
		ChangedByUserID: shopOwnerUserID,
	})

	// Instant Real-Time WebSocket Push (<15ms)
	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(shop.ID, "OFFER_MUTATED", created)
	wsHub.BroadcastToStorefront(shop.Slug, "OFFER_MUTATED", created)

	return created, nil
}

func (s *LoyaltyService) UpdateOffer(ctx context.Context, shopOwnerUserID string, offerID string, input dto.UpdateOfferRequest) (*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	existing, err := s.loyaltyRepo.GetOfferByID(ctx, offerID)
	if err != nil {
		return nil, errors.New("offer not found")
	}

	if existing.ShopID != shop.ID {
		return nil, errors.New("unauthorized to update this offer")
	}

	var expiresAt *time.Time
	if input.ExpiresInDays > 0 {
		exp := time.Now().AddDate(0, 0, input.ExpiresInDays)
		expiresAt = &exp
	} else {
		expiresAt = existing.ExpiresAt
	}

	isActive := existing.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	discType := existing.DiscountType
	if input.DiscountType != nil && *input.DiscountType != "" {
		dt := strings.ToLower(strings.TrimSpace(*input.DiscountType))
		if dt == "flat" || dt == "percentage" {
			discType = dt
		}
	}

	discVal := existing.DiscountValue
	if input.DiscountValue != nil {
		discVal = *input.DiscountValue
	}

	minOrder := existing.MinOrderAmount
	if input.MinOrderAmount != nil {
		minOrder = *input.MinOrderAmount
	}

	maxDisc := existing.MaxDiscount
	if input.MaxDiscount != nil {
		maxDisc = *input.MaxDiscount
	}

	appDays := existing.ApplicableDays
	if input.ApplicableDays != nil && *input.ApplicableDays != "" {
		appDays = strings.ToLower(strings.TrimSpace(*input.ApplicableDays))
	}

	bannerTag := existing.BannerTag
	if input.BannerTag != nil && *input.BannerTag != "" {
		bannerTag = strings.TrimSpace(*input.BannerTag)
	}

	promoCode := existing.PromoCode
	if input.PromoCode != nil {
		promoCode = strings.ToUpper(strings.TrimSpace(*input.PromoCode))
	}

	toUpdate := &model.StoreOffer{
		ID:                offerID,
		ShopID:            shop.ID,
		Title:             input.Title,
		Description:       input.Description,
		DiscountText:      input.DiscountText,
		DiscountType:      discType,
		DiscountValue:     discVal,
		MinOrderAmount:    minOrder,
		MaxDiscount:       maxDisc,
		ApplicableDays:    appDays,
		BannerTag:         bannerTag,
		PromoCode:         promoCode,
		MinPointsRequired: input.MinPointsRequired,
		IsActive:          isActive,
		ExpiresAt:         expiresAt,
	}

	updated, err := s.loyaltyRepo.UpdateOffer(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	// Record Audit Track
	_ = s.loyaltyRepo.RecordOfferAudit(ctx, &dto.OfferAuditRecord{
		OfferID:              offerID,
		ShopID:               shop.ID,
		Action:               "UPDATED",
		PreviousTitle:        existing.Title,
		PreviousDiscountText: existing.DiscountText,
		PreviousDescription:  existing.Description,
		NewTitle:             updated.Title,
		NewDiscountText:      updated.DiscountText,
		NewDescription:       updated.Description,
		ChangedByUserID:      shopOwnerUserID,
	})

	// Instant Real-Time WebSocket Push (<15ms)
	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(shop.ID, "OFFER_MUTATED", updated)
	wsHub.BroadcastToStorefront(shop.Slug, "OFFER_MUTATED", updated)

	return updated, nil
}

// ToggleOffer toggles active/paused status of a shop offer in 0ms with instant sync
func (s *LoyaltyService) ToggleOffer(ctx context.Context, shopOwnerUserID string, offerID string) (*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	existing, err := s.loyaltyRepo.GetOfferByID(ctx, offerID)
	if err != nil {
		return nil, errors.New("offer not found")
	}

	if existing.ShopID != shop.ID {
		return nil, errors.New("unauthorized to toggle this offer")
	}

	updated, err := s.loyaltyRepo.ToggleOffer(ctx, shop.ID, offerID)
	if err != nil {
		return nil, err
	}

	// Record Audit Track
	action := "PAUSED"
	if updated.IsActive {
		action = "ACTIVATED"
	}
	_ = s.loyaltyRepo.RecordOfferAudit(ctx, &dto.OfferAuditRecord{
		OfferID:              offerID,
		ShopID:               shop.ID,
		Action:               action,
		PreviousTitle:        existing.Title,
		PreviousDiscountText: existing.DiscountText,
		NewTitle:             updated.Title,
		NewDiscountText:      updated.DiscountText,
		ChangedByUserID:      shopOwnerUserID,
	})

	// Instant Real-Time WebSocket Push (<15ms)
	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(shop.ID, "OFFER_MUTATED", updated)
	wsHub.BroadcastToStorefront(shop.Slug, "OFFER_MUTATED", updated)

	return updated, nil
}

func (s *LoyaltyService) DeleteOffer(ctx context.Context, shopOwnerUserID string, offerID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	existing, err := s.loyaltyRepo.GetOfferByID(ctx, offerID)
	if err != nil {
		return errors.New("offer not found")
	}

	if existing.ShopID != shop.ID {
		return errors.New("unauthorized to delete this offer")
	}

	err = s.loyaltyRepo.DeleteOffer(ctx, shop.ID, offerID)
	if err != nil {
		return err
	}

	// Record Audit Track
	_ = s.loyaltyRepo.RecordOfferAudit(ctx, &dto.OfferAuditRecord{
		OfferID:              offerID,
		ShopID:               shop.ID,
		Action:               "DELETED",
		PreviousTitle:        existing.Title,
		PreviousDiscountText: existing.DiscountText,
		PreviousDescription:  existing.Description,
		ChangedByUserID:      shopOwnerUserID,
	})

	// Instant Real-Time WebSocket Push (<15ms)
	wsHub := GetWebSocketHub(nil)
	deletePayload := map[string]interface{}{
		"id":      offerID,
		"deleted": true,
	}
	wsHub.BroadcastToShop(shop.ID, "OFFER_MUTATED", deletePayload)
	wsHub.BroadcastToStorefront(shop.Slug, "OFFER_MUTATED", deletePayload)

	return nil
}

func (s *LoyaltyService) GetOfferHistory(ctx context.Context, shopOwnerUserID string) ([]*dto.OfferAuditRecord, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.loyaltyRepo.GetOfferAuditHistory(ctx, shop.ID)
}

// ListMyShopOffers returns all store offers for the shopkeeper dashboard (both active and paused)
func (s *LoyaltyService) ListMyShopOffers(ctx context.Context, shopOwnerUserID string) ([]*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.loyaltyRepo.ListShopOffers(ctx, shop.ID)
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

// ListStorefrontOffers returns active offers for customer storefront with optional day filtering
func (s *LoyaltyService) ListStorefrontOffers(ctx context.Context, slug string, day string) ([]*model.StoreOffer, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.loyaltyRepo.ListStorefrontOffers(ctx, shop.ID, day)
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

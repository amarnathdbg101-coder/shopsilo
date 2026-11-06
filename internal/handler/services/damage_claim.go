// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// DamageClaimService handles tracking and resolution of expired, leaking, or damaged stock claims.
type DamageClaimService struct {
	claimRepo *repository.DamageClaimRepo
	shopRepo  *repository.ShopRepo
	logger    *zap.Logger
}

// NewDamageClaimService initializes DamageClaimService.
func NewDamageClaimService(
	claimRepo *repository.DamageClaimRepo,
	shopRepo *repository.ShopRepo,
	logger *zap.Logger,
) *DamageClaimService {
	return &DamageClaimService{
		claimRepo: claimRepo,
		shopRepo:  shopRepo,
		logger:    logger,
	}
}

// CreateClaim logs a damaged or expired product into the company claim vault.
func (s *DamageClaimService) CreateClaim(ctx context.Context, shopOwnerUserID string, req dto.CreateDamageClaimRequest) (*model.DamageClaim, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	distName := strings.TrimSpace(req.DistributorName)
	if distName == "" {
		return nil, errors.New("distributor name is required to file a company claim")
	}

	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ExpiryDate)
		if err == nil {
			expiryDate = &parsed
		}
	}

	claim := &model.DamageClaim{
		ShopID:           shop.ID,
		ProductID:        req.ProductID,
		VariantID:        req.VariantID,
		ProductName:      strings.TrimSpace(req.ProductName),
		Barcode:          strings.TrimSpace(req.Barcode),
		BatchNumber:      strings.TrimSpace(req.BatchNumber),
		ExpiryDate:       expiryDate,
		Quantity:         req.Quantity,
		UnitCost:         req.UnitCost,
		ClaimType:        req.ClaimType,
		DistributorName:  distName,
		DistributorPhone: strings.TrimSpace(req.DistributorPhone),
		SalesRepName:     strings.TrimSpace(req.SalesRepName),
		PhotoURL:         strings.TrimSpace(req.PhotoURL),
		Status:           model.ClaimStatusPendingClaim,
	}

	return s.claimRepo.Create(ctx, claim)
}

// UpdateClaimStatus handles company Sales Representative (SR) settlement (replacement or credit note).
func (s *DamageClaimService) UpdateClaimStatus(ctx context.Context, shopOwnerUserID, claimID string, req dto.UpdateClaimStatusRequest) (*model.DamageClaim, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	existing, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		return nil, err
	}
	if existing.ShopID != shop.ID {
		return nil, errors.New("unauthorized to update this damage claim")
	}

	return s.claimRepo.UpdateStatus(ctx, claimID, req.Status, strings.TrimSpace(req.ResolutionNotes))
}

// ListClaims returns all damage and expiry records with optional filters.
func (s *DamageClaimService) ListClaims(ctx context.Context, shopOwnerUserID, distributorName, status string, limit int) ([]*model.DamageClaim, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	return s.claimRepo.ListByShop(ctx, shop.ID, distributorName, status, limit)
}

// GetDistributorStatement generates an aggregated claim report when the company SR visits.
func (s *DamageClaimService) GetDistributorStatement(ctx context.Context, shopOwnerUserID, distributorName string) (*dto.DamageClaimSummaryResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	claims, err := s.claimRepo.ListByShop(ctx, shop.ID, distributorName, "", 100)
	if err != nil {
		return nil, err
	}

	resp := &dto.DamageClaimSummaryResponse{
		ShopID:          shop.ID,
		DistributorName: distributorName,
		Claims:          claims,
	}

	for _, c := range claims {
		resp.TotalClaimsCount++
		resp.TotalLossAmount += c.TotalLoss

		if c.Status == model.ClaimStatusPendingClaim || c.Status == model.ClaimStatusSubmittedToSR {
			resp.PendingClaimsCount++
			resp.PendingLossAmount += c.TotalLoss
		} else if c.Status == model.ClaimStatusReplaced || c.Status == model.ClaimStatusCreditNoteIssued {
			resp.RecoveredLossAmount += c.TotalLoss
		}
	}

	return resp, nil
}


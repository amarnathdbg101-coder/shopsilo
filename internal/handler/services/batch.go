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

// BatchService manages product batches, expiry radar, and automated markdown protection.
type BatchService struct {
	batchRepo   *repository.BatchRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
	logger      *zap.Logger
}

// NewBatchService initializes BatchService.
func NewBatchService(
	batchRepo *repository.BatchRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
	logger *zap.Logger,
) *BatchService {
	return &BatchService{
		batchRepo:   batchRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
		logger:      logger,
	}
}

// CreateBatch logs a newly delivered batch with manufacturing and expiry dates.
func (s *BatchService) CreateBatch(ctx context.Context, shopOwnerUserID string, req dto.CreateBatchRequest) (*model.ProductBatch, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil || product == nil || product.ShopID != shop.ID {
		return nil, errors.New("product does not exist in your shop catalog")
	}

	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		return nil, errors.New("invalid expiry_date format, expected YYYY-MM-DD")
	}

	var mfgDate *time.Time
	if req.MfgDate != nil && *req.MfgDate != "" {
		if parsed, err := time.Parse("2006-01-02", *req.MfgDate); err == nil {
			mfgDate = &parsed
		}
	}

	batch := &model.ProductBatch{
		ShopID:               shop.ID,
		ProductID:            product.ID,
		VariantID:            req.VariantID,
		BatchNumber:          strings.TrimSpace(req.BatchNumber),
		MfgDate:              mfgDate,
		ExpiryDate:           expiryDate,
		Quantity:             req.Quantity,
		CostPrice:            req.CostPrice,
		OriginalSellingPrice: req.SellingPrice,
		CurrentSellingPrice:  req.SellingPrice,
		MarkdownPercent:      req.MarkdownPercent,
		IsMarkdownApplied:    false,
		Status:               model.BatchStatusActive,
	}

	return s.batchRepo.Create(ctx, batch)
}

// ListNearExpiryBatches lists batches close to expiration.
func (s *BatchService) ListNearExpiryBatches(ctx context.Context, shopOwnerUserID string, daysWindow int) (*dto.BatchListResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	batches, err := s.batchRepo.ListNearExpiry(ctx, shop.ID, daysWindow)
	if err != nil {
		return nil, err
	}

	return &dto.BatchListResponse{
		Count:   len(batches),
		Batches: batches,
	}, nil
}

// RunMarkdownScan triggers the auto-markdown engine across all inventory batches.
func (s *BatchService) RunMarkdownScan(ctx context.Context, shopOwnerUserID string, daysThreshold int, defaultMarkdownPercent float64) (*dto.RunMarkdownScanResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	scanned, markdownsApplied, expiredCount, capitalProtected, updated, err := s.batchRepo.RunAutoMarkdownEngine(
		ctx,
		shop.ID,
		daysThreshold,
		defaultMarkdownPercent,
	)
	if err != nil {
		return nil, err
	}

	return &dto.RunMarkdownScanResponse{
		BatchesScanned:         scanned,
		MarkdownsApplied:       markdownsApplied,
		ExpiredBatchesCount:    expiredCount,
		CapitalProtectedAmount: capitalProtected,
		Batches:                updated,
	}, nil
}


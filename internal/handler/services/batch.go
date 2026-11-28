// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"errors"
	"math"
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

// validateCreateBatchInput enforces sane quantities, prices, markdown bounds, and date ordering.
func validateCreateBatchInput(req dto.CreateBatchRequest) (time.Time, *time.Time, error) {
	if strings.TrimSpace(req.ProductID) == "" {
		return time.Time{}, nil, errors.New("product_id is required")
	}
	if req.Quantity <= 0 {
		return time.Time{}, nil, errors.New("batch quantity must be greater than zero")
	}
	if math.IsNaN(req.CostPrice) || math.IsInf(req.CostPrice, 0) || req.CostPrice < 0 {
		return time.Time{}, nil, errors.New("cost_price cannot be negative or invalid")
	}
	if math.IsNaN(req.SellingPrice) || math.IsInf(req.SellingPrice, 0) || req.SellingPrice <= 0 {
		return time.Time{}, nil, errors.New("selling_price must be greater than zero")
	}
	if math.IsNaN(req.MarkdownPercent) || math.IsInf(req.MarkdownPercent, 0) || req.MarkdownPercent < 0 || req.MarkdownPercent > 95 {
		return time.Time{}, nil, errors.New("markdown_percent must be between 0 and 95")
	}

	expiryDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.ExpiryDate))
	if err != nil {
		return time.Time{}, nil, errors.New("invalid expiry_date format, expected YYYY-MM-DD")
	}

	var mfgDate *time.Time
	if req.MfgDate != nil && strings.TrimSpace(*req.MfgDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.MfgDate))
		if err != nil {
			return time.Time{}, nil, errors.New("invalid mfg_date format, expected YYYY-MM-DD")
		}
		if parsed.After(expiryDate) {
			return time.Time{}, nil, errors.New("mfg_date cannot be after expiry_date")
		}
		mfgDate = &parsed
	}
	return expiryDate, mfgDate, nil
}

// CreateBatch logs a newly delivered batch with manufacturing and expiry dates.
func (s *BatchService) CreateBatch(ctx context.Context, shopOwnerUserID string, req dto.CreateBatchRequest) (*model.ProductBatch, error) {
	expiryDate, mfgDate, err := validateCreateBatchInput(req)
	if err != nil {
		return nil, err
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil || product == nil || product.ShopID != shop.ID {
		return nil, errors.New("product does not exist in your shop catalog")
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

// GetExpirySummary fetches aggregate financial risk metrics across the merchant's batch inventory.
func (s *BatchService) GetExpirySummary(ctx context.Context, shopOwnerUserID string) (*dto.ExpiryRadarSummaryResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	return s.batchRepo.GetExpirySummary(ctx, shop.ID)
}

// validateBatchMarkdownInput prevents negative/over-100% discounts and invalid custom prices.
func validateBatchMarkdownInput(req dto.ApplyBatchMarkdownRequest) error {
	if req.CustomPrice != nil {
		if math.IsNaN(*req.CustomPrice) || math.IsInf(*req.CustomPrice, 0) || *req.CustomPrice <= 0 {
			return errors.New("custom_price must be greater than zero")
		}
		return nil
	}
	if math.IsNaN(req.MarkdownPercent) || math.IsInf(req.MarkdownPercent, 0) || req.MarkdownPercent <= 0 || req.MarkdownPercent > 95 {
		return errors.New("markdown_percent must be between 0 and 95")
	}
	return nil
}

// ApplyBatchMarkdown applies a targeted clearance markdown to a specific batch lot.
func (s *BatchService) ApplyBatchMarkdown(ctx context.Context, shopOwnerUserID string, batchID string, req dto.ApplyBatchMarkdownRequest) (*model.ProductBatch, error) {
	if err := validateBatchMarkdownInput(req); err != nil {
		return nil, err
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	updated, err := s.batchRepo.ApplyBatchMarkdown(ctx, shop.ID, batchID, req.MarkdownPercent, req.CustomPrice)
	if err != nil {
		return nil, err
	}
	InvalidateAnalyticsCache(shop.ID)
	return updated, nil
}

// WriteOffBatch records physical disposal or vendor return for an expired batch.
func (s *BatchService) WriteOffBatch(ctx context.Context, shopOwnerUserID string, batchID string, req dto.BatchWriteOffRequest) (*model.ProductBatch, error) {
	if req.Quantity <= 0 {
		return nil, errors.New("write-off quantity must be greater than zero")
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	updated, err := s.batchRepo.WriteOff(ctx, shop.ID, batchID, req.Quantity, req.Reason)
	if err != nil {
		return nil, err
	}
	InvalidateAnalyticsCache(shop.ID)
	return updated, nil
}


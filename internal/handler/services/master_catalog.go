package services

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

// MasterCatalogService provides business logic for catalog lookups and admin bulk seeding.
type MasterCatalogService struct {
	repo     *repository.MasterCatalogRepo
	shopRepo *repository.ShopRepo
	logger   *zap.Logger
}

func NewMasterCatalogService(
	repo *repository.MasterCatalogRepo,
	shopRepo *repository.ShopRepo,
	logger *zap.Logger,
) *MasterCatalogService {
	return &MasterCatalogService{
		repo:     repo,
		shopRepo: shopRepo,
		logger:   logger,
	}
}

// LookupByBarcode searches the merchant's store first, then falls back to the universal Master Catalog.
func (s *MasterCatalogService) LookupByBarcode(ctx context.Context, userID, barcode string) (*dto.CatalogLookupResponse, error) {
	cleanBarcode := strings.TrimSpace(barcode)
	if cleanBarcode == "" {
		return nil, errors.New("barcode cannot be empty")
	}

	// 1. Resolve merchant's shop ID
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("merchant shop not found")
	}

	resp := &dto.CatalogLookupResponse{
		Barcode: cleanBarcode,
	}

	// 2. Check if product already exists in merchant's own active inventory (Prevents duplicate product rows!)
	shopProduct, err := s.repo.FindShopProductByBarcode(ctx, shop.ID, cleanBarcode)
	if err != nil {
		s.logger.Warn("error checking shop product by barcode", zap.Error(err), zap.String("barcode", cleanBarcode))
	}
	if shopProduct != nil {
		resp.ExistsInShop = true
		resp.ShopProduct = shopProduct
		return resp, nil
	}

	// 3. Search Universal Master Catalog
	masterProduct, err := s.repo.FindByBarcode(ctx, cleanBarcode)
	if err != nil {
		return nil, err
	}
	if masterProduct != nil {
		resp.MasterFound = true
		resp.MasterProduct = masterProduct

		// Calculate suggested profit margin based on MRP (defaulting to 15% standard retail markup if cost is unknown)
		if masterProduct.MRP > 0 {
			resp.SuggestedMarginPct = 15.0
		}
		return resp, nil
	}

	// 4. Not found in shop and not found in Master Catalog
	resp.MasterFound = false
	return resp, nil
}

// BulkSeed handles high-throughput ingestion of curated products by administrators.
func (s *MasterCatalogService) BulkSeed(ctx context.Context, req *dto.MasterCatalogBulkSeedRequest) (*dto.MasterCatalogBulkSeedResponse, error) {
	if req == nil || len(req.Products) == 0 {
		return nil, errors.New("no products provided for seeding")
	}

	items := make([]model.MasterCatalogProduct, 0, len(req.Products))
	var errorList []string

	for idx, p := range req.Products {
		cleanBarcode := strings.TrimSpace(p.Barcode)
		cleanName := strings.TrimSpace(p.Name)
		if cleanBarcode == "" || cleanName == "" {
			errorList = append(errorList, "row missing barcode or name")
			continue
		}

		sellingPrice := p.SellingPrice
		if sellingPrice <= 0 && p.MRP > 0 {
			sellingPrice = p.MRP
		}

		items = append(items, model.MasterCatalogProduct{
			Barcode:      cleanBarcode,
			Name:         cleanName,
			Brand:        strings.TrimSpace(p.Brand),
			Category:     strings.TrimSpace(p.Category),
			Unit:         strings.TrimSpace(p.Unit),
			MRP:          p.MRP,
			SellingPrice: sellingPrice,
			ImageURL:     strings.TrimSpace(p.ImageURL),
			HSNCode:      strings.TrimSpace(p.HSNCode),
			GSTRate:      p.GSTRate,
			Description:  strings.TrimSpace(p.Description),
			Source:       "admin_seed",
			TrustScore:   100,
			IsVerified:   true,
		})

		_ = idx
	}

	processed, updated, err := s.repo.BulkUpsert(ctx, items)
	if err != nil {
		return nil, err
	}

	return &dto.MasterCatalogBulkSeedResponse{
		TotalProcessed: len(items),
		Inserted:       processed,
		Updated:        updated,
		Errors:         errorList,
	}, nil
}

// AdminHardDeleteProduct permanently deletes a product (Admin Only).
func (s *MasterCatalogService) AdminHardDeleteProduct(ctx context.Context, productID string) error {
	cleanID := strings.TrimSpace(productID)
	if cleanID == "" {
		return errors.New("product id is required")
	}
	return s.repo.HardDeleteProduct(ctx, cleanID)
}

// ListMasterProducts queries master products for admin catalog management.
func (s *MasterCatalogService) ListMasterProducts(ctx context.Context, search, brand, category string, isVerified *bool, limit, offset int) ([]*model.MasterCatalogProduct, int64, error) {
	return s.repo.ListMasterProducts(ctx, search, brand, category, isVerified, limit, offset)
}

// CreateMasterProduct adds or upserts a single master catalog product.
func (s *MasterCatalogService) CreateMasterProduct(ctx context.Context, p *model.MasterCatalogProduct) (*model.MasterCatalogProduct, error) {
	if strings.TrimSpace(p.Barcode) == "" {
		return nil, errors.New("barcode is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("product name is required")
	}
	return s.repo.CreateMasterProduct(ctx, p)
}

// UpdateMasterProduct modifies an existing master product.
func (s *MasterCatalogService) UpdateMasterProduct(ctx context.Context, id string, p *model.MasterCatalogProduct) (*model.MasterCatalogProduct, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, errors.New("product id is required")
	}
	return s.repo.UpdateMasterProduct(ctx, cleanID, p)
}

// DeleteMasterProduct soft-deletes a product from the master catalog.
func (s *MasterCatalogService) DeleteMasterProduct(ctx context.Context, id string) error {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return errors.New("product id is required")
	}
	return s.repo.DeleteMasterProduct(ctx, cleanID)
}

// GetMasterStats returns metrics for master catalog and soft-deleted items.
func (s *MasterCatalogService) GetMasterStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetMasterCatalogStats(ctx)
}

// ListSoftDeletedProducts retrieves soft-deleted products across all merchant shops.
func (s *MasterCatalogService) ListSoftDeletedProducts(ctx context.Context, search string, limit, offset int) ([]map[string]interface{}, int64, error) {
	return s.repo.ListSoftDeletedProducts(ctx, search, limit, offset)
}

// RestoreProduct restores a soft-deleted product back to active store catalog.
func (s *MasterCatalogService) RestoreProduct(ctx context.Context, productID string) error {
	cleanID := strings.TrimSpace(productID)
	if cleanID == "" {
		return errors.New("product id is required")
	}
	return s.repo.RestoreProduct(ctx, cleanID)
}

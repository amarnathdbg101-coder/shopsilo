package services

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/zap"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
)

// MasterCatalogService provides business logic for catalog lookups and admin bulk seeding.
type MasterCatalogService struct {
	repo          *repository.MasterCatalogRepo
	shopRepo      *repository.ShopRepo
	logger        *zap.Logger
	userShopCache sync.Map // map[userID string]shopID string for 0ms repeat resolutions
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

// LookupByBarcode searches the merchant's store first, then falls back to the universal Master Catalog,
// and finally checks any other shop in the database for crowdsourced learning.
func (s *MasterCatalogService) LookupByBarcode(ctx context.Context, userID, barcode string) (*dto.CatalogLookupResponse, error) {
	cleanBarcode := utils.CleanBarcode(barcode)
	if cleanBarcode == "" {
		return nil, errors.New("barcode cannot be empty")
	}

	isValidChecksum := utils.ValidateGS1Checksum(cleanBarcode)

	resp := &dto.CatalogLookupResponse{
		Barcode:          cleanBarcode,
		CanonicalBarcode: cleanBarcode,
		IsValidChecksum:  isValidChecksum,
	}

	// 1. Resolve merchant's shop ID directly
	var merchantShopID string
	cleanUID := strings.TrimSpace(userID)
	if cleanUID != "" {
		if cached, ok := s.userShopCache.Load(cleanUID); ok && cached.(string) != "" {
			merchantShopID = cached.(string)
		} else {
			shop, err := s.shopRepo.FindByUserID(ctx, cleanUID)
			if err == nil && shop != nil {
				merchantShopID = shop.ID
				s.userShopCache.Store(cleanUID, merchantShopID)
			}
		}
	}

	// 2. Check if product already exists in merchant's own active inventory
	if merchantShopID != "" {
		shopProduct, err := s.repo.FindShopProductByBarcode(ctx, merchantShopID, cleanBarcode)
		if err != nil {
			s.logger.Warn("error checking shop product by barcode", zap.Error(err), zap.String("barcode", cleanBarcode))
		}
		if shopProduct != nil {
			resp.ExistsInShop = true
			resp.ShopProduct = shopProduct

			// Also populate MasterProduct from the shop's live product so the Add Product modal
			// can auto-fill all fields (name, price, images, brand, etc.) without leaving the form blank!
			var imgURL string
			if len(shopProduct.Images) > 0 {
				imgURL = shopProduct.Images[0]
			}
			brand := ""
			unit := "piece"
			hsnCode := ""
			gstRate := 5.0
			if shopProduct.Attributes != nil {
				if b, ok := shopProduct.Attributes["brand"].(string); ok && b != "" {
					brand = b
				}
				if u, ok := shopProduct.Attributes["unit"].(string); ok && u != "" {
					unit = u
				}
				if h, ok := shopProduct.Attributes["hsn_code"].(string); ok && h != "" {
					hsnCode = h
				}
				if g, ok := shopProduct.Attributes["gst_rate"].(float64); ok && g > 0 {
					gstRate = g
				}
			}
			mrp := shopProduct.ComparePrice
			if mrp <= 0 {
				mrp = shopProduct.MRP
			}
			if mrp <= 0 {
				mrp = shopProduct.Price
			}

			// If shop product lacks image, brand, or HSN, enrich from Master Catalog
			if masterItem, _ := s.repo.FindByBarcode(ctx, cleanBarcode); masterItem != nil {
				if imgURL == "" && masterItem.ImageURL != "" {
					imgURL = masterItem.ImageURL
				}
				if (brand == "" || brand == "General") && masterItem.Brand != "" {
					brand = masterItem.Brand
				}
				if hsnCode == "" && masterItem.HSNCode != "" {
					hsnCode = masterItem.HSNCode
				}
				if (mrp <= 0 || mrp == shopProduct.Price) && masterItem.MRP > 0 {
					mrp = masterItem.MRP
				}
			}

			resp.MasterFound = true
			resp.MasterProduct = &model.MasterCatalogProduct{
				ID:           shopProduct.ID,
				Barcode:      cleanBarcode,
				Name:         shopProduct.Name,
				Brand:        brand,
				Category:     shopProduct.CategoryName,
				Unit:         unit,
				MRP:          mrp,
				SellingPrice: shopProduct.Price,
				Description:  shopProduct.Description,
				ImageURL:     imgURL,
				HSNCode:      hsnCode,
				GSTRate:      gstRate,
				Source:       "shop_inventory",
				TrustScore:   100,
				IsVerified:   true,
			}
			return resp, nil
		}
	}

	// 3. Search Universal Master Catalog
	masterProduct, err := s.repo.FindByBarcode(ctx, cleanBarcode)
	if err != nil {
		return nil, err
	}
	if masterProduct != nil {
		// If master product is missing an image, fallback to check any crowdsourced shop product
		if strings.TrimSpace(masterProduct.ImageURL) == "" {
			if crowdsourced, _ := s.repo.FindAnyShopProductByBarcode(ctx, cleanBarcode); crowdsourced != nil && strings.TrimSpace(crowdsourced.ImageURL) != "" {
				masterProduct.ImageURL = crowdsourced.ImageURL
			}
		}

		resp.MasterFound = true
		resp.MasterProduct = masterProduct

		// Calculate suggested profit margin based on MRP (defaulting to 15% standard retail markup if cost is unknown)
		if masterProduct.MRP > 0 {
			resp.SuggestedMarginPct = 15.0
		}
		return resp, nil
	}

	// 4. Crowdsourced Network Fallback: Check if ANY other shop in ShopSilo has this product SKU/barcode
	crowdsourcedProduct, err := s.repo.FindAnyShopProductByBarcode(ctx, cleanBarcode)
	if err != nil {
		s.logger.Warn("error searching crowdsourced shop products", zap.Error(err), zap.String("barcode", cleanBarcode))
	}
	if crowdsourcedProduct != nil {
		resp.MasterFound = true
		resp.MasterProduct = crowdsourcedProduct

		if crowdsourcedProduct.MRP > 0 {
			resp.SuggestedMarginPct = 15.0
		}
		return resp, nil
	}

	// 5. Not found in shop and not found in Master Catalog
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

// BatchDeleteMasterProducts soft-deletes selected master catalog products.
func (s *MasterCatalogService) BatchDeleteMasterProducts(ctx context.Context, ids []string) (int64, error) {
	return s.repo.BatchDeleteMasterProducts(ctx, ids)
}

// DeleteAllMasterProducts soft-deletes all master catalog products.
func (s *MasterCatalogService) DeleteAllMasterProducts(ctx context.Context) (int64, error) {
	return s.repo.DeleteAllMasterProducts(ctx)
}

// BatchHardDeleteProducts permanently deletes selected soft-deleted products.
func (s *MasterCatalogService) BatchHardDeleteProducts(ctx context.Context, ids []string) (int64, error) {
	return s.repo.BatchHardDeleteProducts(ctx, ids)
}

// HardDeleteAllSoftDeletedProducts permanently deletes all soft-deleted products.
func (s *MasterCatalogService) HardDeleteAllSoftDeletedProducts(ctx context.Context) (int64, error) {
	return s.repo.HardDeleteAllSoftDeletedProducts(ctx)
}

// ListAllStoreProducts lists all store products across shops with filters (Admin Only).
func (s *MasterCatalogService) ListAllStoreProducts(ctx context.Context, search, shopID, status string, limit, offset int) ([]map[string]interface{}, int64, error) {
	return s.repo.ListAllStoreProducts(ctx, search, shopID, status, limit, offset)
}

// AdminUpdateStoreProduct updates any store product's details (Admin Only).
func (s *MasterCatalogService) AdminUpdateStoreProduct(ctx context.Context, productID, name, category string, price, costPrice float64, isActive bool) error {
	cleanID := strings.TrimSpace(productID)
	if cleanID == "" {
		return errors.New("product id is required")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("product name is required")
	}
	if price < 0 || costPrice < 0 {
		return errors.New("price and cost_price must be non-negative")
	}
	return s.repo.AdminUpdateStoreProduct(ctx, cleanID, strings.TrimSpace(name), strings.TrimSpace(category), price, costPrice, isActive)
}

// AdminToggleStoreProductStatus toggles active/inactive status of any store product (Admin Only).
func (s *MasterCatalogService) AdminToggleStoreProductStatus(ctx context.Context, productID string, isActive bool) error {
	cleanID := strings.TrimSpace(productID)
	if cleanID == "" {
		return errors.New("product id is required")
	}
	return s.repo.AdminToggleStoreProductStatus(ctx, cleanID, isActive)
}

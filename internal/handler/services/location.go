// Package services handles business logic and domain coordination.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

var (
	ErrLocationNotFound = errors.New("location not found")
	ErrInvalidMatrix    = errors.New("sizes or colors/shades must be specified to generate variant matrix")
)

// LocationService orchestrates multi-store godown logistics, apparel/beauty variants, and stock transfers.
type LocationService struct {
	locationRepo *repository.LocationRepo
	shopRepo     *repository.ShopRepo
	productRepo  *repository.ProductRepo
}

// NewLocationService initializes the location service.
func NewLocationService(
	locationRepo *repository.LocationRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
) *LocationService {
	return &LocationService{
		locationRepo: locationRepo,
		shopRepo:     shopRepo,
		productRepo:  productRepo,
	}
}

// ListLocations retrieves all godowns and showrooms of the merchant.
func (s *LocationService) ListLocations(ctx context.Context, shopOwnerUserID string) ([]*model.Location, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.ListLocationsByShop(ctx, shop.ID)
}

// CreateLocation creates a new showroom, godown, or warehouse for the shop.
func (s *LocationService) CreateLocation(ctx context.Context, shopOwnerUserID string, input dto.CreateLocationRequest) (*model.Location, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.CreateLocation(ctx, shop.ID, input)
}

// UpdateLocation modifies details of an existing location.
func (s *LocationService) UpdateLocation(ctx context.Context, shopOwnerUserID, locationID string, input dto.UpdateLocationRequest) (*model.Location, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.UpdateLocation(ctx, shop.ID, locationID, input)
}

// GetLocationInventory fetches real-time stock levels at a specific location.
func (s *LocationService) GetLocationInventory(ctx context.Context, shopOwnerUserID, locationID string) ([]*model.LocationInventory, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.GetLocationInventory(ctx, shop.ID, locationID)
}

// AdjustStock applies a manual adjustment or wholesale arrival at a specific location.
func (s *LocationService) AdjustStock(ctx context.Context, shopOwnerUserID string, input dto.AdjustLocationStockRequest) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return ErrShopNotFound
	}
	err = s.locationRepo.AdjustLocationStock(ctx, shop.ID, input.LocationID, input.ProductID, input.VariantID, input.Adjustment)
	if err != nil {
		return err
	}

	// Broadcast stock change event via WebSocket
	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "STOCK_CHANGED", map[string]interface{}{
		"location_id": input.LocationID,
		"product_id":  input.ProductID,
		"variant_id":  input.VariantID,
		"adjustment":  input.Adjustment,
	})
	return nil
}

// ListVariants retrieves all variants of a product.
func (s *LocationService) ListVariants(ctx context.Context, shopOwnerUserID, productID, locationID string) ([]*model.ProductVariant, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.ListVariantsByProduct(ctx, shop.ID, productID, locationID)
}

// ListVariantsPublic retrieves all active variants of a product for public customers.
func (s *LocationService) ListVariantsPublic(ctx context.Context, productID string) ([]*model.ProductVariant, error) {
	return s.locationRepo.ListVariantsPublic(ctx, productID)
}

// CreateVariant creates an individual child SKU for apparel or beauty.
func (s *LocationService) CreateVariant(ctx context.Context, shopOwnerUserID, productID string, input dto.CreateVariantRequest) (*model.ProductVariant, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	v, err := s.locationRepo.CreateVariant(ctx, shop.ID, productID, input)
	if err != nil {
		return nil, err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "VARIANT_CREATED", map[string]interface{}{
		"product_id": productID,
		"variant_id": v.ID,
		"sku":        v.SKU,
	})
	return v, nil
}

// GenerateVariantMatrix generates the full Cartesian product matrix (Size x Color or Shades) in 1 tap.
func (s *LocationService) GenerateVariantMatrix(ctx context.Context, shopOwnerUserID, productID string, input dto.GenerateVariantMatrixRequest) ([]*model.ProductVariant, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	var variantsToCreate []dto.CreateVariantRequest
	baseSKU := prod.SKU
	if baseSKU == "" {
		baseSKU = strings.ToUpper(strings.ReplaceAll(prod.Slug, "-", ""))
	}

	// 1. Apparel Matrix: Sizes x Colors
	if len(input.Sizes) > 0 && len(input.Colors) > 0 {
		for _, color := range input.Colors {
			for _, size := range input.Sizes {
				cleanColor := strings.ToUpper(strings.ReplaceAll(color.Name, " ", ""))
				cleanSize := strings.ToUpper(strings.ReplaceAll(size, " ", ""))
				sku := fmt.Sprintf("%s-%s-%s", baseSKU, cleanColor, cleanSize)
				title := fmt.Sprintf("%s / %s", color.Name, size)

				variantsToCreate = append(variantsToCreate, dto.CreateVariantRequest{
					SKU:          sku,
					Title:        title,
					Size:         size,
					Color:        color.Name,
					ColorHex:     color.Hex,
					Price:        input.Price,
					CostPrice:    input.CostPrice,
					ComparePrice: input.ComparePrice,
					FloorPrice:   input.FloorPrice,
				})
			}
		}
	} else if len(input.Sizes) > 0 {
		// Only Sizes (e.g. Jeans size 30, 32, 34, 36)
		for _, size := range input.Sizes {
			cleanSize := strings.ToUpper(strings.ReplaceAll(size, " ", ""))
			sku := fmt.Sprintf("%s-%s", baseSKU, cleanSize)
			variantsToCreate = append(variantsToCreate, dto.CreateVariantRequest{
				SKU:          sku,
				Title:        fmt.Sprintf("Size %s", size),
				Size:         size,
				Price:        input.Price,
				CostPrice:    input.CostPrice,
				ComparePrice: input.ComparePrice,
				FloorPrice:   input.FloorPrice,
			})
		}
	} else if len(input.Shades) > 0 {
		// Cosmetics: Shades (e.g. Lipstick, Foundation shades)
		for _, shade := range input.Shades {
			cleanShade := strings.ToUpper(strings.ReplaceAll(shade, " ", ""))
			sku := fmt.Sprintf("%s-SHD-%s", baseSKU, cleanShade)
			variantsToCreate = append(variantsToCreate, dto.CreateVariantRequest{
				SKU:          sku,
				Title:        fmt.Sprintf("Shade %s", shade),
				ShadeCode:    shade,
				Price:        input.Price,
				CostPrice:    input.CostPrice,
				ComparePrice: input.ComparePrice,
				FloorPrice:   input.FloorPrice,
			})
		}
	} else {
		return nil, ErrInvalidMatrix
	}

	results, err := s.locationRepo.BulkCreateVariants(ctx, shop.ID, productID, variantsToCreate, input.InitialStockEach, input.LocationID)
	if err != nil {
		return nil, err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "VARIANT_MATRIX_GENERATED", map[string]interface{}{
		"product_id": productID,
		"count":      len(results),
	})
	return results, nil
}

// UpdateVariant modifies a variant's attributes or pricing.
func (s *LocationService) UpdateVariant(ctx context.Context, shopOwnerUserID, variantID string, input dto.UpdateVariantRequest) (*model.ProductVariant, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.UpdateVariant(ctx, shop.ID, variantID, input)
}

// DeleteVariant disables a variant SKU.
func (s *LocationService) DeleteVariant(ctx context.Context, shopOwnerUserID, variantID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return ErrShopNotFound
	}
	return s.locationRepo.DeleteVariant(ctx, shop.ID, variantID)
}

// CreateStockTransfer submits an inter-location stock transfer order.
func (s *LocationService) CreateStockTransfer(ctx context.Context, shopOwnerUserID string, input dto.CreateStockTransferRequest) (*model.StockTransfer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	if input.FromLocationID == input.ToLocationID {
		return nil, errors.New("source and destination locations cannot be identical")
	}

	transfer, err := s.locationRepo.CreateStockTransfer(ctx, shop.ID, shopOwnerUserID, input)
	if err != nil {
		return nil, err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "TRANSFER_CREATED", map[string]interface{}{
		"transfer_id":     transfer.ID,
		"transfer_number": transfer.TransferNumber,
	})
	return transfer, nil
}

// ListTransfers lists stock transfers for a merchant.
func (s *LocationService) ListTransfers(ctx context.Context, shopOwnerUserID string, status *model.TransferStatus) ([]*model.StockTransfer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.ListStockTransfers(ctx, shop.ID, status)
}

// GetTransfer fetches details of a specific transfer by ID.
func (s *LocationService) GetTransfer(ctx context.Context, shopOwnerUserID, transferID string) (*model.StockTransfer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.locationRepo.GetStockTransferByID(ctx, shop.ID, transferID)
}

// UpdateTransferStatus advances the transfer workflow (DISPATCHED, IN_TRANSIT, RECEIVED, RECONCILED).
func (s *LocationService) UpdateTransferStatus(ctx context.Context, shopOwnerUserID, transferID string, input dto.UpdateTransferStatusRequest) (*model.StockTransfer, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	transfer, err := s.locationRepo.UpdateStockTransferStatus(ctx, shop.ID, transferID, shopOwnerUserID, input.Status, input.Notes)
	if err != nil {
		return nil, err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "TRANSFER_STATUS_CHANGED", map[string]interface{}{
		"transfer_id": transfer.ID,
		"status":      transfer.Status,
	})
	return transfer, nil
}

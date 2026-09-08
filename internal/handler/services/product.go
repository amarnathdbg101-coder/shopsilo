// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"strings"
	"time"
)


type ProductService struct {
	productRepo  *repository.ProductRepo
	shopRepo     *repository.ShopRepo
	categoryRepo *repository.CategoryRepo
}

func NewProductService(
	productRepo *repository.ProductRepo,
	shopRepo *repository.ShopRepo,
	categoryRepo *repository.CategoryRepo,
) *ProductService {
	return &ProductService{
		productRepo:  productRepo,
		shopRepo:     shopRepo,
		categoryRepo: categoryRepo,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, userID string, input dto.CreateProductRequest) (*model.Product, error) {
	// 1. Verify user owns a shop
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("you must have a registered shop to create products")
	}

	// 2. Validate category
	if _, err := s.categoryRepo.FindByID(ctx, input.CategoryID); err != nil {
		return nil, ErrInvalidCategory
	}

	// 3. Enforce max 4 images (Cost & Storage efficiency rule)
	if len(input.Images) > 4 {
		return nil, ErrTooManyProductImages
	}

	// 4. Generate unique slug
	slug := reuse.Slugify(input.Name)
	if _, err := s.productRepo.FindBySlug(ctx, slug); err == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		slug = fmt.Sprintf("%s-%d", slug, r.Intn(9000)+1000)
	}

	minStock := 1
	if input.MinStock != nil && *input.MinStock > 0 {
		minStock = *input.MinStock
	} else if input.LowStockThreshold != nil && *input.LowStockThreshold > 0 {
		minStock = *input.LowStockThreshold
	}

	product := &model.Product{
		ShopID:            shop.ID,
		Name:              strings.TrimSpace(input.Name),
		Slug:              slug,
		Description:       strings.TrimSpace(input.Description),
		SKU:               strings.TrimSpace(strings.ToUpper(input.SKU)),
		Price:             input.Price,
		CostPrice:         input.CostPrice,
		ComparePrice:      input.ComparePrice,
		CategoryID:        input.CategoryID,
		Images:            input.Images,
		Weight:            input.Weight,
		IsActive:          true,
		IsFeatured:        input.IsFeatured,
		Tags:              input.Tags,
		Attributes:        input.Attributes,
		MinStock:          minStock,
		LowStockThreshold: minStock,
	}

	return s.productRepo.Create(ctx, product, input.StockQuantity)
}

func (s *ProductService) GetProductByID(ctx context.Context, id string) (*model.Product, error) {
	return s.productRepo.FindByID(ctx, id)
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*model.Product, error) {
	return s.productRepo.FindBySlug(ctx, slug)
}

func (s *ProductService) ListProducts(ctx context.Context, filter dto.ProductFilter) (*dto.ProductPaginationResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 12
	}

	products, totalCount, err := s.productRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	if products == nil {
		products = []*model.Product{}
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ProductPaginationResponse{
		Products:   products,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, userID, productID string, input dto.UpdateProductRequest) (*model.Product, error) {
	// 1. Verify user owns the shop
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	// 2. Fetch existing product
	existing, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if existing.ShopID != shop.ID {
		return nil, errors.New("unauthorized: you do not own this product")
	}

	// 3. Update fields
	if input.Name != nil {
		existing.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		existing.Description = strings.TrimSpace(*input.Description)
	}
	if input.SKU != nil {
		existing.SKU = strings.TrimSpace(strings.ToUpper(*input.SKU))
	}
	if input.Price != nil {
		existing.Price = *input.Price
	}
	if input.ComparePrice != nil {
		existing.ComparePrice = *input.ComparePrice
	}
	if input.CategoryID != nil {
		if _, err := s.categoryRepo.FindByID(ctx, *input.CategoryID); err != nil {
			return nil, ErrInvalidCategory
		}
		existing.CategoryID = *input.CategoryID
	}
	if input.Weight != nil {
		existing.Weight = *input.Weight
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	if input.IsFeatured != nil {
		existing.IsFeatured = *input.IsFeatured
	}
	if input.Tags != nil {
		existing.Tags = *input.Tags
	}
	if input.CostPrice != nil {
		existing.CostPrice = *input.CostPrice
	}
	if input.Attributes != nil {
		existing.Attributes = *input.Attributes
	}
	if input.MinStock != nil {
		val := *input.MinStock
		if val < 1 {
			val = 1
		}
		existing.MinStock = val
		existing.LowStockThreshold = val
	} else if input.LowStockThreshold != nil {
		val := *input.LowStockThreshold
		if val < 1 {
			val = 1
		}
		existing.MinStock = val
		existing.LowStockThreshold = val
	}

	// Handle images update & clean orphaned images from R2
	if input.Images != nil {
		if len(*input.Images) > 4 {
			return nil, ErrTooManyProductImages
		}

		newImagesMap := make(map[string]bool)
		for _, img := range *input.Images {
			newImagesMap[img] = true
		}

		// Delete images that were removed in the update to save R2 storage
		for _, oldImg := range existing.Images {
			if !newImagesMap[oldImg] {
				_ = reuse.DeleteImage(oldImg)
			}
		}

		existing.Images = *input.Images
	}

	return s.productRepo.Update(ctx, existing, input.StockQuantity)
}

func (s *ProductService) DeleteProduct(ctx context.Context, userID, productID string) error {
	// 1. Verify user owns the shop
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return errors.New("shop not found")
	}

	// 2. Fetch existing product to get its images
	existing, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return err
	}
	if existing.ShopID != shop.ID {
		return errors.New("unauthorized: you do not own this product")
	}

	// 3. Delete from database
	if err := s.productRepo.Delete(ctx, productID, shop.ID); err != nil {
		return err
	}

	// 4. Automatically delete all product images from R2 to avoid storage costs
	for _, img := range existing.Images {
		_ = reuse.DeleteImage(img)
	}

	return nil
}

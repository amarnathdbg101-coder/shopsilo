// Package services handle business logic.
package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/xuri/excelize/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/url"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
	"strconv"
	"strings"
	"time"
)

type ProductService struct {
	productRepo    *repository.ProductRepo
	shopRepo       *repository.ShopRepo
	categoryRepo   *repository.CategoryRepo
	catalogService *CatalogService
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

func (s *ProductService) SetCatalogService(cs *CatalogService) {
	s.catalogService = cs
}

// resolveCategoryID accepts UUID, slug, name, or auto-creates on the fly.
// Ensures product creation & updates never fail with ErrInvalidCategory.
func (s *ProductService) resolveCategoryID(ctx context.Context, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if def, err := s.categoryRepo.FindDefault(ctx); err == nil && def != nil {
			return def.ID, nil
		}
		return "", ErrInvalidCategory
	}

	// 1. If 36-char UUID format, try FindByID
	if len(trimmed) == 36 && strings.Count(trimmed, "-") == 4 {
		if category, err := s.categoryRepo.FindByID(ctx, trimmed); err == nil && category != nil {
			return category.ID, nil
		}
	}

	// 2. Try exact slug match
	if category, err := s.categoryRepo.FindBySlug(ctx, strings.ToLower(trimmed)); err == nil && category != nil {
		return category.ID, nil
	}

	// 2.5. Try canonical category alias resolution (e.g., "Grocery & Staples" -> "kirana-grocery")
	if canonSlug, ok := utils.ResolveCanonicalCategorySlug(trimmed); ok {
		if category, err := s.categoryRepo.FindBySlug(ctx, canonSlug); err == nil && category != nil {
			return category.ID, nil
		}
	}

	// 3. Try slugified match
	slugified := reuse.Slugify(trimmed)
	if slugified != "" && slugified != strings.ToLower(trimmed) {
		if category, err := s.categoryRepo.FindBySlug(ctx, slugified); err == nil && category != nil {
			return category.ID, nil
		}
	}

	// 4. Try name or partial search
	if category, err := s.categoryRepo.FindByNameOrSlug(ctx, trimmed); err == nil && category != nil {
		return category.ID, nil
	}

	// 5. If category is not in DB yet, dynamically insert it so merchant is never blocked
	if len(trimmed) >= 2 {
		catName := strings.Title(strings.ReplaceAll(trimmed, "-", " "))
		newCat, err := s.categoryRepo.Create(ctx, &model.Category{
			Name:     catName,
			Slug:     reuse.Slugify(trimmed),
			IsActive: true,
		})
		if err == nil && newCat != nil {
			return newCat.ID, nil
		}
	}

	// 6. Fallback to default category
	if def, err := s.categoryRepo.FindDefault(ctx); err == nil && def != nil {
		return def.ID, nil
	}

	return "", ErrInvalidCategory
}

func (s *ProductService) CreateProduct(ctx context.Context, userID string, input dto.CreateProductRequest) (*model.Product, error) {
	// 1. Verify user owns a shop
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	// 2. Validate category
	categoryID, err := s.resolveCategoryID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}

	sku := strings.TrimSpace(strings.ToUpper(input.SKU))
	images := input.Images

	// 3. Enforce max 4 images (Cost & Storage efficiency rule)
	if len(images) > 10 {
		return nil, ErrTooManyProductImages
	}

	// 5. Generate unique slug
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

	allowBargain := false
	if input.AllowBargain != nil {
		allowBargain = *input.AllowBargain
	}

	isPricePublic := true
	if input.IsPricePublic != nil {
		isPricePublic = *input.IsPricePublic
	} else if input.Attributes != nil {
		if val, exists := input.Attributes["is_price_public"]; exists {
			if b, ok := val.(bool); ok {
				isPricePublic = b
			} else if s, ok := val.(string); ok {
				isPricePublic = (s == "true" || s == "1")
			}
		}
	}

	attrs := input.Attributes
	if attrs == nil {
		attrs = make(map[string]interface{})
	}
	attrs["is_price_public"] = isPricePublic

	product := &model.Product{
		ShopID:            shop.ID,
		Name:              strings.TrimSpace(input.Name),
		Slug:              slug,
		Description:       strings.TrimSpace(input.Description),
		SKU:               sku,
		Price:             input.Price,
		CostPrice:         input.CostPrice,
		ComparePrice:      input.ComparePrice,
		FloorPrice:        input.FloorPrice,
		AllowBargain:      allowBargain,
		IsPricePublic:     isPricePublic,
		CategoryID:        categoryID,
		Images:            images,
		Weight:            input.Weight,
		IsActive:          true,
		IsFeatured:        input.IsFeatured,
		Tags:              input.Tags,
		Attributes:        attrs,
		MinStock:          minStock,
		LowStockThreshold: minStock,
	}

	created, err := s.productRepo.Create(ctx, product, input.StockQuantity)
	if err == nil && created != nil && s.catalogService != nil {
		barcode := sku
		if b, ok := attrs["barcode"].(string); ok && b != "" {
			barcode = b
		}
		brand := ""
		if br, ok := attrs["brand"].(string); ok {
			brand = br
		}
		unit := "piece"
		if u, ok := attrs["unit"].(string); ok {
			unit = u
		}
		img := ""
		if len(images) > 0 {
			img = images[0]
		}
		s.catalogService.EnrichFromProduct(
			ctx, barcode, created.Name, brand, created.CategoryName, unit, created.MRP, created.Price, created.Description, img, attrs,
		)
	}

	return created, err
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
		categoryID, err := s.resolveCategoryID(ctx, *input.CategoryID)
		if err != nil {
			return nil, err
		}
		existing.CategoryID = categoryID
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
	if input.FloorPrice != nil {
		existing.FloorPrice = *input.FloorPrice
	}
	if input.AllowBargain != nil {
		existing.AllowBargain = *input.AllowBargain
	}
	if input.IsPricePublic != nil {
		existing.IsPricePublic = *input.IsPricePublic
	} else if input.Attributes != nil {
		if val, exists := (*input.Attributes)["is_price_public"]; exists {
			if b, ok := val.(bool); ok {
				existing.IsPricePublic = b
			} else if s, ok := val.(string); ok {
				existing.IsPricePublic = (s == "true" || s == "1")
			}
		}
	}
	if input.Attributes != nil {
		existing.Attributes = *input.Attributes
	}
	if existing.Attributes == nil {
		existing.Attributes = make(map[string]interface{})
	}
	existing.Attributes["is_price_public"] = existing.IsPricePublic
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
		if len(*input.Images) > 10 {
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

	updated, err := s.productRepo.Update(ctx, existing, input.StockQuantity)
	if err == nil && updated != nil && s.catalogService != nil {
		barcode := updated.SKU
		if b, ok := updated.Attributes["barcode"].(string); ok && b != "" {
			barcode = b
		}
		brand := ""
		if br, ok := updated.Attributes["brand"].(string); ok {
			brand = br
		}
		unit := "piece"
		if u, ok := updated.Attributes["unit"].(string); ok {
			unit = u
		}
		img := ""
		if len(updated.Images) > 0 {
			img = updated.Images[0]
		}
		s.catalogService.EnrichFromProduct(
			ctx, barcode, updated.Name, brand, updated.CategoryName, unit, updated.MRP, updated.Price, updated.Description, img, updated.Attributes,
		)
	}

	return updated, err
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

	return nil
}

func (s *ProductService) BatchDeleteProducts(ctx context.Context, userID string, productIDs []string) (*dto.BatchOperationResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	count, err := s.productRepo.BatchDelete(ctx, shop.ID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to batch delete products: %w", err)
	}

	return &dto.BatchOperationResponse{
		AffectedCount: int(count),
		Message:       fmt.Sprintf("%d products deleted successfully", count),
	}, nil
}

func (s *ProductService) BatchToggleProductStatus(ctx context.Context, userID string, productIDs []string, isActive bool) (*dto.BatchOperationResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	count, err := s.productRepo.BatchToggleStatus(ctx, shop.ID, productIDs, isActive)
	if err != nil {
		return nil, fmt.Errorf("failed to batch update product status: %w", err)
	}

	statusText := "activated"
	if !isActive {
		statusText = "deactivated"
	}

	return &dto.BatchOperationResponse{
		AffectedCount: int(count),
		Message:       fmt.Sprintf("%d products %s successfully", count, statusText),
	}, nil
}

// FindNearbyProducts searches for in-stock products across nearby shops via full database query.
func (s *ProductService) FindNearbyProducts(
	ctx context.Context,
	lat, lng, radiusKm float64,
	query, category string,
	openNow bool,
	page, limit int,
) (*dto.NearbyProductResponse, error) {
	if radiusKm <= 0 {
		radiusKm = 10
	}
	if radiusKm > 50 {
		radiusKm = 50
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	filter := dto.ProductFilter{
		Search:     query,
		CategoryID: category,
		Page:       page,
		Limit:      limit,
		Lat:        &lat,
		Lng:        &lng,
		RadiusKm:   &radiusKm,
	}

	products, totalCount, err := s.productRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	nearbyItems := make([]*dto.NearbyProductItem, 0, len(products))
	for _, p := range products {
		if p == nil {
			continue
		}
		var shopLat, shopLng float64
		if p.ShopLatitude != nil {
			shopLat = *p.ShopLatitude
		}
		if p.ShopLongitude != nil {
			shopLng = *p.ShopLongitude
		}

		item := &dto.NearbyProductItem{
			ID:                p.ID,
			ProductID:         p.ID,
			ShopID:            p.ShopID,
			Name:              p.Name,
			Slug:              p.Slug,
			SKU:               p.SKU,
			Price:             p.Price,
			ComparePrice:      p.ComparePrice,
			CategoryID:        p.CategoryID,
			CategoryName:      p.CategoryName,
			Images:            p.Images,
			Attributes:        p.Attributes,
			AvailableQuantity: p.StockQuantity,
			IsInStock:         p.StockQuantity > 0,
			IsPricePublic:     p.IsPricePublic,
			AllowBargain:      p.AllowBargain,
			ShopName:          p.ShopName,
			ShopSlug:          p.ShopSlug,
			ShopPhone:         p.ShopPhone,
			ShopAddress:       p.ShopAddress,
			ShopCity:          p.ShopCity,
			ShopLatitude:      shopLat,
			ShopLongitude:     shopLng,
		}
		nearbyItems = append(nearbyItems, item)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	}

	return &dto.NearbyProductResponse{
		Products:   nearbyItems,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		RadiusKm:   radiusKm,
	}, nil
}

// ApplyClearanceMarkdown applies a markdown discount to a slow-moving product, tags it, and generates WhatsApp broadcast copy.
func (s *ProductService) ApplyClearanceMarkdown(ctx context.Context, userID, productID string, input dto.ApplyClearanceMarkdownRequest) (*dto.ClearanceMarkdownResponse, error) {
	if input.DiscountPercentage <= 0 || input.DiscountPercentage >= 100 {
		return nil, errors.New("discount percentage must be between 1 and 99")
	}

	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	existing, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	if existing.ShopID != shop.ID {
		return nil, errors.New("unauthorized: product does not belong to your shop")
	}

	originalPrice := existing.Price
	if existing.ComparePrice > 0 {
		originalPrice = existing.ComparePrice
	} else {
		existing.ComparePrice = originalPrice
	}

	// Calculate discounted price: round to 2 decimal places
	discountFactor := (100.0 - input.DiscountPercentage) / 100.0
	newPrice := math.Round(originalPrice*discountFactor*100) / 100
	if newPrice <= 0 {
		newPrice = 1.0
	}
	existing.Price = newPrice

	// Add "Clearance Sale" tag if not already present
	hasTag := false
	for _, t := range existing.Tags {
		if strings.EqualFold(t, "Clearance Sale") {
			hasTag = true
			break
		}
	}
	if !hasTag {
		existing.Tags = append(existing.Tags, "Clearance Sale")
	}

	updated, err := s.productRepo.Update(ctx, existing, nil)
	if err != nil {
		return nil, err
	}

	stock := 0
	if updated.Inventory != nil {
		stock = updated.Inventory.AvailableQuantity
	}

	customMsg := strings.TrimSpace(input.CustomMessage)
	if customMsg != "" {
		customMsg = " " + customMsg
	}

	broadcastText := fmt.Sprintf(
		"🔥 CLEARANCE SALE at %s! %s is now available at %.0f%% OFF! Original: Rs.%.2f, Offer Price: Rs.%.2f. Only %d left in stock!%s Visit us at %s or reply to order.",
		shop.Name, updated.Name, input.DiscountPercentage, originalPrice, newPrice, stock, customMsg, shop.Address,
	)

	waURL := fmt.Sprintf("https://wa.me/?text=%s", url.QueryEscape(broadcastText))

	return &dto.ClearanceMarkdownResponse{
		ProductID:          updated.ID,
		Name:               updated.Name,
		OriginalPrice:      originalPrice,
		DiscountedPrice:    newPrice,
		DiscountPercentage: input.DiscountPercentage,
		StockQuantity:      stock,
		Tags:               updated.Tags,
		BroadcastMessage:   broadcastText,
		WhatsAppShareURL:   waURL,
	}, nil
}

// NegotiateBargainOffer handles shopper offer proposals using dynamic margin protection and sweet-spot counter offers.
func (s *ProductService) NegotiateBargainOffer(ctx context.Context, productID string, req dto.MakeOfferRequest) (*dto.BargainNegotiationResponse, error) {
	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	shop, _ := s.shopRepo.FindByID(ctx, prod.ShopID)
	shopName := "Store"
	shopWhatsApp := ""
	if shop != nil {
		shopName = shop.Name
		shopWhatsApp = shop.WhatsAppNumber
		if shopWhatsApp == "" {
			shopWhatsApp = shop.Phone
		}
	}

	// 1. If bargaining is explicitly disabled for this product
	if !prod.AllowBargain {
		return &dto.BargainNegotiationResponse{
			Status:        "BARGAIN_DISABLED",
			ProductID:     prod.ID,
			ProductName:   prod.Name,
			OriginalPrice: prod.Price,
			OfferedPrice:  req.OfferedPrice,
			AgreedPrice:   prod.Price,
			Message:       "This item has fixed price as per store policy.",
		}, nil
	}

	// 2. Determine effective floor price (bottom margin line)
	floorPrice := prod.FloorPrice
	if floorPrice <= 0 {
		// Default safe floor: at least cost + 15% margin or 80% of price
		costWithMargin := prod.CostPrice * 1.15
		priceThreshold := prod.Price * 0.80
		if costWithMargin > 0 && costWithMargin < prod.Price {
			floorPrice = math.Max(costWithMargin, priceThreshold)
		} else {
			floorPrice = priceThreshold
		}
	}
	floorPrice = math.Round(floorPrice*100) / 100

	cleanPhone := strings.TrimSpace(req.CustomerPhone)
	cleanName := strings.TrimSpace(req.CustomerName)
	if cleanName == "" {
		cleanName = "Valued Customer"
	}

	// 3. Evaluate customer offer
	if req.OfferedPrice >= prod.Price {
		// Customer offered full price or above
		dealCode := generateDealCode()
		expiresAt := time.Now().Add(30 * time.Minute)
		deal := &model.ProductBargainDeal{
			ProductID:      prod.ID,
			ShopID:         prod.ShopID,
			CustomerPhone:  cleanPhone,
			CustomerName:   cleanName,
			DealCode:       dealCode,
			OfferedPrice:   req.OfferedPrice,
			AgreedPrice:    prod.Price,
			BundleQuantity: 1,
			Status:         "accepted",
			ExpiresAt:      expiresAt,
		}
		_ = s.productRepo.CreateBargainDeal(ctx, deal)

		waMsg := fmt.Sprintf("Hi %s, I would like to purchase %s at Rs.%.2f. Deal Code: %s", shopName, prod.Name, prod.Price, dealCode)
		waURL := fmt.Sprintf("https://wa.me/%s?text=%s", shopWhatsApp, url.QueryEscape(waMsg))

		return &dto.BargainNegotiationResponse{
			Status:            "DEAL_ACCEPTED",
			ProductID:         prod.ID,
			ProductName:       prod.Name,
			OriginalPrice:     prod.Price,
			OfferedPrice:      req.OfferedPrice,
			AgreedPrice:       prod.Price,
			SavingsAmount:     0,
			SavingsPercentage: 0,
			DealCode:          dealCode,
			ExpiresAt:         &expiresAt,
			Message:           fmt.Sprintf("Deal accepted! Order %s at standard price Rs.%.2f.", prod.Name, prod.Price),
			WhatsAppOrderURL:  waURL,
		}, nil
	}

	if req.OfferedPrice >= floorPrice {
		// Customer offer is above floor price -> Accept!
		agreedPrice := math.Round(req.OfferedPrice*100) / 100
		savings := prod.Price - agreedPrice
		savingsPct := math.Round((savings/prod.Price)*1000) / 10
		dealCode := generateDealCode()
		expiresAt := time.Now().Add(30 * time.Minute)

		deal := &model.ProductBargainDeal{
			ProductID:      prod.ID,
			ShopID:         prod.ShopID,
			CustomerPhone:  cleanPhone,
			CustomerName:   cleanName,
			DealCode:       dealCode,
			OfferedPrice:   req.OfferedPrice,
			AgreedPrice:    agreedPrice,
			BundleQuantity: 1,
			Status:         "accepted",
			ExpiresAt:      expiresAt,
		}
		_ = s.productRepo.CreateBargainDeal(ctx, deal)

		waMsg := fmt.Sprintf("Hi %s! Deal Code: %s. My bargained price of Rs.%.2f for %s has been accepted. Please confirm my order!", shopName, dealCode, agreedPrice, prod.Name)
		waURL := fmt.Sprintf("https://wa.me/%s?text=%s", shopWhatsApp, url.QueryEscape(waMsg))

		return &dto.BargainNegotiationResponse{
			Status:            "DEAL_ACCEPTED",
			ProductID:         prod.ID,
			ProductName:       prod.Name,
			OriginalPrice:     prod.Price,
			OfferedPrice:      req.OfferedPrice,
			AgreedPrice:       agreedPrice,
			SavingsAmount:     savings,
			SavingsPercentage: savingsPct,
			DealCode:          dealCode,
			ExpiresAt:         &expiresAt,
			Message:           fmt.Sprintf("Congratulations! %s accepted your offer of Rs.%.2f (You save Rs.%.2f / %.1f%%). Deal locked for 30 minutes!", shopName, agreedPrice, savings, savingsPct),
			WhatsAppOrderURL:  waURL,
		}, nil
	}

	// 4. Offer is below floor price -> Generate smart sweet-spot counter-offer & bundle upsell!
	counterPrice := math.Round((floorPrice+0.35*(prod.Price-floorPrice))*100) / 100
	if counterPrice < floorPrice {
		counterPrice = floorPrice
	}
	savings := prod.Price - counterPrice
	savingsPct := math.Round((savings/prod.Price)*1000) / 10
	dealCode := generateDealCode()
	expiresAt := time.Now().Add(30 * time.Minute)

	deal := &model.ProductBargainDeal{
		ProductID:      prod.ID,
		ShopID:         prod.ShopID,
		CustomerPhone:  cleanPhone,
		CustomerName:   cleanName,
		DealCode:       dealCode,
		OfferedPrice:   req.OfferedPrice,
		AgreedPrice:    counterPrice,
		BundleQuantity: 1,
		Status:         "counter_offered",
		ExpiresAt:      expiresAt,
	}
	_ = s.productRepo.CreateBargainDeal(ctx, deal)

	// Calculate bundle upsell (Buy 2 for X)
	bundleUnitPrice := math.Round((floorPrice+0.10*(prod.Price-floorPrice))*100) / 100
	totalBundlePrice := bundleUnitPrice * 2
	bundleSavings := (prod.Price * 2) - totalBundlePrice

	bundleSuggestion := &dto.BundleUpsellSuggestion{
		BundleQuantity:   2,
		UnitPrice:        bundleUnitPrice,
		TotalBundlePrice: totalBundlePrice,
		TotalSavings:     bundleSavings,
		Description:      fmt.Sprintf("Buy 2 pieces at Rs.%.2f each (Total Rs.%.2f). You save Rs.%.2f!", bundleUnitPrice, totalBundlePrice, bundleSavings),
	}

	waMsg := fmt.Sprintf("Hi %s! Deal Code: %s. I want to lock the special counter-offer of Rs.%.2f for %s. Please confirm!", shopName, dealCode, counterPrice, prod.Name)
	waURL := fmt.Sprintf("https://wa.me/%s?text=%s", shopWhatsApp, url.QueryEscape(waMsg))

	return &dto.BargainNegotiationResponse{
		Status:            "COUNTER_OFFER",
		ProductID:         prod.ID,
		ProductName:       prod.Name,
		OriginalPrice:     prod.Price,
		OfferedPrice:      req.OfferedPrice,
		AgreedPrice:       counterPrice,
		SavingsAmount:     savings,
		SavingsPercentage: savingsPct,
		DealCode:          dealCode,
		ExpiresAt:         &expiresAt,
		Message:           fmt.Sprintf("Rs.%.2f is below our wholesale cost. But as a special store gesture, we can offer it for Rs.%.2f! (You save Rs.%.2f / %.1f%%).", req.OfferedPrice, counterPrice, savings, savingsPct),
		BundleUpsell:      bundleSuggestion,
		WhatsAppOrderURL:  waURL,
	}, nil
}

// GetPOSBargainAssist provides real-time margin guidance for counter cashiers negotiating in person.
func (s *ProductService) GetPOSBargainAssist(ctx context.Context, shopOwnerUserID, productID string, proposedPrice float64) (*dto.POSBargainAssistResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	if prod.ShopID != shop.ID {
		return nil, errors.New("product does not belong to your shop")
	}

	floorPrice := prod.FloorPrice
	if floorPrice <= 0 {
		costWithMargin := prod.CostPrice * 1.15
		priceThreshold := prod.Price * 0.80
		if costWithMargin > 0 && costWithMargin < prod.Price {
			floorPrice = math.Max(costWithMargin, priceThreshold)
		} else {
			floorPrice = priceThreshold
		}
	}
	floorPrice = math.Round(floorPrice*100) / 100

	profit := proposedPrice - prod.CostPrice
	marginPct := 0.0
	if proposedPrice > 0 {
		marginPct = math.Round((profit/proposedPrice)*1000) / 10
	}

	statusColor := "green"
	advice := fmt.Sprintf("Safe Deal! Profit: Rs.%.2f (%.1f%% margin). Accept offer.", profit, marginPct)

	if proposedPrice < prod.CostPrice || proposedPrice < floorPrice {
		statusColor = "red"
		advice = fmt.Sprintf("Loss Alert! Proposed price Rs.%.2f is below floor threshold Rs.%.2f. Do not accept.", proposedPrice, floorPrice)
	} else if proposedPrice < (floorPrice + (prod.Price-floorPrice)*0.4) {
		statusColor = "yellow"
		advice = fmt.Sprintf("Floor Limit! Profit: Rs.%.2f (%.1f%% margin). Breakeven margin, do not discount further.", profit, marginPct)
	}

	return &dto.POSBargainAssistResponse{
		ProductID:        prod.ID,
		ProductName:      prod.Name,
		DisplayPrice:     prod.Price,
		CostPrice:        prod.CostPrice,
		FloorPrice:       floorPrice,
		ProposedPrice:    proposedPrice,
		MarginPercentage: marginPct,
		ProfitAmount:     profit,
		StatusColor:      statusColor,
		Advice:           advice,
	}, nil
}

func generateDealCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Helper to clean number strings containing currency symbols, commas, percent, etc.
func cleanImportNumber(val string) string {
	v := strings.TrimSpace(val)
	v = strings.Trim(v, "\uFEFF\"' ")
	var b strings.Builder
	hasDot := false
	hasDigit := false
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			hasDigit = true
		} else if r == '.' && !hasDot {
			b.WriteRune(r)
			hasDot = true
		} else if r == '-' && b.Len() == 0 {
			b.WriteRune(r)
		}
	}
	if !hasDigit {
		return "0"
	}
	return b.String()
}

// Helper to normalize SKU string, handling scientific notation from Excel (e.g. 8.90103E+12)
func cleanImportSKU(val string) string {
	s := strings.TrimSpace(val)
	s = strings.Trim(s, "\uFEFF\"' ")
	if strings.Contains(s, "E+") || strings.Contains(s, "e+") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return fmt.Sprintf("%.0f", f)
		}
	}
	return s
}

// parseRecordsToImportItems extracts product items from a 2D grid of CSV/Excel cells.
func parseRecordsToImportItems(records [][]string) []dto.BulkImportProductItem {
	if len(records) <= 1 {
		return nil
	}

	header := records[0]
	nameCol := -1
	skuCol := -1
	priceCol := -1
	costCol := -1
	compareCol := -1
	stockCol := -1
	minStockCol := -1
	catCol := -1
	unitCol := -1
	descCol := -1
	brandCol := -1

	type attrPair struct {
		nameCol int
		valCol  int
	}
	var attrPairs []attrPair

	for i, h := range header {
		cleanH := strings.ToLower(strings.Trim(strings.TrimSpace(h), "\uFEFF\"' "))
		isAttr := strings.Contains(cleanH, "attribute") || strings.Contains(cleanH, "attr")
		isBadge := strings.Contains(cleanH, "badge") || strings.Contains(cleanH, "featured")
		isDiscount := strings.Contains(cleanH, "discount") || strings.Contains(cleanH, "%") ||
			strings.Contains(cleanH, "off") || strings.Contains(cleanH, "profit") || strings.Contains(cleanH, "margin")
		isAlert := strings.Contains(cleanH, "alert") || strings.Contains(cleanH, "threshold") ||
			strings.Contains(cleanH, "min") || strings.Contains(cleanH, "low") || strings.Contains(cleanH, "reorder")
		isCost := strings.Contains(cleanH, "cost") || strings.Contains(cleanH, "khared") ||
			strings.Contains(cleanH, "purchase") || strings.Contains(cleanH, "wholesale") || strings.Contains(cleanH, "buy")
		isCompare := strings.Contains(cleanH, "mrp") || strings.Contains(cleanH, "compare") ||
			strings.Contains(cleanH, "original") || strings.Contains(cleanH, "market") || strings.Contains(cleanH, "list")
		isStock := strings.Contains(cleanH, "stock") || strings.Contains(cleanH, "qty") ||
			strings.Contains(cleanH, "quantity") || strings.Contains(cleanH, "inventory") || strings.Contains(cleanH, "units")

		// 1. Product Name (Strict: cannot be attribute, badge, discount, alert, or cost)
		if !isAttr && !isBadge && !isDiscount && !isAlert && !isCost {
			if cleanH == "product name" || cleanH == "product_name" || cleanH == "item name" || cleanH == "item_name" ||
				cleanH == "title" || cleanH == "name" || cleanH == "product" || cleanH == "item" || cleanH == "saman" || cleanH == "naam" {
				nameCol = i
			} else if nameCol == -1 && (strings.Contains(cleanH, "name") || strings.Contains(cleanH, "title") || strings.Contains(cleanH, "product") || strings.Contains(cleanH, "item")) &&
				!strings.Contains(cleanH, "category") && !strings.Contains(cleanH, "brand") {
				nameCol = i
			}
		}

		// 2. SKU / Barcode
		if !isAttr && !isBadge {
			if strings.Contains(cleanH, "sku") || strings.Contains(cleanH, "barcode") ||
				strings.Contains(cleanH, "item code") || strings.Contains(cleanH, "product code") ||
				strings.Contains(cleanH, "upc") || strings.Contains(cleanH, "ean") || cleanH == "code" {
				if skuCol == -1 {
					skuCol = i
				}
			}
		}

		// 3. Cost Price
		if isCost && !isDiscount {
			if costCol == -1 {
				costCol = i
			}
		}

		// 4. MRP / Compare Price (strictly exclude discount/margin)
		if isCompare && !isDiscount {
			if compareCol == -1 {
				compareCol = i
			}
		}

		// 5. Selling Price
		if !isCost && !isCompare && !isDiscount {
			if strings.Contains(cleanH, "selling") || strings.Contains(cleanH, "sell") ||
				strings.Contains(cleanH, "sales") || strings.Contains(cleanH, "price") ||
				strings.Contains(cleanH, "rate") || cleanH == "sp" {
				if priceCol == -1 {
					priceCol = i
				}
			}
		}

		// 6. Min Stock / Low Stock Alert
		if isAlert {
			if minStockCol == -1 {
				minStockCol = i
			}
		}

		// 7. Stock Quantity (strictly exclude alert / min / low)
		if isStock && !isAlert {
			if stockCol == -1 {
				stockCol = i
			}
		}

		// 8. Category
		if !isAttr {
			if strings.Contains(cleanH, "category") || cleanH == "cat" ||
				strings.Contains(cleanH, "group") || strings.Contains(cleanH, "department") {
				if catCol == -1 {
					catCol = i
				}
			}
		}

		// 9. Unit
		if !isAttr {
			if strings.Contains(cleanH, "unit") || strings.Contains(cleanH, "uom") ||
				cleanH == "pack" || cleanH == "packaging" {
				if unitCol == -1 {
					unitCol = i
				}
			}
		}

		// 10. Description
		if strings.Contains(cleanH, "desc") || strings.Contains(cleanH, "detail") ||
			strings.Contains(cleanH, "notes") || strings.Contains(cleanH, "about") {
			if descCol == -1 {
				descCol = i
			}
		}

		// 11. Brand
		if !isAttr {
			if strings.Contains(cleanH, "brand") || strings.Contains(cleanH, "company") ||
				strings.Contains(cleanH, "manufacturer") || strings.Contains(cleanH, "mfg") {
				if brandCol == -1 {
					brandCol = i
				}
			}
		}

		// 12. Attribute pair detection (e.g. Attribute 1 Name, Attribute 1 Value)
		if isAttr && strings.Contains(cleanH, "name") {
			if i+1 < len(header) && strings.Contains(strings.ToLower(header[i+1]), "val") {
				attrPairs = append(attrPairs, attrPair{nameCol: i, valCol: i + 1})
			}
		}
	}

	if nameCol == -1 {
		nameCol = 0
	}
	if priceCol == -1 {
		priceCol = 1
	}

	var items []dto.BulkImportProductItem

	for _, row := range records[1:] {
		if len(row) == 0 {
			continue
		}

		name := ""
		if nameCol < len(row) {
			name = strings.TrimSpace(row[nameCol])
		}
		if name == "" {
			continue
		}

		item := dto.BulkImportProductItem{
			Name: name,
		}

		if skuCol != -1 && skuCol < len(row) {
			item.SKU = cleanImportSKU(row[skuCol])
		}

		if priceCol < len(row) {
			pVal, _ := strconv.ParseFloat(cleanImportNumber(row[priceCol]), 64)
			if pVal <= 0 {
				pVal = 10.0 // Default price
			}
			item.Price = pVal
		} else {
			item.Price = 10.0
		}

		if costCol != -1 && costCol < len(row) {
			cVal, _ := strconv.ParseFloat(cleanImportNumber(row[costCol]), 64)
			item.CostPrice = cVal
		}

		if compareCol != -1 && compareCol < len(row) {
			cmpVal, _ := strconv.ParseFloat(cleanImportNumber(row[compareCol]), 64)
			item.ComparePrice = cmpVal
		}

		if stockCol != -1 && stockCol < len(row) {
			sVal, _ := strconv.Atoi(cleanImportNumber(row[stockCol]))
			item.StockQuantity = sVal
		} else {
			item.StockQuantity = 10
		}

		if minStockCol != -1 && minStockCol < len(row) {
			mVal, _ := strconv.Atoi(cleanImportNumber(row[minStockCol]))
			item.MinStock = mVal
		} else {
			item.MinStock = 5
		}

		if catCol != -1 && catCol < len(row) {
			item.CategoryName = strings.TrimSpace(row[catCol])
		}

		if unitCol != -1 && unitCol < len(row) {
			unitVal := strings.TrimSpace(row[unitCol])
			if unitVal != "" {
				item.Unit = unitVal
			} else {
				item.Unit = "Piece"
			}
		} else {
			item.Unit = "Piece"
		}

		baseDesc := ""
		if descCol != -1 && descCol < len(row) {
			baseDesc = strings.TrimSpace(row[descCol])
		}

		brandVal := ""
		if brandCol != -1 && brandCol < len(row) {
			brandVal = strings.TrimSpace(row[brandCol])
		}

		var extraParts []string
		if brandVal != "" && !strings.Contains(strings.ToLower(baseDesc), strings.ToLower(brandVal)) {
			extraParts = append(extraParts, fmt.Sprintf("Brand: %s", brandVal))
		}

		for _, ap := range attrPairs {
			if ap.nameCol < len(row) && ap.valCol < len(row) {
				an := strings.TrimSpace(row[ap.nameCol])
				av := strings.TrimSpace(row[ap.valCol])
				if an != "" && av != "" {
					extraParts = append(extraParts, fmt.Sprintf("%s: %s", an, av))
				}
			}
		}

		if len(extraParts) > 0 {
			extras := strings.Join(extraParts, " | ")
			if baseDesc != "" {
				item.Description = fmt.Sprintf("%s | %s", baseDesc, extras)
			} else {
				item.Description = extras
			}
		} else {
			item.Description = baseDesc
		}

		items = append(items, item)
	}

	return items
}

// BulkImportProductsFromCSV parses CSV bytes and bulk imports products into shop catalog in batch.
func (s *ProductService) BulkImportProductsFromCSV(ctx context.Context, userID string, csvReader io.Reader, updateExisting bool) (*dto.BulkImportResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	rawBytes, err := io.ReadAll(csvReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV data: %w", err)
	}

	// Strip UTF-8 BOM if present
	if len(rawBytes) >= 3 && rawBytes[0] == 0xEF && rawBytes[1] == 0xBB && rawBytes[2] == 0xBF {
		rawBytes = rawBytes[3:]
	}

	// Delimiter detection: check for semicolon or tab if comma isn't prominent
	commaCount := bytes.Count(rawBytes, []byte(","))
	semiCount := bytes.Count(rawBytes, []byte(";"))
	tabCount := bytes.Count(rawBytes, []byte("	"))

	var delimiter rune = ','
	if semiCount > commaCount && semiCount > tabCount {
		delimiter = ';'
	} else if tabCount > commaCount && tabCount > semiCount {
		delimiter = '	'
	}

	reader := csv.NewReader(bytes.NewReader(rawBytes))
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // flexible column count per row

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV file format: %w", err)
	}

	if len(records) <= 1 {
		return nil, errors.New("CSV file is empty or missing data rows")
	}

	items := parseRecordsToImportItems(records)
	if len(items) == 0 {
		return nil, errors.New("no valid product rows found in CSV")
	}

	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
}

// BulkImportProductsFromExcel parses Excel (.xlsx) data and bulk imports products into shop catalog.
func (s *ProductService) BulkImportProductsFromExcel(ctx context.Context, userID string, excelReader io.Reader, updateExisting bool) (*dto.BulkImportResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	f, err := excelize.OpenReader(excelReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel workbook contains no sheets")
	}

	// Read rows from the first sheet
	records, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(records) <= 1 {
		return nil, errors.New("Excel sheet is empty or contains no product rows")
	}

	items := parseRecordsToImportItems(records)
	if len(items) == 0 {
		return nil, errors.New("no valid product rows found in Excel sheet")
	}

	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
}

// BulkImportProductsFromFile automatically detects CSV vs Excel format and imports products.
func (s *ProductService) BulkImportProductsFromFile(ctx context.Context, userID string, filename string, fileReader io.Reader, updateExisting bool) (*dto.BulkImportResponse, error) {
	rawBytes, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	lowerName := strings.ToLower(filename)
	isExcel := strings.HasSuffix(lowerName, ".xlsx") ||
		strings.HasSuffix(lowerName, ".xlsm") ||
		strings.HasSuffix(lowerName, ".xls") ||
		bytes.HasPrefix(rawBytes, []byte("PK"))

	if isExcel {
		return s.BulkImportProductsFromExcel(ctx, userID, bytes.NewReader(rawBytes), updateExisting)
	}

	return s.BulkImportProductsFromCSV(ctx, userID, bytes.NewReader(rawBytes), updateExisting)
}

// BulkImportProductsFromJSON bulk imports products provided via JSON array.
func (s *ProductService) BulkImportProductsFromJSON(ctx context.Context, userID string, items []dto.BulkImportProductItem, updateExisting bool) (*dto.BulkImportResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
}

// GenerateCSVImportTemplate returns a ready-to-use sample CSV template for shopkeepers.
func (s *ProductService) GenerateCSVImportTemplate() []byte {
	template := `Name,SKU,Price,CostPrice,ComparePrice,StockQuantity,MinStock,Category,Unit,Description
Aashirvaad Shuddh Chakki Atta 5kg,ATT-5KG,245.00,210.00,270.00,50,5,Atta & Flour,kg,5kg Whole Wheat Flour Pack
Fortune Sunlite Refined Oil 1L,OIL-1L,135.00,115.00,150.00,40,5,Oils & Ghee,L,1 Liter Pouch
Tata Salt Iodized 1kg,SALT-1KG,28.00,22.00,30.00,100,10,Masalas & Spices,kg,1kg Vacuum Evaporated Iodized Salt
Dettol Original Soap 125g,DET-125G,55.00,45.00,60.00,60,8,Personal Care,pcs,Antiseptic Bathing Bar
Maggi 2-Minute Masala Noodles 70g,MAG-70G,14.00,11.50,15.00,120,15,Snacks & Instant,pack,Instant Noodles Pack
`
	return []byte(template)
}

// GenerateExcelImportTemplate creates a professionally formatted sample Excel spreadsheet.
func (s *ProductService) GenerateExcelImportTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Products"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Product Name *", "SKU / Barcode", "Selling Price (₹) *", "Cost Price (₹)",
		"MRP / Compare Price (₹)", "Stock Quantity", "Min Stock Alert", "Category", "Unit", "Description",
	}

	// Write header row
	for colIdx, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// Style header
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err == nil {
		_ = f.SetRowStyle(sheet, 1, 1, headerStyle)
	}
	_ = f.SetRowHeight(sheet, 1, 26)

	// Sample rows
	sampleData := [][]interface{}{
		{"Aashirvaad Shuddh Chakki Atta 5kg", "ATT-5KG", 245.00, 210.00, 270.00, 50, 5, "Atta & Flour", "kg", "5kg Whole Wheat Flour Pack"},
		{"Fortune Sunlite Refined Oil 1L", "OIL-1L", 135.00, 115.00, 150.00, 40, 5, "Oils & Ghee", "L", "1 Liter Pouch"},
		{"Tata Salt Iodized 1kg", "SALT-1KG", 28.00, 22.00, 30.00, 100, 10, "Masalas & Spices", "kg", "1kg Vacuum Evaporated Iodized Salt"},
		{"Dettol Original Soap 125g", "DET-125G", 55.00, 45.00, 60.00, 60, 8, "Personal Care", "pcs", "Antiseptic Bathing Bar"},
		{"Maggi 2-Minute Masala Noodles 70g", "MAG-70G", 14.00, 11.50, 15.00, 120, 15, "Snacks & Instant", "pack", "Instant Noodles Pack"},
	}

	for rowIdx, rowData := range sampleData {
		rNum := rowIdx + 2
		for colIdx, val := range rowData {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rNum)
			_ = f.SetCellValue(sheet, cell, val)
		}
		_ = f.SetRowHeight(sheet, rNum, 20)
	}

	// Set column widths
	colWidths := map[string]float64{
		"A": 36, "B": 18, "C": 18, "D": 16, "E": 22,
		"F": 16, "G": 16, "H": 20, "I": 12, "J": 36,
	}
	for col, width := range colWidths {
		_ = f.SetColWidth(sheet, col, col, width)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
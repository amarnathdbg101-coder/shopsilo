// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/url"
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

// FindNearbyProducts searches for in-stock products across nearby shops.
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
	if limit < 1 || limit > 50 {
		limit = 10
	}

	items, totalCount, err := s.productRepo.FindNearbyProducts(ctx, lat, lng, radiusKm, query, category, openNow, page, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*dto.NearbyProductItem{}
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	}

	return &dto.NearbyProductResponse{
		Products:   items,
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
			Status:           "DEAL_ACCEPTED",
			ProductID:        prod.ID,
			ProductName:      prod.Name,
			OriginalPrice:    prod.Price,
			OfferedPrice:     req.OfferedPrice,
			AgreedPrice:      prod.Price,
			SavingsAmount:    0,
			SavingsPercentage: 0,
			DealCode:         dealCode,
			ExpiresAt:        &expiresAt,
			Message:          fmt.Sprintf("Deal accepted! Order %s at standard price Rs.%.2f.", prod.Name, prod.Price),
			WhatsAppOrderURL: waURL,
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



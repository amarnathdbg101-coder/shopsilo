// Package services handle business logic.
package services

import (
	"bytes"
	"context"
	"encoding/csv"
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
	"strconv"
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

// resolveCategoryID accepts both the persisted UUID and the frontend-owned slug.
// This keeps the category catalog decoupled from the product form while preserving the DB relation.
func (s *ProductService) resolveCategoryID(ctx context.Context, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if category, err := s.categoryRepo.FindByID(ctx, trimmed); err == nil {
		return category.ID, nil
	}
	category, err := s.categoryRepo.FindBySlug(ctx, strings.ToLower(trimmed))
	if err != nil {
		return "", ErrInvalidCategory
	}
	return category.ID, nil
}

func (s *ProductService) CreateProduct(ctx context.Context, userID string, input dto.CreateProductRequest) (*model.Product, error) {
	// 1. Verify user owns a shop
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("you must have a registered shop to create products")
	}

	// 2. Validate category
	categoryID, err := s.resolveCategoryID(ctx, input.CategoryID)
	if err != nil {
		return nil, err
	}

	sku := strings.TrimSpace(strings.ToUpper(input.SKU))
	images := input.Images

	// 3. Enforce max 4 images (Cost & Storage efficiency rule)
	if len(images) > 4 {
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

	product := &model.Product{
		ShopID:            shop.ID,
		Name:              strings.TrimSpace(input.Name),
		Slug:              slug,
		Description:       strings.TrimSpace(input.Description),
		SKU:               sku,
		Price:             input.Price,
		CostPrice:         input.CostPrice,
		ComparePrice:      input.ComparePrice,
		CategoryID:        categoryID,
		Images:            images,
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

	// Note: We intentionally do NOT delete the images from Cloudflare R2 here.
	// Because the product is only soft-deleted (is_active = false) so that past
	// POS sales, receipts, and Khata bills can still display the product image.

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
	tabCount := bytes.Count(rawBytes, []byte("\t"))

	var delimiter rune = ','
	if semiCount > commaCount && semiCount > tabCount {
		delimiter = ';'
	} else if tabCount > commaCount && tabCount > semiCount {
		delimiter = '\t'
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

	// Identify header positions
	header := records[0]
	nameCol, skuCol, priceCol, costCol, compareCol, stockCol, minStockCol, catCol, unitCol, descCol := -1, -1, -1, -1, -1, -1, -1, -1, -1, -1

	for i, h := range header {
		cleanH := strings.ToLower(strings.TrimSpace(h))
		if strings.Contains(cleanH, "name") || strings.Contains(cleanH, "title") || strings.Contains(cleanH, "item") || strings.Contains(cleanH, "saman") {
			nameCol = i
		} else if strings.Contains(cleanH, "sku") || strings.Contains(cleanH, "code") || strings.Contains(cleanH, "barcode") || strings.Contains(cleanH, "upc") || strings.Contains(cleanH, "ean") {
			skuCol = i
		} else if strings.Contains(cleanH, "mrp") || strings.Contains(cleanH, "compare") || strings.Contains(cleanH, "original") {
			compareCol = i
		} else if strings.Contains(cleanH, "price") || strings.Contains(cleanH, "rate") || strings.Contains(cleanH, "selling") {
			if strings.Contains(cleanH, "cost") || strings.Contains(cleanH, "buy") || strings.Contains(cleanH, "wholesale") || strings.Contains(cleanH, "purchase") {
				costCol = i
			} else if priceCol == -1 {
				priceCol = i
			}
		} else if strings.Contains(cleanH, "cost") || strings.Contains(cleanH, "wholesale") || strings.Contains(cleanH, "buy") || strings.Contains(cleanH, "purchase") {
			costCol = i
		} else if strings.Contains(cleanH, "stock") || strings.Contains(cleanH, "qty") || strings.Contains(cleanH, "quantity") || strings.Contains(cleanH, "units") {
			stockCol = i
		} else if strings.Contains(cleanH, "min") || strings.Contains(cleanH, "threshold") || strings.Contains(cleanH, "alert") {
			minStockCol = i
		} else if strings.Contains(cleanH, "category") || strings.Contains(cleanH, "cat") || strings.Contains(cleanH, "group") || strings.Contains(cleanH, "type") {
			catCol = i
		} else if strings.Contains(cleanH, "unit") || strings.Contains(cleanH, "uom") || strings.Contains(cleanH, "pack") {
			unitCol = i
		} else if strings.Contains(cleanH, "desc") || strings.Contains(cleanH, "detail") {
			descCol = i
		}
	}

	if nameCol == -1 {
		nameCol = 0
	}
	if priceCol == -1 {
		priceCol = 1
	}

	cleanNumber := func(val string) string {
		v := strings.TrimSpace(val)
		v = strings.ReplaceAll(v, "₹", "")
		v = strings.ReplaceAll(v, "Rs.", "")
		v = strings.ReplaceAll(v, "Rs", "")
		v = strings.ReplaceAll(v, "INR", "")
		v = strings.ReplaceAll(v, ",", "")
		return strings.TrimSpace(v)
	}

	cleanSKU := func(val string) string {
		s := strings.TrimSpace(val)
		if strings.Contains(s, "E+") || strings.Contains(s, "e+") {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return fmt.Sprintf("%.0f", f)
			}
		}
		return s
	}

	var items []dto.BulkImportProductItem

	for _, row := range records[1:] {
		if len(row) == 0 {
			continue
		}

		item := dto.BulkImportProductItem{}

		if nameCol < len(row) {
			item.Name = strings.TrimSpace(row[nameCol])
		}
		if skuCol != -1 && skuCol < len(row) {
			item.SKU = cleanSKU(row[skuCol])
		}
		if priceCol < len(row) {
			pVal, _ := strconv.ParseFloat(cleanNumber(row[priceCol]), 64)
			if pVal <= 0 {
				pVal = 10.0 // Default price
			}
			item.Price = pVal
		} else {
			item.Price = 10.0
		}
		if costCol != -1 && costCol < len(row) {
			cVal, _ := strconv.ParseFloat(cleanNumber(row[costCol]), 64)
			item.CostPrice = cVal
		}
		if compareCol != -1 && compareCol < len(row) {
			cmpVal, _ := strconv.ParseFloat(cleanNumber(row[compareCol]), 64)
			item.ComparePrice = cmpVal
		}
		if stockCol != -1 && stockCol < len(row) {
			sVal, _ := strconv.Atoi(cleanNumber(row[stockCol]))
			item.StockQuantity = sVal
		} else {
			item.StockQuantity = 10
		}
		if minStockCol != -1 && minStockCol < len(row) {
			mVal, _ := strconv.Atoi(cleanNumber(row[minStockCol]))
			item.MinStock = mVal
		}
		if catCol != -1 && catCol < len(row) {
			item.CategoryName = strings.TrimSpace(row[catCol])
		}
		if unitCol != -1 && unitCol < len(row) {
			item.Unit = strings.TrimSpace(row[unitCol])
		}
		if descCol != -1 && descCol < len(row) {
			item.Description = strings.TrimSpace(row[descCol])
		}

		if item.Name != "" {
			items = append(items, item)
		}
	}

	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
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
	template := "Name,SKU,Price,CostPrice,StockQuantity,MinStock,Description\n" +
		"Aashirvaad Shuddh Chakki Atta 5kg,ATT-5KG,245.00,210.00,50,5,5kg Whole Wheat Flour Pack\n" +
		"Fortune Sunlite Refined Oil 1L,OIL-1L,135.00,115.00,40,5,1 Liter Pouch\n" +
		"Tata Salt Iodized 1kg,SALT-1KG,28.00,22.00,100,10,1kg Vacuum Evaporated Iodized Salt\n" +
		"Dettol Original Soap 125g,DET-125G,55.00,45.00,60,8,Antiseptic Bathing Bar\n" +
		"Maggi 2-Minute Masala Noodles 70g,MAG-70G,14.00,11.50,120,15,Instant Noodles Pack\n"

	return []byte(template)
}

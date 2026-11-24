// Package services handle business logic.
package services

import (
	"bufio"
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

// SuggestProducts returns fast autocomplete search suggestions using trigram index scans.
func (s *ProductService) SuggestProducts(ctx context.Context, query string, shopID string, limit int) ([]string, error) {
	return s.productRepo.SuggestProducts(ctx, query, shopID, limit)
}

// resolveCategoryID accepts UUID, slug, name, or auto-creates a permanent shop category on the fly.
// Ensures product creation & updates never fail with ErrInvalidCategory, and custom categories
// are permanently persisted in PostgreSQL under the merchant's shop with proper Title Case capitalization.
func (s *ProductService) resolveCategoryID(ctx context.Context, shopID string, value string) (string, error) {
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

	// 2. Try looking up in shop's custom categories or global categories by exact/slugified name
	if category, err := s.categoryRepo.FindShopCategoryByNameOrSlug(ctx, shopID, trimmed); err == nil && category != nil {
		return category.ID, nil
	}

	// 3. Try canonical category alias resolution (e.g., "Grocery & Staples" -> "kirana-grocery")
	if canonSlug, ok := utils.ResolveCanonicalCategorySlug(trimmed); ok {
		if category, err := s.categoryRepo.FindShopCategoryBySlug(ctx, shopID, canonSlug); err == nil && category != nil {
			return category.ID, nil
		}
	}

	// 4. Try slugified match directly
	slugified := reuse.Slugify(trimmed)
	if slugified != "" && slugified != strings.ToLower(trimmed) {
		if category, err := s.categoryRepo.FindShopCategoryBySlug(ctx, shopID, slugified); err == nil && category != nil {
			return category.ID, nil
		}
	}

	// 5. If category is not in DB yet, dynamically insert it as a permanent custom category for THIS shop!
	if len(trimmed) >= 2 {
		catName := utils.FormatCategoryTitle(trimmed)
		slug := reuse.Slugify(trimmed)
		if slug == "" {
			slug = fmt.Sprintf("category-%d", time.Now().UnixNano()%100000)
		}

		// Ensure slug uniqueness within this shop's scope
		if existing, err := s.categoryRepo.FindShopCategoryBySlug(ctx, shopID, slug); err == nil && existing != nil {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			slug = fmt.Sprintf("%s-%d", slug, r.Intn(9000)+1000)
		}

		newCat, err := s.categoryRepo.CreateShopCategory(ctx, shopID, &model.Category{
			Name:        catName,
			Slug:        slug,
			Icon:        "📦",
			Description: fmt.Sprintf("Category for %s", catName),
			IsActive:    true,
		})
		if err == nil && newCat != nil {
			// Broadcast WebSocket mutation so merchant frontend & customer storefront immediately sync the new category
			wsHub := GetWebSocketHub(nil)
			if wsHub != nil {
				if shop, sErr := s.shopRepo.FindByID(ctx, shopID); sErr == nil && shop != nil {
					wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "CATEGORY_MUTATED", newCat)
				}
			}
			return newCat.ID, nil
		}
	}

	// 6. Fallback to default category if all else fails
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
	categoryID, err := s.resolveCategoryID(ctx, shop.ID, input.CategoryID)
	if err != nil {
		return nil, err
	}

	sku := strings.TrimSpace(strings.ToUpper(input.SKU))
	images := input.Images

	// 3. Enforce max 10 images (Cost & Storage efficiency rule)
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

	if err == nil && created != nil {
		wsHub := GetWebSocketHub(nil)
		wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "PRODUCT_CREATED", created)

		// Auto-sync newly created product into master catalog so any other shop can instantly scan & auto-fill it
		code := strings.TrimSpace(created.SKU)
		if bc, ok := created.Attributes["barcode"].(string); ok && strings.TrimSpace(bc) != "" {
			code = strings.TrimSpace(bc)
		}
		if len(code) >= 2 {
			brand, _ := created.Attributes["brand"].(string)
			unit, _ := created.Attributes["unit"].(string)
			hsn, _ := created.Attributes["hsn_code"].(string)
			img := ""
			if len(created.Images) > 0 {
				img = created.Images[0]
			}
			go func(c *model.Product, barCode, bName, uName, hCode, iUrl string) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.productRepo.SyncToMasterCatalog(bgCtx, c, barCode, bName, uName, hCode, iUrl)
			}(created, code, brand, unit, hsn, img)
		}
	}

	return created, err
}

func (s *ProductService) GetProductByID(ctx context.Context, id string) (*model.Product, error) {
	return s.productRepo.FindByID(ctx, id)
}

// GetProductByBarcode retrieves a product by barcode/SKU within the merchant's own shop.
func (s *ProductService) GetProductByBarcode(ctx context.Context, userID, barcode string) (*model.Product, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	trimmed := strings.TrimSpace(barcode)
	if trimmed == "" {
		return nil, errors.New("barcode cannot be empty")
	}

	prod, err := s.productRepo.FindBySKU(ctx, shop.ID, trimmed)
	if err != nil {
		return nil, err
	}
	return prod, nil
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
		categoryID, err := s.resolveCategoryID(ctx, shop.ID, *input.CategoryID)
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
	if existing.Attributes == nil {
		existing.Attributes = make(map[string]interface{})
	}
	if input.Attributes != nil {
		for k, v := range *input.Attributes {
			if v == nil {
				delete(existing.Attributes, k)
			} else {
				existing.Attributes[k] = v
			}
		}
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

	if err == nil && updated != nil {
		wsHub := GetWebSocketHub(nil)
		wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "PRODUCT_MUTATED", updated)
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

	// 3. Soft delete from database (images are preserved for past bills and invoices)
	if err := s.productRepo.Delete(ctx, productID, shop.ID); err != nil {
		return err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShopAndStorefront(shop.ID, shop.Slug, "PRODUCT_DELETED", map[string]interface{}{
		"id":      productID,
		"shop_id": shop.ID,
	})

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
// FindNearbyProducts searches for in-stock products across nearby shops via strict PostGIS spatial geofencing.
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

	nearbyItems, totalCount, err := s.productRepo.FindNearbyProducts(ctx, lat, lng, radiusKm, query, category, openNow, page, limit)
	if err != nil {
		return nil, err
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

type attrPair struct {
	nameCol int
	valCol  int
}

type importColumnMap struct {
	nameCol     int
	skuCol      int
	priceCol    int
	costCol     int
	compareCol  int
	stockCol    int
	minStockCol int
	catCol      int
	unitCol     int
	descCol     int
	brandCol    int
	imageCol    int
	hsnCol      int
	gstCol      int
	attrPairs   []attrPair
}

// resolveImportColumnMap inspects the header row to dynamically locate and map product attribute columns.
func resolveImportColumnMap(header []string) *importColumnMap {
	m := &importColumnMap{
		nameCol:     -1,
		skuCol:      -1,
		priceCol:    -1,
		costCol:     -1,
		compareCol:  -1,
		stockCol:    -1,
		minStockCol: -1,
		catCol:      -1,
		unitCol:     -1,
		descCol:     -1,
		brandCol:    -1,
		imageCol:    -1,
		hsnCol:      -1,
		gstCol:      -1,
	}

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
				m.nameCol = i
			} else if m.nameCol == -1 && (strings.Contains(cleanH, "name") || strings.Contains(cleanH, "title") || strings.Contains(cleanH, "product") || strings.Contains(cleanH, "item")) &&
				!strings.Contains(cleanH, "category") && !strings.Contains(cleanH, "brand") {
				m.nameCol = i
			}
		}

		// 2. SKU / Barcode
		if !isAttr && !isBadge {
			if strings.Contains(cleanH, "sku") || strings.Contains(cleanH, "barcode") ||
				strings.Contains(cleanH, "item code") || strings.Contains(cleanH, "product code") ||
				strings.Contains(cleanH, "upc") || strings.Contains(cleanH, "ean") || cleanH == "code" {
				if m.skuCol == -1 {
					m.skuCol = i
				}
			}
		}

		// 3. Cost Price
		if isCost && !isDiscount {
			if m.costCol == -1 {
				m.costCol = i
			}
		}

		// 4. MRP / Compare Price (strictly exclude discount/margin)
		if isCompare && !isDiscount {
			if m.compareCol == -1 {
				m.compareCol = i
			}
		}

		// 5. Selling Price
		if !isCost && !isCompare && !isDiscount {
			if strings.Contains(cleanH, "selling") || strings.Contains(cleanH, "sell") ||
				strings.Contains(cleanH, "sales") || strings.Contains(cleanH, "price") ||
				strings.Contains(cleanH, "rate") || cleanH == "sp" {
				if m.priceCol == -1 {
					m.priceCol = i
				}
			}
		}

		// 6. Min Stock / Low Stock Alert
		if isAlert {
			if m.minStockCol == -1 {
				m.minStockCol = i
			}
		}

		// 7. Stock Quantity (strictly exclude alert / min / low)
		if isStock && !isAlert {
			if m.stockCol == -1 {
				m.stockCol = i
			}
		}

		// 8. Category
		if !isAttr {
			if strings.Contains(cleanH, "category") || cleanH == "cat" ||
				strings.Contains(cleanH, "group") || strings.Contains(cleanH, "department") {
				if m.catCol == -1 {
					m.catCol = i
				}
			}
		}

		// 9. Unit
		if !isAttr {
			if strings.Contains(cleanH, "unit") || strings.Contains(cleanH, "uom") ||
				cleanH == "pack" || cleanH == "packaging" {
				if m.unitCol == -1 {
					m.unitCol = i
				}
			}
		}

		// 10. Description
		if strings.Contains(cleanH, "desc") || strings.Contains(cleanH, "detail") ||
			strings.Contains(cleanH, "notes") || strings.Contains(cleanH, "about") {
			if m.descCol == -1 {
				m.descCol = i
			}
		}

		// 11. Brand
		if !isAttr {
			if strings.Contains(cleanH, "brand") || strings.Contains(cleanH, "company") ||
				strings.Contains(cleanH, "manufacturer") || strings.Contains(cleanH, "mfg") {
				if m.brandCol == -1 {
					m.brandCol = i
				}
			}
		}

		// 12. Image URL
		if strings.Contains(cleanH, "image") || strings.Contains(cleanH, "photo") || cleanH == "img" || cleanH == "pic" {
			if m.imageCol == -1 {
				m.imageCol = i
			}
		}

		// 13. HSN Code
		if strings.Contains(cleanH, "hsn") {
			if m.hsnCol == -1 {
				m.hsnCol = i
			}
		}

		// 14. GST Rate
		if strings.Contains(cleanH, "gst") || strings.Contains(cleanH, "tax") {
			if m.gstCol == -1 {
				m.gstCol = i
			}
		}

		// 15. Attribute pair detection (e.g. Attribute 1 Name, Attribute 1 Value)
		if isAttr && strings.Contains(cleanH, "name") {
			if i+1 < len(header) && strings.Contains(strings.ToLower(header[i+1]), "val") {
				m.attrPairs = append(m.attrPairs, attrPair{nameCol: i, valCol: i + 1})
			}
		}
	}

	if m.nameCol == -1 {
		m.nameCol = 0
	}
	if m.priceCol == -1 {
		m.priceCol = 1
	}

	return m
}

// parseRow extracts a single product item from an imported row using the mapped column positions.
func (m *importColumnMap) parseRow(row []string) *dto.BulkImportProductItem {
	if len(row) == 0 {
		return nil
	}

	name := ""
	if m.nameCol < len(row) {
		name = strings.TrimSpace(row[m.nameCol])
	}
	if name == "" {
		return nil
	}

	item := dto.BulkImportProductItem{
		Name: name,
	}

	if m.skuCol != -1 && m.skuCol < len(row) {
		item.SKU = cleanImportSKU(row[m.skuCol])
	}

	if m.priceCol < len(row) {
		pVal, _ := strconv.ParseFloat(cleanImportNumber(row[m.priceCol]), 64)
		if pVal <= 0 {
			pVal = 10.0 // Default price
		}
		item.Price = pVal
	} else {
		item.Price = 10.0
	}

	if m.costCol != -1 && m.costCol < len(row) {
		cVal, _ := strconv.ParseFloat(cleanImportNumber(row[m.costCol]), 64)
		item.CostPrice = cVal
	}

	if m.compareCol != -1 && m.compareCol < len(row) {
		cmpVal, _ := strconv.ParseFloat(cleanImportNumber(row[m.compareCol]), 64)
		item.ComparePrice = cmpVal
	}

	if m.stockCol != -1 && m.stockCol < len(row) {
		sVal, _ := strconv.Atoi(cleanImportNumber(row[m.stockCol]))
		item.StockQuantity = sVal
	} else {
		item.StockQuantity = 10
	}

	if m.minStockCol != -1 && m.minStockCol < len(row) {
		mVal, _ := strconv.Atoi(cleanImportNumber(row[m.minStockCol]))
		item.MinStock = mVal
	} else {
		item.MinStock = 5
	}

	if m.catCol != -1 && m.catCol < len(row) {
		item.CategoryName = strings.TrimSpace(row[m.catCol])
	}

	if m.unitCol != -1 && m.unitCol < len(row) {
		unitVal := strings.TrimSpace(row[m.unitCol])
		if unitVal != "" {
			item.Unit = unitVal
		} else {
			item.Unit = "Piece"
		}
	} else {
		item.Unit = "Piece"
	}

	baseDesc := ""
	if m.descCol != -1 && m.descCol < len(row) {
		baseDesc = strings.TrimSpace(row[m.descCol])
	}

	brandVal := ""
	if m.brandCol != -1 && m.brandCol < len(row) {
		brandVal = strings.TrimSpace(row[m.brandCol])
	}

	var extraParts []string
	if brandVal != "" && !strings.Contains(strings.ToLower(baseDesc), strings.ToLower(brandVal)) {
		extraParts = append(extraParts, fmt.Sprintf("Brand: %s", brandVal))
	}

	for _, ap := range m.attrPairs {
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

	if m.imageCol != -1 && m.imageCol < len(row) {
		item.ImageURL = strings.TrimSpace(row[m.imageCol])
	}
	if m.hsnCol != -1 && m.hsnCol < len(row) {
		item.HSNCode = strings.TrimSpace(row[m.hsnCol])
	}
	if m.gstCol != -1 && m.gstCol < len(row) {
		gVal, _ := strconv.ParseFloat(cleanImportNumber(row[m.gstCol]), 64)
		item.GSTRate = gVal
	}
	if brandVal != "" {
		item.Brand = brandVal
	}
	if item.Barcode == "" && item.SKU != "" {
		item.Barcode = item.SKU
	}

	return &item
}

// parseRecordsToImportItems extracts product items from a 2D grid of CSV/Excel cells (kept for backward compatibility).
func parseRecordsToImportItems(records [][]string) []dto.BulkImportProductItem {
	if len(records) <= 1 {
		return nil
	}

	colMap := resolveImportColumnMap(records[0])
	var items []dto.BulkImportProductItem

	for _, row := range records[1:] {
		if item := colMap.parseRow(row); item != nil {
			items = append(items, *item)
		}
	}

	return items
}

// BulkImportProductsFromCSV parses CSV stream line-by-line and bulk imports products into shop catalog.
func (s *ProductService) BulkImportProductsFromCSV(ctx context.Context, userID string, csvReader io.Reader, updateExisting bool) (*dto.BulkImportResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	bufReader := bufio.NewReader(csvReader)

	// Peek first 512 bytes for UTF-8 BOM and delimiter detection without loading whole file in RAM
	peekBytes, _ := bufReader.Peek(512)

	// Strip UTF-8 BOM if present
	if bytes.HasPrefix(peekBytes, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = bufReader.Discard(3)
		peekBytes = peekBytes[3:]
	}

	// Delimiter detection: check for semicolon or tab if comma isn't prominent
	commaCount := bytes.Count(peekBytes, []byte(","))
	semiCount := bytes.Count(peekBytes, []byte(";"))
	tabCount := bytes.Count(peekBytes, []byte("\t"))

	var delimiter rune = ','
	if semiCount > commaCount && semiCount > tabCount {
		delimiter = ';'
	} else if tabCount > commaCount && tabCount > semiCount {
		delimiter = '\t'
	}

	reader := csv.NewReader(bufReader)
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // flexible column count per row

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("CSV file is empty or missing data rows")
	}

	colMap := resolveImportColumnMap(header)

	// Stream line-by-line without buffering entire dataset in memory
	var items []dto.BulkImportProductItem
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed individual row and continue reading
			continue
		}
		if item := colMap.parseRow(row); item != nil {
			items = append(items, *item)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("no valid product rows found in CSV")
	}

	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
}

// MapProductToCustomerDTO converts a DB Product model into a public, clean CustomerProductResponseDTO.
func MapProductToCustomerDTO(p *model.Product) *dto.CustomerProductResponseDTO {
	if p == nil {
		return nil
	}

	isInStock := p.StockQuantity > 0
	stockBadge := "Out of Stock"
	if p.StockQuantity > 5 {
		stockBadge = "In Stock"
	} else if p.StockQuantity > 0 {
		stockBadge = "Only Few Left"
	}

	var discountPct float64
	if p.ComparePrice > p.Price && p.ComparePrice > 0 {
		discountPct = math.Round(((p.ComparePrice - p.Price) / p.ComparePrice) * 100)
	}

	return &dto.CustomerProductResponseDTO{
		ID:                 p.ID,
		ShopID:             p.ShopID,
		Name:               p.Name,
		Slug:               p.Slug,
		Description:        p.Description,
		SellingPrice:       p.Price,
		ComparePrice:       p.ComparePrice,
		DiscountPercentage: discountPct,
		MRP:                p.MRP,
		CategoryID:         p.CategoryID,
		Images:             p.Images,
		IsInStock:          isInStock,
		StockBadge:         stockBadge,
		Rating:             4.8, // Default rating
		ReviewCount:        12,  // Default count
		Attributes:         p.Attributes,
		CreatedAt:          p.CreatedAt,
	}
}

// MapProductToMerchantDTO converts a DB Product model into a rich MerchantProductResponseDTO.
func MapProductToMerchantDTO(p *model.Product) *dto.MerchantProductResponseDTO {
	if p == nil {
		return nil
	}

	marginAmt := p.Price - p.CostPrice
	var marginPct float64
	if p.Price > 0 {
		marginPct = math.Round((marginAmt / p.Price) * 10000) / 100
	}

	isLowStock := p.StockQuantity <= p.MinStock

	return &dto.MerchantProductResponseDTO{
		ID:                  p.ID,
		ShopID:              p.ShopID,
		Name:                p.Name,
		Slug:                p.Slug,
		Description:         p.Description,
		SKU:                 p.SKU,
		CostPrice:           p.CostPrice,
		SellingPrice:        p.Price,
		ComparePrice:        p.ComparePrice,
		MRP:                 p.MRP,
		ProfitMarginAmount:  marginAmt,
		ProfitMarginPercent: marginPct,
		CategoryID:          p.CategoryID,
		ExactStockQuantity:  p.StockQuantity,
		MinStock:            p.MinStock,
		LowStockThreshold:   p.LowStockThreshold,
		IsLowStock:          isLowStock,
		Images:              p.Images,
		Weight:              p.Weight,
		IsActive:            p.IsActive,
		IsFeatured:          p.IsFeatured,
		Tags:                p.Tags,
		Attributes:          p.Attributes,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

// ListCustomerProducts retrieves public products formatted with CustomerProductResponseDTO.
func (s *ProductService) ListCustomerProducts(ctx context.Context, filter dto.ProductFilter) (*dto.CustomerProductPaginationResponse, error) {
	products, totalCount, err := s.productRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	customerProducts := make([]*dto.CustomerProductResponseDTO, 0, len(products))
	for _, p := range products {
		if p.IsActive {
			customerProducts = append(customerProducts, MapProductToCustomerDTO(p))
		}
	}

	return &dto.CustomerProductPaginationResponse{
		Products:   customerProducts,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// ListMerchantProducts retrieves full shop inventory formatted with MerchantProductResponseDTO.
func (s *ProductService) ListMerchantProducts(ctx context.Context, userID string, filter dto.ProductFilter) (*dto.MerchantProductPaginationResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopNotFound
	}

	filter.ShopID = shop.ID
	products, totalCount, err := s.productRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

	merchantProducts := make([]*dto.MerchantProductResponseDTO, 0, len(products))
	for _, p := range products {
		merchantProducts = append(merchantProducts, MapProductToMerchantDTO(p))
	}

	return &dto.MerchantProductPaginationResponse{
		Products:   merchantProducts,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}


// BulkImportProductsFromExcel parses Excel (.xlsx) data with streaming row iterator.
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

	// Read rows streaming from the first sheet
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("Excel sheet is empty or contains no product rows")
	}

	header, err := rows.Columns()
	if err != nil || len(header) == 0 {
		return nil, errors.New("Excel sheet missing header row")
	}

	colMap := resolveImportColumnMap(header)
	var items []dto.BulkImportProductItem

	for rows.Next() {
		row, err := rows.Columns()
		if err != nil || len(row) == 0 {
			continue
		}
		if item := colMap.parseRow(row); item != nil {
			items = append(items, *item)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("no valid product rows found in Excel sheet")
	}

	return s.productRepo.BulkImportProducts(ctx, shop.ID, items, updateExisting)
}

// BulkImportProductsFromFile automatically detects CSV vs Excel format and imports products via streaming readers.
func (s *ProductService) BulkImportProductsFromFile(ctx context.Context, userID string, filename string, fileReader io.Reader, updateExisting bool) (*dto.BulkImportResponse, error) {
	bufReader := bufio.NewReader(fileReader)

	// Peek first 512 bytes for magic bytes (e.g. PK\x03\x04 zip header) without consuming
	peekBytes, _ := bufReader.Peek(512)

	lowerName := strings.ToLower(filename)
	isExcel := strings.HasSuffix(lowerName, ".xlsx") ||
		strings.HasSuffix(lowerName, ".xlsm") ||
		strings.HasSuffix(lowerName, ".xls") ||
		bytes.HasPrefix(peekBytes, []byte("PK\x03\x04")) ||
		bytes.HasPrefix(peekBytes, []byte("PK\x05\x06")) ||
		bytes.HasPrefix(peekBytes, []byte("PK\x07\x08"))

	if isExcel {
		return s.BulkImportProductsFromExcel(ctx, userID, bufReader, updateExisting)
	}

	return s.BulkImportProductsFromCSV(ctx, userID, bufReader, updateExisting)
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
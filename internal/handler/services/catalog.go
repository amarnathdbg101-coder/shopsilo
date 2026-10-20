package services

import (
	"context"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

type CatalogService struct {
	repo *repository.CatalogRepo
}

func NewCatalogService(repo *repository.CatalogRepo) *CatalogService {
	return &CatalogService{repo: repo}
}

// Lookup queries the master catalog by barcode.
// 1. Searches PostgreSQL master_catalog_products (fast sub-millisecond B-Tree index).
// 2. If not found, falls back to server-side GS1 India manufacturer rules.
func (s *CatalogService) Lookup(ctx context.Context, rawBarcode string) (*dto.CatalogLookupResponse, error) {
	clean := strings.TrimSpace(rawBarcode)
	if clean == "" {
		return nil, repository.ErrCatalogItemNotFound
	}

	// 1. Check PostgreSQL Database
	item, err := s.repo.FindByBarcode(ctx, clean)
	if err == nil && item != nil {
		return &dto.CatalogLookupResponse{
			Barcode:      item.Barcode,
			Name:         item.Name,
			Brand:        item.Brand,
			Category:     item.Category,
			Unit:         item.Unit,
			MRP:          item.MRP,
			SellingPrice: item.SellingPrice,
			Description:  item.Description,
			ImageURL:     item.ImageURL,
			Attributes:   item.Attributes,
			Source:       item.Source,
			ScanCount:    item.ScanCount,
			TrustScore:   item.TrustScore,
			IsFound:      true,
		}, nil
	}

	// 2. Server-Side GS1 India Manufacturer Heuristic Fallback
	stripped := strings.TrimLeft(clean, "0")
	if stripped == "" {
		stripped = clean
	}

	for prefix, rule := range serverPrefixRules {
		if strings.HasPrefix(clean, prefix) || strings.HasPrefix(stripped, prefix) {
			return &dto.CatalogLookupResponse{
				Barcode:      clean,
				Name:         rule.BrandGroup + " Pack",
				Brand:        strings.Split(rule.BrandGroup, " ")[0],
				Category:     rule.Category,
				Unit:         rule.Unit,
				MRP:          rule.MRP,
				SellingPrice: rule.SellingPrice,
				Description:  rule.BrandGroup + " packaged product",
				Source:       "gs1_rule",
				IsFound:      true,
			}, nil
		}
	}

	// 3. Universal 890 GS1 India Retail Fallback
	if strings.HasPrefix(clean, "890") || strings.HasPrefix(stripped, "890") {
		return &dto.CatalogLookupResponse{
			Barcode:      clean,
			Name:         "Indian Retail Pack (" + clean + ")",
			Brand:        "Indian Retail",
			Category:     "General Store",
			Unit:         "1 pc",
			MRP:          99.0,
			SellingPrice: 95.0,
			Description:  "Packaged retail item manufactured in India",
			Source:       "gs1_890",
			IsFound:      true,
		}, nil
	}

	return nil, repository.ErrCatalogItemNotFound
}

// EnrichFromProduct is the Crowdsourcing Hook:
// When any merchant creates or saves a product with a valid barcode and name,
// this silently records/updates it into the global master catalog so all merchants benefit.
func (s *CatalogService) EnrichFromProduct(ctx context.Context, barcode, name, brand, category, unit string, mrp, sellingPrice float64, description, imageURL string, attributes map[string]interface{}) {
	code := strings.TrimSpace(barcode)
	cleanName := strings.TrimSpace(name)
	if code == "" || cleanName == "" || len(code) < 6 {
		return
	}

	// Background write to PostgreSQL
	p := &model.MasterCatalogProduct{
		Barcode:      code,
		Name:         cleanName,
		Brand:        strings.TrimSpace(brand),
		Category:     category,
		Unit:         unit,
		MRP:          mrp,
		SellingPrice: sellingPrice,
		Description:  description,
		ImageURL:     imageURL,
		Attributes:   attributes,
		Source:       "crowdsourced",
	}

	_ = s.repo.Upsert(ctx, p)
}

// BulkSeed inserts an initial list of standard FMCG products.
func (s *CatalogService) BulkSeed(ctx context.Context, items []*model.MasterCatalogProduct) (int, error) {
	return s.repo.BulkUpsert(ctx, items)
}

func (s *CatalogService) Search(ctx context.Context, query string, limit int) ([]*dto.CatalogLookupResponse, error) {
	items, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.CatalogLookupResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, &dto.CatalogLookupResponse{
			Barcode:      item.Barcode,
			Name:         item.Name,
			Brand:        item.Brand,
			Category:     item.Category,
			Unit:         item.Unit,
			MRP:          item.MRP,
			SellingPrice: item.SellingPrice,
			Description:  item.Description,
			ImageURL:     item.ImageURL,
			Attributes:   item.Attributes,
			Source:       item.Source,
			ScanCount:    item.ScanCount,
			TrustScore:   item.TrustScore,
			IsFound:      true,
		})
	}
	return responses, nil
}

type prefixRule struct {
	BrandGroup   string
	Category     string
	Unit         string
	MRP          float64
	SellingPrice float64
}

var serverPrefixRules = map[string]prefixRule{
	// Vini Cosmetics / Fogg
	"8904238": {"Fogg Scent / Perfume (Vini Cosmetics)", "Personal Care & Beauty", "100ml", 499.0, 449.0},
	"8908001": {"Fogg Body Spray / Deodorant (Vini Cosmetics)", "Personal Care & Beauty", "150ml", 250.0, 230.0},
	// HUL
	"8901030": {"Hindustan Unilever (Dove / Surf Excel / Lifebuoy / Vim)", "Personal Care & Beauty", "1 pc", 185.0, 175.0},
	// Nestlé
	"8901058": {"Nestlé India (Maggi / KitKat / Nescafé)", "Grocery & Staples", "1 pc", 14.0, 14.0},
	// Parle
	"8901499": {"Parle Products (Parle-G / Monaco / Hide & Seek)", "Snacks & Munchies", "1 pc", 25.0, 24.0},
	// Britannia
	"8901719": {"Britannia (Good Day / Marie Gold / Milk Bikis)", "Snacks & Munchies", "1 pc", 30.0, 28.0},
	// Tata Consumer
	"8901063": {"Tata Consumer (Tata Salt / Tata Tea / Sampann)", "Grocery & Staples", "1kg", 28.0, 26.0},
	// ITC
	"8906007": {"ITC Limited (Aashirvaad / Sunfeast / Bingo / Engage)", "Grocery & Staples", "1 pc", 60.0, 55.0},
	// Amul
	"8906010": {"Amul / GCMMF (Butter / Milk / Ghee / Cheese)", "Dairy, Bakery & Frozen", "500g", 60.0, 58.0},
	// Dabur
	"8901207": {"Dabur India (Dabur Red / Honey / Real / Amla)", "Personal Care & Beauty", "100ml", 75.0, 70.0},
	// Marico
	"8901233": {"Marico (Parachute / Saffola)", "Grocery & Staples", "1L", 160.0, 150.0},
	// Colgate
	"8901023": {"Colgate-Palmolive (Colgate Strong Teeth / MaxFresh)", "Personal Care & Beauty", "100g", 65.0, 60.0},
	// Haldiram's
	"8901725": {"Haldiram's (Bhujia / Namkeen / Sweets)", "Snacks & Munchies", "200g", 45.0, 42.0},
	// Cadbury / Mondelēz
	"8901491": {"Cadbury / Mondelēz (Dairy Milk / Oreo / Bournvita)", "Snacks & Munchies", "1 pc", 40.0, 40.0},
	// Fortune
	"8901088": {"Fortune / Adani Wilmar (Edible Oils / Atta)", "Grocery & Staples", "1L", 145.0, 135.0},
	// Reckitt Benckiser
	"8901052": {"Reckitt Benckiser (Dettol / Harpic / Lizol)", "Household & Cleaning", "250ml", 85.0, 80.0},
	// PepsiCo
	"8901331": {"PepsiCo India (Lay's / Kurkure / Doritos / Sting)", "Snacks & Munchies", "50g", 20.0, 20.0},
	// Coca-Cola
	"8901764": {"Coca-Cola India (Thums Up / Sprite / Maaza)", "Beverages & Drinks", "750ml", 40.0, 40.0},
	// P&G
	"8901037": {"Procter & Gamble (Ariel / Tide / Whisper / Pantene)", "Household & Cleaning", "500g", 120.0, 110.0},
	// Wild Stone
	"8906009": {"Wild Stone (McNROE)", "Personal Care & Beauty", "150ml", 225.0, 210.0},
	// Denver
	"8906024": {"Denver / Envy (Vanesa Care)", "Personal Care & Beauty", "165ml", 240.0, 220.0},
	// Park Avenue
	"8901277": {"Park Avenue Fragrance", "Personal Care & Beauty", "150ml", 250.0, 225.0},
	// Patanjali
	"8906019": {"Patanjali Ayurved (Dant Kanti / Ghee / Honey)", "Grocery & Staples", "200g", 95.0, 90.0},
	// Everest
	"8901362": {"Everest Spices (Everest Masala)", "Grocery & Staples", "100g", 65.0, 62.0},
	// MDH
	"8901312": {"MDH Spices (MDH Masala)", "Grocery & Staples", "100g", 75.0, 72.0},
}

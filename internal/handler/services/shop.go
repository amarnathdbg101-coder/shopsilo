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
	"shopMe/internal/utils"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type ShopService struct {
	shopRepo         *repository.ShopRepo
	userRepo         *repository.UserRepo
	modRepo          *repository.ModerationRepo
	categoryRepo     *repository.CategoryRepo
	productRepo      *repository.ProductRepo
	posRepo          *repository.POSRepo
	loyaltyRepo      *repository.LoyaltyRepo
	catMu            sync.RWMutex
	cachedCategories []*model.Category
	categoriesExpiry time.Time
}

func NewShopService(shopRepo *repository.ShopRepo, userRepo *repository.UserRepo, modRepo *repository.ModerationRepo) *ShopService {
	return &ShopService{
		shopRepo: shopRepo,
		userRepo: userRepo,
		modRepo:  modRepo,
	}
}

// SetAggregatedRepos injects catalog, POS, and loyalty repositories to power unified batch endpoints.
func (s *ShopService) SetAggregatedRepos(
	categoryRepo *repository.CategoryRepo,
	productRepo *repository.ProductRepo,
	posRepo *repository.POSRepo,
	loyaltyRepo *repository.LoyaltyRepo,
) {
	s.categoryRepo = categoryRepo
	s.productRepo = productRepo
	s.posRepo = posRepo
	s.loyaltyRepo = loyaltyRepo
}

// enrichShop adds convenient URLs for in-store physical visits (Google Maps navigation & WhatsApp inquiry).
func enrichShop(s *model.Shop) {
	if s == nil {
		return
	}

	// 1. Google Maps URL
	if s.Latitude != nil && s.Longitude != nil {
		s.GoogleMapsURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%f,%f", *s.Latitude, *s.Longitude)
	} else if s.Address != "" {
		s.GoogleMapsURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", strings.ReplaceAll(s.Address, " ", "+"))
	}

	// 2. WhatsApp URL
	targetPhone := s.WhatsAppNumber
	if targetPhone == "" {
		targetPhone = s.Phone
	}
	if targetPhone != "" {
		var digits strings.Builder
		for _, r := range targetPhone {
			if r >= '0' && r <= '9' {
				digits.WriteRune(r)
			}
		}
		num := digits.String()
		if num != "" {
			greeting := fmt.Sprintf("Hello %s, I found your store on ShopMe!", s.Name)
			s.WhatsAppURL = fmt.Sprintf("https://wa.me/%s?text=%s", num, strings.ReplaceAll(greeting, " ", "%20"))
		}
	}
}

func (s *ShopService) CreateShop(
	ctx context.Context,
	userID, userEmail string,
	clientIP, userAgent, deviceFingerprint string,
	input dto.CreateShopRequest,
) (*dto.ShopResponse, error) {
	// 0. Unrestricted shop creation: Device, IP, or phone ban restriction removed per user request

	// 1. Verify user doesn't already own a shop
	existingShop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err == nil && existingShop != nil {
		return nil, ErrUserAlreadyHasShop
	}

	// 2. Generate slug
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = reuse.Slugify(input.Name)
	} else {
		slug = reuse.Slugify(slug)
	}

	// Check slug uniqueness; if taken, append short suffix
	if _, err := s.shopRepo.FindBySlug(ctx, slug); err == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		slug = fmt.Sprintf("%s-%d", slug, r.Intn(9000)+1000)
	}

	// 3. Begin database transaction
	tx, err := s.shopRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start database transaction")
	}
	defer tx.Rollback(ctx)

	// 4. Create shop record
	shopLat := input.Latitude
	shopLng := input.Longitude
	if shopLat == nil || shopLng == nil || !reuse.IsValidIndiaCoordinates(*shopLat, *shopLng) {
		defaultLat := 26.1542
		defaultLng := 85.8918
		shopLat = &defaultLat
		shopLng = &defaultLng
	}

	shopToCreate := &model.Shop{
		UserID:            userID,
		Name:              strings.TrimSpace(input.Name),
		Slug:              slug,
		Description:       strings.TrimSpace(input.Description),
		Category:          strings.TrimSpace(input.Category),
		Phone:             strings.TrimSpace(input.Phone),
		WhatsAppNumber:    strings.TrimSpace(input.WhatsAppNumber),
		Address:           strings.TrimSpace(input.Address),
		City:              strings.TrimSpace(input.City),
		Pincode:           strings.TrimSpace(input.Pincode),
		Latitude:          shopLat,
		Longitude:         shopLng,
		LogoURL:           strings.TrimSpace(input.LogoURL),
		Banners:           input.Banners,
		Timing:            strings.TrimSpace(input.Timing),
		OpeningTime:       strings.TrimSpace(input.OpeningTime),
		ClosingTime:       strings.TrimSpace(input.ClosingTime),
		WeeklyOff:         strings.TrimSpace(input.WeeklyOff),
		IsOpen:            true,
		IsActive:          true,
		Status:            model.ShopStatusActive,
		CreationIP:        clientIP,
		CreationUserAgent: userAgent,
		DeviceFingerprint: deviceFingerprint,
	}

	createdShop, err := s.shopRepo.CreateWithTx(ctx, tx, shopToCreate)
	if err != nil {
		if errors.Is(err, repository.ErrShopExists) {
			return nil, ErrUserAlreadyHasShop
		}
		if errors.Is(err, repository.ErrSlugTaken) {
			return nil, ErrSlugAlreadyTaken
		}
		return nil, err
	}

	// 5. Update user role to "shop"
	if err := s.userRepo.UpdateRoleWithTx(ctx, tx, userID, "shop"); err != nil {
		return nil, errors.New("failed to upgrade user role to shop")
	}

	// 6. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit shop creation")
	}

	enrichShop(createdShop)

	// 7. Generate upgraded JWT token with role "shop"
	token, err := reuse.GenerateJwt(userID, userEmail, "shop")
	if err != nil {
		return &dto.ShopResponse{Shop: createdShop}, nil
	}

	return &dto.ShopResponse{
		Shop:        createdShop,
		AccessToken: token,
	}, nil
}

func (s *ShopService) GetMyShop(ctx context.Context, userID string) (*model.Shop, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}
	enrichShop(shop)
	return shop, nil
}

func (s *ShopService) GetShopByID(ctx context.Context, id string) (*model.Shop, error) {
	shop, err := s.shopRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}
	enrichShop(shop)
	return shop, nil
}

func (s *ShopService) GetShopBySlug(ctx context.Context, slug string) (*model.Shop, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}
	enrichShop(shop)
	return shop, nil
}

func (s *ShopService) ListShops(ctx context.Context, filter dto.ShopFilter) (*dto.ShopPaginationResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}

	shops, totalCount, err := s.shopRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	if shops == nil {
		shops = []*model.Shop{}
	}

	for _, sh := range shops {
		enrichShop(sh)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ShopPaginationResponse{
		Shops:      shops,
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ShopService) UpdateMyShop(ctx context.Context, userID string, input dto.UpdateShopRequest) (*model.Shop, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		shop.Name = strings.TrimSpace(*input.Name)
	}
	if input.Slug != nil {
		shop.Slug = reuse.Slugify(*input.Slug)
	}
	if input.Description != nil {
		shop.Description = strings.TrimSpace(*input.Description)
	}
	if input.Category != nil {
		shop.Category = strings.TrimSpace(*input.Category)
	}
	if input.Phone != nil {
		shop.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.WhatsAppNumber != nil {
		shop.WhatsAppNumber = strings.TrimSpace(*input.WhatsAppNumber)
	}
	if input.Address != nil {
		shop.Address = strings.TrimSpace(*input.Address)
	}
	if input.City != nil {
		shop.City = strings.TrimSpace(*input.City)
	}
	if input.Pincode != nil {
		shop.Pincode = strings.TrimSpace(*input.Pincode)
	}
	if input.Latitude != nil && input.Longitude != nil {
		if reuse.IsValidIndiaCoordinates(*input.Latitude, *input.Longitude) {
			shop.Latitude = input.Latitude
			shop.Longitude = input.Longitude
		}
	} else if input.Latitude != nil {
		if shop.Longitude != nil && reuse.IsValidIndiaCoordinates(*input.Latitude, *shop.Longitude) {
			shop.Latitude = input.Latitude
		}
	} else if input.Longitude != nil {
		if shop.Latitude != nil && reuse.IsValidIndiaCoordinates(*shop.Latitude, *input.Longitude) {
			shop.Longitude = input.Longitude
		}
	}
	if input.LogoURL != nil {
		shop.LogoURL = strings.TrimSpace(*input.LogoURL)
	}
	if input.Banners != nil {
		shop.Banners = *input.Banners
	}
	if input.Timing != nil {
		shop.Timing = strings.TrimSpace(*input.Timing)
	}
	if input.OpeningTime != nil {
		shop.OpeningTime = strings.TrimSpace(*input.OpeningTime)
	}
	if input.ClosingTime != nil {
		shop.ClosingTime = strings.TrimSpace(*input.ClosingTime)
	}
	if input.WeeklyOff != nil {
		shop.WeeklyOff = strings.TrimSpace(*input.WeeklyOff)
	}
	if input.IsOpen != nil {
		shop.IsOpen = *input.IsOpen
	}

	updated, err := s.shopRepo.Update(ctx, shop)
	if err != nil {
		if errors.Is(err, repository.ErrSlugTaken) {
			return nil, ErrSlugAlreadyTaken
		}
		return nil, err
	}

	enrichShop(updated)
	return updated, nil
}

func (s *ShopService) ToggleShopStatus(ctx context.Context, userID string, isOpen bool) (*model.Shop, error) {
	shop, err := s.shopRepo.ToggleStatus(ctx, userID, isOpen)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}
	enrichShop(shop)
	return shop, nil
}

func (s *ShopService) DeleteMyShop(ctx context.Context, userID, userEmail string) (string, error) {
	_, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return "", ErrShopNotFound
		}
		return "", err
	}

	tx, err := s.shopRepo.BeginTx(ctx)
	if err != nil {
		return "", errors.New("failed to start database transaction")
	}
	defer tx.Rollback(ctx)

	// 1. Delete shop
	if err := s.shopRepo.DeleteWithTx(ctx, tx, userID); err != nil {
		return "", err
	}

	// 2. Revert user role back to "customer"
	if err := s.userRepo.UpdateRoleWithTx(ctx, tx, userID, "customer"); err != nil {
		return "", errors.New("failed to revert user role to customer")
	}

	// 3. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return "", errors.New("failed to commit shop deletion")
	}

	// 4. Generate new token with reverted role "customer"
	token, err := reuse.GenerateJwt(userID, userEmail, "customer")
	if err != nil {
		return "", nil
	}

	return token, nil
}

// GenerateShopQRCode generates a scannable PNG QR code for the shop by slug.
func (s *ShopService) GenerateShopQRCode(ctx context.Context, slug string) ([]byte, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	// In production, this can point to the frontend web app or deep-link app URL
	targetURL := fmt.Sprintf("https://shopme.app/shops/%s", shop.Slug)
	return utils.GenerateQRCodePNG(targetURL, 300)
}

// GenerateMyShopQRCode generates a scannable PNG QR code for the authenticated shop owner.
func (s *ShopService) GenerateMyShopQRCode(ctx context.Context, userID string) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	targetURL := fmt.Sprintf("https://shopme.app/shops/%s", shop.Slug)
	return utils.GenerateQRCodePNG(targetURL, 300)
}

// GetDailyDigest retrieves today's key retail digest numbers for the shop dashboard.
func (s *ShopService) GetDailyDigest(ctx context.Context, shopOwnerUserID string) (*model.ShopDailyDigest, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.shopRepo.GetShopDailyDigest(ctx, shop.ID)
}

// GetMerchantDashboard aggregates shop profile, today's retail digest, weekly scorecard, and active promotions
// in a single concurrent query to minimize mobile network roundtrips.
func (s *ShopService) GetMerchantDashboard(ctx context.Context, shopOwnerUserID string) (*dto.MerchantDashboardResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}
	enrichShop(shop)

	resp := &dto.MerchantDashboardResponse{
		Shop: shop,
	}

	var g errgroup.Group

	// 1. Fetch Daily Digest
	g.Go(func() error {
		digest, err := s.shopRepo.GetShopDailyDigest(ctx, shop.ID)
		if err != nil {
			return err
		}
		resp.Digest = digest
		return nil
	})

	// 2. Fetch Weekly Scorecard (if posRepo is injected)
	if s.posRepo != nil {
		g.Go(func() error {
			sc, err := s.posRepo.GetWeeklyScorecardData(ctx, shop.ID)
			if err != nil {
				return nil
			}
			sc.ShopName = shop.Name
			resp.WeeklyScorecard = sc
			return nil
		})
	}

	// 3. Fetch Active Offers Count (if loyaltyRepo is injected)
	if s.loyaltyRepo != nil {
		g.Go(func() error {
			offers, err := s.loyaltyRepo.ListOffers(ctx, shop.ID, 0)
			if err == nil {
				resp.ActiveOffersCount = len(offers)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *ShopService) getCachedCategories(ctx context.Context) []*model.Category {
	s.catMu.RLock()
	if len(s.cachedCategories) > 0 && time.Now().Before(s.categoriesExpiry) {
		cats := s.cachedCategories
		s.catMu.RUnlock()
		return cats
	}
	s.catMu.RUnlock()

	if s.categoryRepo == nil {
		return []*model.Category{}
	}

	cats, err := s.categoryRepo.FindAll(ctx)
	if err != nil || cats == nil {
		return []*model.Category{}
	}

	s.catMu.Lock()
	s.cachedCategories = cats
	s.categoriesExpiry = time.Now().Add(5 * time.Minute)
	s.catMu.Unlock()

	return cats
}

// GetHomeFeed returns in a single consolidated roundtrip the data needed for customer explore screen:
// categories, nearby local shops, and trending catalog products.
func (s *ShopService) GetHomeFeed(ctx context.Context, lat, lng *float64, city string, limitShops, limitProducts int) (*dto.HomeFeedResponse, error) {
	if limitShops <= 0 || limitShops > 20 {
		limitShops = 6
	}
	if limitProducts <= 0 || limitProducts > 50 {
		limitProducts = 20
	}

	resp := &dto.HomeFeedResponse{
		Categories: make([]*model.Category, 0),
		Shops:      make([]*model.Shop, 0),
		Products:   make([]*model.Product, 0),
	}

	var g errgroup.Group

	// 1. Categories (Thread-safe 5-min TTL Memory Cache)
	g.Go(func() error {
		resp.Categories = s.getCachedCategories(ctx)
		return nil
	})

	// 2. Nearby Shops
	g.Go(func() error {
		filter := dto.ShopFilter{
			Lat:   lat,
			Lng:   lng,
			City:  city,
			Limit: limitShops,
			Page:  1,
		}
		shops, _, err := s.shopRepo.FindAll(ctx, filter)
		if err != nil {
			return nil
		}
		for _, sh := range shops {
			enrichShop(sh)
		}
		resp.Shops = shops
		return nil
	})

	// 3. Trending Products
	if s.productRepo != nil {
		g.Go(func() error {
			pFilter := dto.ProductFilter{
				Limit: limitProducts,
				Page:  1,
			}
			products, _, err := s.productRepo.FindAll(ctx, pFilter)
			if err != nil {
				return nil
			}
			resp.Products = products
			return nil
		})
	}

	_ = g.Wait()

	return resp, nil
}


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
	"time"
)

type ShopService struct {
	shopRepo *repository.ShopRepo
	userRepo *repository.UserRepo
}

func NewShopService(shopRepo *repository.ShopRepo, userRepo *repository.UserRepo) *ShopService {
	return &ShopService{
		shopRepo: shopRepo,
		userRepo: userRepo,
	}
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

func (s *ShopService) CreateShop(ctx context.Context, userID, userEmail string, input dto.CreateShopRequest) (*dto.ShopResponse, error) {
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
	if shopLat == nil || shopLng == nil || (*shopLat == 0 && *shopLng == 0) {
		defaultLat := 26.1542
		defaultLng := 85.8918
		shopLat = &defaultLat
		shopLng = &defaultLng
	}

	shopToCreate := &model.Shop{
		UserID:         userID,
		Name:           strings.TrimSpace(input.Name),
		Slug:           slug,
		Description:    strings.TrimSpace(input.Description),
		Category:       strings.TrimSpace(input.Category),
		Phone:          strings.TrimSpace(input.Phone),
		WhatsAppNumber: strings.TrimSpace(input.WhatsAppNumber),
		Address:        strings.TrimSpace(input.Address),
		City:           strings.TrimSpace(input.City),
		Pincode:        strings.TrimSpace(input.Pincode),
		Latitude:       shopLat,
		Longitude:      shopLng,
		LogoURL:        strings.TrimSpace(input.LogoURL),
		Banners:        input.Banners,
		Timing:         strings.TrimSpace(input.Timing),
		OpeningTime:    strings.TrimSpace(input.OpeningTime),
		ClosingTime:    strings.TrimSpace(input.ClosingTime),
		WeeklyOff:      strings.TrimSpace(input.WeeklyOff),
		IsOpen:         true,
		IsActive:       true,
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
	if input.Latitude != nil {
		shop.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		shop.Longitude = input.Longitude
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

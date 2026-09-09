// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"sync"
)
type UploadService struct {
	userRepo *repository.UserRepo
	shopRepo *repository.ShopRepo
	modRepo  *repository.ModerationRepo
}

func NewUploadService(userRepo *repository.UserRepo, shopRepo *repository.ShopRepo, modRepo *repository.ModerationRepo) *UploadService {
	return &UploadService{
		userRepo: userRepo,
		shopRepo: shopRepo,
		modRepo:  modRepo,
	}
}

// validateImageSafety checks if the image matches any banned perceptual hashes
func (s *UploadService) validateImageSafety(ctx context.Context, file multipart.File) error {
	if s.modRepo == nil {
		return nil
	}

	hash, err := reuse.ComputeDHash(file)
	_, _ = file.Seek(0, io.SeekStart)
	if err != nil || hash == "" {
		return nil
	}

	bannedHashes, err := s.modRepo.GetBannedImageHashes(ctx)
	if err != nil || len(bannedHashes) == 0 {
		return nil
	}

	for _, bh := range bannedHashes {
		if reuse.IsImageHashSimilar(hash, bh, reuse.MaxAllowedHammingDistance) {
			return ErrImageFlaggedBanned
		}
	}

	return nil
}

// UploadUserAvatar handles uploading a single user profile picture and deletes previous avatar from R2
func (s *UploadService) UploadUserAvatar(
	ctx context.Context,
	userID string,
	file multipart.File,
	header *multipart.FileHeader,
) (string, error) {
	if err := s.validateImageSafety(ctx, file); err != nil {
		return "", err
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", errors.New("user not found")
	}

	folder := fmt.Sprintf("users/%s", userID)
	newURL, err := reuse.UploadImage(file, header, folder)
	if err != nil {
		return "", err
	}

	// Update in database
	if err := s.userRepo.UpdateAvatar(ctx, userID, newURL); err != nil {
		_ = reuse.DeleteImage(newURL) // rollback uploaded image if DB update fails
		return "", errors.New("failed to update user avatar in database")
	}

	// Clean up previous avatar from R2 storage
	if user.AvatarURL != "" && user.AvatarURL != newURL {
		_ = reuse.DeleteImage(user.AvatarURL)
	}

	return newURL, nil
}

// UploadShopImages handles uploading 1 Logo and up to 2 Promotional Banners for a shop
func (s *UploadService) UploadShopImages(
	ctx context.Context,
	userID string,
	logoFile multipart.File,
	logoHeader *multipart.FileHeader,
	bannerFiles []multipart.File,
	bannerHeaders []*multipart.FileHeader,
) (string, []string, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return "", nil, errors.New("shop not found")
	}

	if len(bannerFiles) > 2 {
		return "", nil, ErrTooManyBannerImages
	}

	var newLogoURL string
	var newBanners []string

	folder := fmt.Sprintf("shops/%s", shop.ID)

	// 1. Upload logo if provided
	if logoFile != nil && logoHeader != nil {
		if err := s.validateImageSafety(ctx, logoFile); err != nil {
			return "", nil, err
		}

		uploadedLogo, err := reuse.UploadImage(logoFile, logoHeader, folder+"/logo")
		if err != nil {
			return "", nil, err
		}
		newLogoURL = uploadedLogo

		// Clean old logo from R2
		if shop.LogoURL != "" {
			_ = reuse.DeleteImage(shop.LogoURL)
		}
		shop.LogoURL = newLogoURL
	} else {
		newLogoURL = shop.LogoURL
	}

	// 2. Upload banners if provided
	if len(bannerFiles) > 0 {
		for i, bFile := range bannerFiles {
			if err := s.validateImageSafety(ctx, bFile); err != nil {
				return "", nil, err
			}

			bHeader := bannerHeaders[i]
			bURL, err := reuse.UploadImage(bFile, bHeader, folder+"/promotions")
			if err != nil {
				// Clean any already uploaded in this batch
				for _, uploaded := range newBanners {
					_ = reuse.DeleteImage(uploaded)
				}
				return "", nil, err
			}
			newBanners = append(newBanners, bURL)
		}

		// Clean old promotional banners from R2
		for _, oldBanner := range shop.Banners {
			_ = reuse.DeleteImage(oldBanner)
		}
		shop.Banners = newBanners
	} else {
		newBanners = shop.Banners
	}

	// 3. Save changes in DB
	if _, err := s.shopRepo.Update(ctx, shop); err != nil {
		return "", nil, errors.New("failed to update shop images in database")
	}

	return newLogoURL, newBanners, nil
}

// UploadProductImages uploads up to 4 images for a product
func (s *UploadService) UploadProductImages(
	ctx context.Context,
	userID string,
	files []multipart.File,
	headers []*multipart.FileHeader,
) ([]string, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, errors.New("shop not found")
	}

	if len(files) > 4 {
		return nil, ErrTooManyProductImages
	}

	// Validate safety of all product images before uploading
	for _, f := range files {
		if err := s.validateImageSafety(ctx, f); err != nil {
			return nil, err
		}
	}

	type uploadResult struct {
		index int
		url   string
		err   error
	}

	resChan := make(chan uploadResult, len(files))
	var wg sync.WaitGroup

	folder := fmt.Sprintf("products/%s", shop.ID)

	for i, f := range files {
		wg.Add(1)
		go func(idx int, file multipart.File, header *multipart.FileHeader) {
			defer wg.Done()
			url, err := reuse.UploadImage(file, header, folder)
			resChan <- uploadResult{index: idx, url: url, err: err}
		}(i, f, headers[i])
	}

	wg.Wait()
	close(resChan)

	uploadedURLs := make([]string, len(files))
	var uploadErr error

	for res := range resChan {
		if res.err != nil && uploadErr == nil {
			uploadErr = res.err
		}
		if res.url != "" {
			uploadedURLs[res.index] = res.url
		}
	}

	if uploadErr != nil {
		// Rollback all successfully uploaded images from this concurrent batch
		for _, u := range uploadedURLs {
			if u != "" {
				_ = reuse.DeleteImage(u)
			}
		}
		return nil, uploadErr
	}

	return uploadedURLs, nil
}

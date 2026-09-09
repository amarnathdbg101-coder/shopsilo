// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
)

var (
	ErrSelfReviewNotAllowed = errors.New("shop owners cannot review their own shop")
)

type ReviewService struct {
	reviewRepo *repository.ReviewRepo
	shopRepo   *repository.ShopRepo
}

func NewReviewService(reviewRepo *repository.ReviewRepo, shopRepo *repository.ShopRepo) *ReviewService {
	return &ReviewService{
		reviewRepo: reviewRepo,
		shopRepo:   shopRepo,
	}
}

func (s *ReviewService) AddOrUpdateReview(ctx context.Context, slug string, userID string, input dto.CreateReviewRequest) (*model.ShopReview, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	// Prevent shop owner from reviewing their own store
	if shop.UserID == userID {
		return nil, ErrSelfReviewNotAllowed
	}

	if input.Rating < 1 || input.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	cleanComment := reuse.SanitizeReviewText(input.Comment)
	return s.reviewRepo.UpsertReview(ctx, shop.ID, userID, input.Rating, cleanComment)
}

func (s *ReviewService) ListShopReviews(ctx context.Context, slug string, page, limit int) (*dto.ReviewPaginationResponse, error) {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	reviews, totalCount, err := s.reviewRepo.FindByShopID(ctx, shop.ID, page, limit)
	if err != nil {
		return nil, err
	}

	stats, err := s.reviewRepo.GetShopRatingStats(ctx, shop.ID)
	if err != nil {
		stats = &model.ShopRatingStats{AverageRating: 0, TotalReviews: 0}
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(limit)))
	}

	return &dto.ReviewPaginationResponse{
		Reviews:       reviews,
		AverageRating: stats.AverageRating,
		TotalReviews:  stats.TotalReviews,
		Page:          page,
		Limit:         limit,
		TotalPages:    totalPages,
	}, nil
}

func (s *ReviewService) DeleteMyReview(ctx context.Context, slug string, userID string) error {
	shop, err := s.shopRepo.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	err = s.reviewRepo.DeleteReview(ctx, shop.ID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrReviewNotFound) {
			return ErrReviewNotFound
		}
		return err
	}

	return nil
}

// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"strings"
	"time"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategorySlugTaken = errors.New("category slug already in use")
)

type CategoryService struct {
	repo     *repository.CategoryRepo
	shopRepo *repository.ShopRepo
}

func NewCategoryService(repo *repository.CategoryRepo, shopRepo *repository.ShopRepo) *CategoryService {
	return &CategoryService{repo: repo, shopRepo: shopRepo}
}

func (s *CategoryService) ListCategories(ctx context.Context) ([]*model.Category, error) {
	return s.repo.FindAll(ctx)
}

func (s *CategoryService) AdminListCategories(ctx context.Context) ([]*dto.AdminCategoryItem, error) {
	return s.repo.AdminFindAll(ctx)
}

func (s *CategoryService) CreateCategory(ctx context.Context, input dto.CreateCategoryRequest) (*model.Category, error) {
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = reuse.Slugify(input.Name)
	} else {
		slug = reuse.Slugify(slug)
	}

	// Check slug uniqueness
	if existing, err := s.repo.FindBySlug(ctx, slug); err == nil && existing != nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		slug = fmt.Sprintf("%s-%d", slug, r.Intn(9000)+1000)
	}

	cat := &model.Category{
		Name:        strings.TrimSpace(input.Name),
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		ParentID:    input.ParentID,
		IsActive:    true,
	}

	return s.repo.Create(ctx, cat)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id string, input dto.UpdateCategoryRequest) (*model.Category, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrCategoryNotFound
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = existing.Name
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = existing.Slug
	} else {
		slug = reuse.Slugify(slug)
		if slug != existing.Slug {
			if other, err := s.repo.FindBySlug(ctx, slug); err == nil && other != nil && other.ID != id {
				return nil, ErrCategorySlugTaken
			}
		}
	}

	description := input.Description
	imageURL := input.ImageURL
	parentID := input.ParentID
	isActive := existing.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	cat := &model.Category{
		ID:          id,
		Name:        name,
		Slug:        slug,
		Description: description,
		ImageURL:    imageURL,
		ParentID:    parentID,
		IsActive:    isActive,
	}

	return s.repo.Update(ctx, cat)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ListShopCategories fetches global categories combined with custom categories created by this merchant's shop.
func (s *CategoryService) ListShopCategories(ctx context.Context, userID string) ([]*dto.ShopCategoryItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}
	return s.repo.ListByShop(ctx, shop.ID)
}

// ListShopCategoriesByShopID fetches global + custom categories for public storefront view.
func (s *CategoryService) ListShopCategoriesByShopID(ctx context.Context, shopID string) ([]*dto.ShopCategoryItem, error) {
	return s.repo.ListByShop(ctx, shopID)
}

// CreateShopCategory creates a new custom category tied specifically to the merchant's shop.
func (s *CategoryService) CreateShopCategory(ctx context.Context, userID string, input dto.CreateCategoryRequest) (*model.Category, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("category name is required")
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = reuse.Slugify(name)
	} else {
		slug = reuse.Slugify(slug)
	}

	// Ensure slug uniqueness within this shop's scope
	if existing, err := s.repo.FindShopCategoryBySlug(ctx, shop.ID, slug); err == nil && existing != nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		slug = fmt.Sprintf("%s-%d", slug, r.Intn(9000)+1000)
	}

	icon := strings.TrimSpace(input.Icon)
	if icon == "" {
		icon = "📦"
	}

	cat := &model.Category{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		Icon:        icon,
		ParentID:    input.ParentID,
		IsActive:    true,
	}

	return s.repo.CreateShopCategory(ctx, shop.ID, cat)
}

// UpdateShopCategory modifies a custom category owned by the merchant's shop.
func (s *CategoryService) UpdateShopCategory(ctx context.Context, userID, categoryID string, input dto.UpdateCategoryRequest) (*model.Category, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	name := strings.TrimSpace(input.Name)
	slug := strings.TrimSpace(input.Slug)
	if slug == "" && name != "" {
		slug = reuse.Slugify(name)
	} else if slug != "" {
		slug = reuse.Slugify(slug)
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	icon := strings.TrimSpace(input.Icon)
	if icon == "" {
		icon = "📦"
	}

	cat := &model.Category{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		Icon:        icon,
		ParentID:    input.ParentID,
		IsActive:    isActive,
	}

	return s.repo.UpdateShopCategory(ctx, shop.ID, categoryID, cat)
}

// DeleteShopCategory safely deletes a custom category if no products are assigned to it.
func (s *CategoryService) DeleteShopCategory(ctx context.Context, userID, categoryID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}

	// Safety check: Prevent deleting categories with active products
	count, err := s.repo.CountProductsByCategoryID(ctx, shop.ID, categoryID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete category: %d product(s) are assigned to it. Please reassign or delete those products first", count)
	}

	return s.repo.DeleteShopCategory(ctx, shop.ID, categoryID)
}


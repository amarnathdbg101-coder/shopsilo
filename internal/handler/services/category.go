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
	repo *repository.CategoryRepo
}

func NewCategoryService(repo *repository.CategoryRepo) *CategoryService {
	return &CategoryService{repo: repo}
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


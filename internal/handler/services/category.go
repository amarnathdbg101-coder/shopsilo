// Package services handle business logic.
package services

import (
	"context"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
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

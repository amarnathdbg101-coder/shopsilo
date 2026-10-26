// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"strings"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

type ProcurementService struct {
	repo     *repository.ProcurementRepo
	shopRepo *repository.ShopRepo
}

func NewProcurementService(repo *repository.ProcurementRepo, shopRepo *repository.ShopRepo) *ProcurementService {
	return &ProcurementService{
		repo:     repo,
		shopRepo: shopRepo,
	}
}

// ListProcurementItems retrieves all procurement items for the merchant's shop.
func (s *ProcurementService) ListProcurementItems(ctx context.Context, userID string) ([]*model.ProcurementItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}
	return s.repo.List(ctx, shop.ID)
}

// CreateProcurementItem adds a new item to the shop's procurement list and broadcasts real-time update.
func (s *ProcurementService) CreateProcurementItem(ctx context.Context, userID string, input dto.CreateProcurementItemRequest) (*model.ProcurementItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("item name is required")
	}

	qty := strings.TrimSpace(input.Qty)
	if qty == "" {
		qty = "1"
	}

	cat := strings.TrimSpace(input.Category)
	if cat == "" {
		cat = "General Store"
	}

	it := &model.ProcurementItem{
		Name:           name,
		Category:       cat,
		Qty:            qty,
		EstimatedPrice: input.EstimatedPrice,
		ActualPrice:    input.ActualPrice,
		DealerName:     strings.TrimSpace(input.DealerName),
		DealerPhone:    strings.TrimSpace(input.DealerPhone),
		Notes:          strings.TrimSpace(input.Notes),
		IsCompleted:    input.IsCompleted,
	}

	created, err := s.repo.Create(ctx, shop.ID, it)
	if err == nil && created != nil {
		wsHub := GetWebSocketHub(nil)
		wsHub.BroadcastToShop(shop.ID, "PROCUREMENT_MUTATED", created)
	}
	return created, err
}

// UpdateProcurementItem updates an existing procurement item.
func (s *ProcurementService) UpdateProcurementItem(ctx context.Context, userID, id string, input dto.UpdateProcurementItemRequest) (*model.ProcurementItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	items, err := s.repo.List(ctx, shop.ID)
	if err != nil {
		return nil, err
	}

	var existing *model.ProcurementItem
	for _, it := range items {
		if it.ID == id {
			existing = it
			break
		}
	}
	if existing == nil {
		return nil, repository.ErrProcurementItemNotFound
	}

	if input.Name != nil && strings.TrimSpace(*input.Name) != "" {
		existing.Name = strings.TrimSpace(*input.Name)
	}
	if input.Category != nil {
		existing.Category = strings.TrimSpace(*input.Category)
	}
	if input.Qty != nil && strings.TrimSpace(*input.Qty) != "" {
		existing.Qty = strings.TrimSpace(*input.Qty)
	}
	if input.EstimatedPrice != nil {
		existing.EstimatedPrice = *input.EstimatedPrice
	}
	if input.ActualPrice != nil {
		existing.ActualPrice = *input.ActualPrice
	}
	if input.DealerName != nil {
		existing.DealerName = strings.TrimSpace(*input.DealerName)
	}
	if input.DealerPhone != nil {
		existing.DealerPhone = strings.TrimSpace(*input.DealerPhone)
	}
	if input.Notes != nil {
		existing.Notes = strings.TrimSpace(*input.Notes)
	}
	if input.IsCompleted != nil {
		existing.IsCompleted = *input.IsCompleted
	}

	updated, err := s.repo.Update(ctx, shop.ID, id, existing)
	if err == nil && updated != nil {
		wsHub := GetWebSocketHub(nil)
		wsHub.BroadcastToShop(shop.ID, "PROCUREMENT_MUTATED", updated)
	}
	return updated, err
}

// ToggleProcurementItem flips the completed checkbox for an item.
func (s *ProcurementService) ToggleProcurementItem(ctx context.Context, userID, id string) (*model.ProcurementItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	toggled, err := s.repo.Toggle(ctx, shop.ID, id)
	if err == nil && toggled != nil {
		wsHub := GetWebSocketHub(nil)
		wsHub.BroadcastToShop(shop.ID, "PROCUREMENT_MUTATED", toggled)
	}
	return toggled, err
}

// DeleteProcurementItem removes an item from the procurement list.
func (s *ProcurementService) DeleteProcurementItem(ctx context.Context, userID, id string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}

	if err := s.repo.Delete(ctx, shop.ID, id); err != nil {
		return err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(shop.ID, "PROCUREMENT_MUTATED", map[string]interface{}{"deleted_id": id})
	return nil
}

// ClearCompleted removes all completed items from the procurement list.
func (s *ProcurementService) ClearCompleted(ctx context.Context, userID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}

	if err := s.repo.ClearCompleted(ctx, shop.ID); err != nil {
		return err
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(shop.ID, "PROCUREMENT_MUTATED", map[string]interface{}{"cleared_completed": true})
	return nil
}

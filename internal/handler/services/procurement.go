// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"math"
	"strings"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

// validateProcurementPrices ensures estimated and actual prices are finite and non-negative.
func validateProcurementPrices(estimatedPrice, actualPrice float64) error {
	if math.IsNaN(estimatedPrice) || math.IsInf(estimatedPrice, 0) || estimatedPrice < 0 {
		return errors.New("estimated_price must be a valid non-negative amount")
	}
	if math.IsNaN(actualPrice) || math.IsInf(actualPrice, 0) || actualPrice < 0 {
		return errors.New("actual_price must be a valid non-negative amount")
	}
	return nil
}

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
	if err := validateProcurementPrices(input.EstimatedPrice, input.ActualPrice); err != nil {
		return nil, err
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
		if err := validateProcurementPrices(*input.EstimatedPrice, 0); err != nil {
			return nil, err
		}
		existing.EstimatedPrice = *input.EstimatedPrice
	}
	if input.ActualPrice != nil {
		if err := validateProcurementPrices(0, *input.ActualPrice); err != nil {
			return nil, err
		}
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
	if strings.TrimSpace(id) == "" {
		return nil, repository.ErrProcurementItemNotFound
	}
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
	if strings.TrimSpace(id) == "" {
		return repository.ErrProcurementItemNotFound
	}
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

// ListWholesalers retrieves all registered permanent wholesalers for a shop.
func (s *ProcurementService) ListWholesalers(ctx context.Context, userID string) ([]*model.Wholesaler, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}
	return s.repo.ListWholesalers(ctx, shop.ID)
}

// CreateWholesaler adds a new permanent wholesaler to the shop's supplier directory.
func (s *ProcurementService) CreateWholesaler(ctx context.Context, userID string, input dto.CreateWholesalerRequest) (*model.Wholesaler, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("wholesaler name is required")
	}

	w := &model.Wholesaler{
		ShopID:  shop.ID,
		Name:    name,
		Phone:   strings.TrimSpace(input.Phone),
		Address: strings.TrimSpace(input.Address),
		GSTIN:   strings.TrimSpace(input.GSTIN),
		Notes:   strings.TrimSpace(input.Notes),
	}

	if err := s.repo.CreateWholesaler(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// UpdateWholesaler modifies an existing wholesaler in the shop directory.
func (s *ProcurementService) UpdateWholesaler(ctx context.Context, userID, id string, input dto.UpdateWholesalerRequest) (*model.Wholesaler, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	w := &model.Wholesaler{
		ID:     id,
		ShopID: shop.ID,
	}

	if input.Name != nil {
		w.Name = strings.TrimSpace(*input.Name)
	}
	if input.Phone != nil {
		w.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Address != nil {
		w.Address = strings.TrimSpace(*input.Address)
	}
	if input.GSTIN != nil {
		w.GSTIN = strings.TrimSpace(*input.GSTIN)
	}
	if input.Notes != nil {
		w.Notes = strings.TrimSpace(*input.Notes)
	}

	if err := s.repo.UpdateWholesaler(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// DeleteWholesaler removes a wholesaler from the shop directory.
func (s *ProcurementService) DeleteWholesaler(ctx context.Context, userID, id string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}
	return s.repo.DeleteWholesaler(ctx, shop.ID, id)
}

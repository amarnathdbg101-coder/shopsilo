// Package services handle business logic.
package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
	"time"
)

type ReservationService struct {
	resRepo     *repository.ReservationRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
}

func NewReservationService(
	resRepo *repository.ReservationRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
) *ReservationService {
	return &ReservationService{
		resRepo:     resRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
	}
}

func generatePickupCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func generateReservationNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	randPart := int64(1000)
	if err == nil {
		randPart = n.Int64() + 1000
	}
	return fmt.Sprintf("HLD-%d-%04d", time.Now().Unix()%100000, randPart)
}

// CreateReservation lets a customer place an in-store hold on a product.
func (s *ReservationService) CreateReservation(ctx context.Context, userID string, input dto.CreateReservationRequest) (*model.Reservation, error) {
	// 1. Anti-abuse check: max 5 active reservations per customer
	activeCount, err := s.resRepo.CountActiveByUserID(ctx, userID)
	if err == nil && activeCount >= 5 {
		return nil, ErrMaxActiveReservations
	}

	// 2. Fetch product
	prod, err := s.productRepo.FindByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.New("product not found or unavailable")
	}
	if !prod.IsActive {
		return nil, errors.New("product is currently inactive")
	}

	// 3. Check inventory if available
	if prod.Inventory != nil {
		availableStock := prod.Inventory.Quantity - prod.Inventory.ReservedQuantity
		if availableStock < input.Quantity {
			return nil, ErrProductOutOfStock
		}
	}

	// 4. Fetch and verify shop
	shop, err := s.shopRepo.FindByID(ctx, prod.ShopID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	if !shop.IsActive {
		return nil, errors.New("shop is currently inactive")
	}

	// 5. Default hold duration: 4 hours (bounds: 1-24)
	holdHours := input.HoldHours
	if holdHours < 1 || holdHours > 24 {
		holdHours = 4
	}

	reservation := &model.Reservation{
		ReservationNumber: generateReservationNumber(),
		UserID:            userID,
		ShopID:            prod.ShopID,
		ProductID:         prod.ID,
		Quantity:          input.Quantity,
		PickupCode:        generatePickupCode(),
		Status:            "active",
		ExpiresAt:         time.Now().Add(time.Duration(holdHours) * time.Hour),
		Notes:             strings.TrimSpace(input.Notes),
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start database transaction")
	}
	defer tx.Rollback(ctx)

	created, err := s.resRepo.CreateWithTx(ctx, tx, reservation)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit reservation")
	}

	// Return enriched reservation
	fullRes, err := s.resRepo.FindByID(ctx, created.ID)
	if err != nil {
		return created, nil
	}

	enrichShop(fullRes.Shop)
	return fullRes, nil
}

func (s *ReservationService) ListUserReservations(ctx context.Context, userID string, filter dto.ReservationFilter) (*dto.ReservationPaginationResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		filter.Limit = 10
	}

	list, totalCount, err := s.resRepo.FindByUserID(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	for _, r := range list {
		enrichShop(r.Shop)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ReservationPaginationResponse{
		Reservations: list,
		TotalCount:   totalCount,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

func (s *ReservationService) GetReservationByID(ctx context.Context, id, userID string) (*model.Reservation, error) {
	res, err := s.resRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if res.UserID != userID {
		return nil, ErrReservationNotFound
	}

	enrichShop(res.Shop)
	return res, nil
}

func (s *ReservationService) CancelUserReservation(ctx context.Context, id, userID string) (*model.Reservation, error) {
	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	res, err := s.resRepo.CancelWithTx(ctx, tx, id, userID, "")
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit cancellation")
	}

	return res, nil
}

// ListShopReservations lists incoming customer reservations for the authenticated shop owner.
func (s *ReservationService) ListShopReservations(ctx context.Context, shopOwnerUserID string, filter dto.ReservationFilter) (*dto.ReservationPaginationResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		filter.Limit = 10
	}

	list, totalCount, err := s.resRepo.FindByShopID(ctx, shop.ID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ReservationPaginationResponse{
		Reservations: list,
		TotalCount:   totalCount,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

// VerifyShopReservation validates a customer's 6-digit pickup code at the store counter and completes the hold.
func (s *ReservationService) VerifyShopReservation(ctx context.Context, shopOwnerUserID string, input dto.VerifyReservationRequest) (*model.Reservation, error) {
	if strings.TrimSpace(input.PickupCode) == "" && strings.TrimSpace(input.ReservationNumber) == "" {
		return nil, errors.New("either pickup_code or reservation_number is required")
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	completed, err := s.resRepo.VerifyAndCompleteWithTx(ctx, tx, shop.ID, input.PickupCode, input.ReservationNumber)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrInvalidVerificationCode
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit verification")
	}

	return completed, nil
}

// CancelShopReservation allows the shopkeeper to cancel an in-store reservation (e.g. damaged or customer no-show).
func (s *ReservationService) CancelShopReservation(ctx context.Context, id, shopOwnerUserID string) (*model.Reservation, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	cancelled, err := s.resRepo.CancelWithTx(ctx, tx, id, "", shop.ID)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit cancellation")
	}

	return cancelled, nil
}

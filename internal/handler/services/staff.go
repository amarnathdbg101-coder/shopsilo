// Package services handles business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"strings"
)

var (
	ErrInvalidPINFormat = errors.New("pin must be exactly 4 numeric digits")
)

type StaffService struct {
	staffRepo *repository.StaffRepo
	shopRepo  *repository.ShopRepo
}

func NewStaffService(staffRepo *repository.StaffRepo, shopRepo *repository.ShopRepo) *StaffService {
	return &StaffService{
		staffRepo: staffRepo,
		shopRepo:  shopRepo,
	}
}

// CreateStaff allows the shop owner to add a cashier/helper sub-account.
func (s *StaffService) CreateStaff(ctx context.Context, shopOwnerUserID string, req dto.CreateStaffRequest) (*model.ShopStaff, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	pin := strings.TrimSpace(req.PIN)
	if len(pin) != 4 {
		return nil, ErrInvalidPINFormat
	}

	pinHash, err := reuse.HashPassword(pin)
	if err != nil {
		return nil, errors.New("failed to secure staff PIN")
	}

	staffToCreate := &model.ShopStaff{
		ShopID:   shop.ID,
		FullName: strings.TrimSpace(req.FullName),
		Phone:    strings.TrimSpace(req.Phone),
		PinHash:  pinHash,
		Role:     req.Role,
		IsActive: true,
	}

	return s.staffRepo.Create(ctx, staffToCreate)
}

// ListStaff returns all cashiers and helpers working in the shop.
func (s *StaffService) ListStaff(ctx context.Context, shopOwnerUserID string) ([]*model.ShopStaff, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.staffRepo.ListByShopID(ctx, shop.ID)
}

// UpdateStaff updates staff profile, PIN, or active status.
func (s *StaffService) UpdateStaff(ctx context.Context, shopOwnerUserID, staffID string, req dto.UpdateStaffRequest) (*model.ShopStaff, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	existing, err := s.staffRepo.FindByID(ctx, staffID, shop.ID)
	if err != nil {
		return nil, err
	}

	if req.FullName != nil {
		existing.FullName = strings.TrimSpace(*req.FullName)
	}
	if req.Phone != nil {
		existing.Phone = strings.TrimSpace(*req.Phone)
	}
	if req.Role != nil {
		existing.Role = *req.Role
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.PIN != nil && strings.TrimSpace(*req.PIN) != "" {
		pin := strings.TrimSpace(*req.PIN)
		if len(pin) != 4 {
			return nil, ErrInvalidPINFormat
		}
		newHash, err := reuse.HashPassword(pin)
		if err != nil {
			return nil, errors.New("failed to secure new PIN")
		}
		existing.PinHash = newHash
	}

	return s.staffRepo.Update(ctx, existing)
}

// DeleteStaff removes a cashier sub-account.
func (s *StaffService) DeleteStaff(ctx context.Context, shopOwnerUserID, staffID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	return s.staffRepo.Delete(ctx, staffID, shop.ID)
}

// StaffLogin authenticates a cashier at counter using 4-digit PIN and issues a cashier JWT token.
func (s *StaffService) StaffLogin(ctx context.Context, req dto.StaffLoginRequest) (*dto.StaffLoginResponse, error) {
	shopID := strings.TrimSpace(req.ShopID)
	phone := strings.TrimSpace(req.Phone)
	pin := strings.TrimSpace(req.PIN)

	staff, err := s.staffRepo.FindByPhoneAndShopID(ctx, shopID, phone)
	if err != nil {
		if errors.Is(err, repository.ErrStaffNotFound) {
			return nil, repository.ErrInvalidStaffPIN
		}
		return nil, err
	}

	if !staff.IsActive {
		return nil, repository.ErrStaffDeactivated
	}

	if !reuse.CheckPasswordHash(pin, staff.PinHash) {
		return nil, repository.ErrInvalidStaffPIN
	}

	// Generate Cashier Token
	token, err := reuse.GenerateJwt(staff.ID, fmt.Sprintf("%s@staff.local", staff.Phone), staff.Role)
	if err != nil {
		return nil, errors.New("failed to generate staff token")
	}

	return &dto.StaffLoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		StaffID:     staff.ID,
		FullName:    staff.FullName,
		Role:        staff.Role,
		ShopID:      staff.ShopID,
	}, nil
}

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

// validateNumericPIN ensures a staff PIN is exactly 4 ASCII digits.
func validateNumericPIN(raw string) (string, error) {
	pin := strings.TrimSpace(raw)
	if len(pin) != 4 {
		return "", ErrInvalidPINFormat
	}
	for i := 0; i < len(pin); i++ {
		if pin[i] < '0' || pin[i] > '9' {
			return "", ErrInvalidPINFormat
		}
	}
	return pin, nil
}

// validateCreateStaffInput validates staff name, phone, role, and 4-digit numeric PIN.
func validateCreateStaffInput(req *dto.CreateStaffRequest) (string, error) {
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		return "", errors.New("staff full_name is required")
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		return "", errors.New("staff phone is required")
	}
	pin, err := validateNumericPIN(req.PIN)
	if err != nil {
		return "", err
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "cashier"
	}
	req.Role = role
	return pin, nil
}

// CreateStaff allows the shop owner to add a cashier/helper sub-account.
func (s *StaffService) CreateStaff(ctx context.Context, shopOwnerUserID string, req dto.CreateStaffRequest) (*model.ShopStaff, error) {
	pin, err := validateCreateStaffInput(&req)
	if err != nil {
		return nil, err
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	pinHash, err := reuse.HashPassword(pin)
	if err != nil {
		return nil, errors.New("failed to secure staff PIN")
	}

	staffToCreate := &model.ShopStaff{
		ShopID:   shop.ID,
		FullName: req.FullName,
		Phone:    req.Phone,
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
	if strings.TrimSpace(staffID) == "" {
		return nil, repository.ErrStaffNotFound
	}
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
		pin, err := validateNumericPIN(*req.PIN)
		if err != nil {
			return nil, err
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
	if strings.TrimSpace(staffID) == "" {
		return repository.ErrStaffNotFound
	}
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
	pin, err := validateNumericPIN(req.PIN)
	if err != nil {
		return nil, repository.ErrInvalidStaffPIN
	}
	if shopID == "" || phone == "" {
		return nil, repository.ErrInvalidStaffPIN
	}

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

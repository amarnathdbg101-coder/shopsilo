// Package services handles business logic.
package services

import (
	"context"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)
type ModerationService struct {
	modRepo  *repository.ModerationRepo
	shopRepo *repository.ShopRepo
	userRepo *repository.UserRepo
}

func NewModerationService(
	modRepo *repository.ModerationRepo,
	shopRepo *repository.ShopRepo,
	userRepo *repository.UserRepo,
) *ModerationService {
	return &ModerationService{
		modRepo:  modRepo,
		shopRepo: shopRepo,
		userRepo: userRepo,
	}
}

// SubmitReport creates a content report and automatically quarantines shops exceeding the report threshold.
func (s *ModerationService) SubmitReport(
	ctx context.Context,
	reporterUserID *string,
	req dto.CreateReportRequest,
	ip, userAgent string,
) (*model.ContentReport, bool, error) {
	// 1. Validate target existence if target is a shop
	if req.TargetType == model.TargetTypeShop {
		shop, err := s.shopRepo.FindByID(ctx, req.TargetID)
		if err != nil || shop == nil {
			return nil, false, ErrTargetNotFound
		}
	}

	report := &model.ContentReport{
		ReporterUserID:    reporterUserID,
		TargetType:        req.TargetType,
		TargetID:          req.TargetID,
		Reason:            req.Reason,
		Details:           req.Details,
		ReporterIP:        ip,
		ReporterUserAgent: userAgent,
	}

	createdReport, err := s.modRepo.CreateReport(ctx, report)
	if err != nil {
		return nil, false, fmt.Errorf("failed to save report: %w", err)
	}

	// 2. Increment flag counter if shop
	if req.TargetType == model.TargetTypeShop {
		_ = s.modRepo.IncrementShopFlagCount(ctx, req.TargetID)
	}

	// 3. Automated Safe Harbor Quarantine Rule (IT Rules 2021)
	// If 3 or more unique reports for sexual content or illegal substances arrive in 24 hours:
	// Automatically quarantine (flag) the shop so it is hidden from public directories pending admin review.
	quarantined := false
	if req.Reason == model.ReasonSexualContent || req.Reason == model.ReasonIllegalSubstance {
		count, err := s.modRepo.CountRecentSevereReports(ctx, req.TargetType, req.TargetID)
		if err == nil && count >= 3 {
			if req.TargetType == model.TargetTypeShop {
				quarantineReason := "Auto-quarantined: 3+ community reports for sexual content or illegal materials"
				_ = s.modRepo.SetShopStatus(ctx, req.TargetID, model.ShopStatusFlagged, quarantineReason)
				quarantined = true
			}
		}
	}

	return createdReport, quarantined, nil
}

// BanShop performs a complete 1-click ban: bans the shop, deactivates owner account, and blacklists IP/device/phone.
func (s *ModerationService) BanShop(
	ctx context.Context,
	shopID string,
	req dto.BanShopRequest,
	adminID string,
) error {
	shop, err := s.shopRepo.FindByID(ctx, shopID)
	if err != nil {
		return repository.ErrShopNotFound
	}

	// 1. Mark shop status as banned
	if err := s.modRepo.SetShopStatus(ctx, shopID, model.ShopStatusBanned, req.Reason); err != nil {
		return fmt.Errorf("failed to update shop status: %w", err)
	}

	// 2. Deactivate owner account
	_ = s.userRepo.DeactivateUser(ctx, shop.UserID)

	// 3. Blacklist IP if requested
	if req.BlacklistIP && shop.CreationIP != "" {
		_ = s.modRepo.BanEntity(
			ctx,
			model.EntityTypeIP,
			shop.CreationIP,
			fmt.Sprintf("Banned shop (%s) IP address: %s", shop.Name, req.Reason),
			&adminID,
		)
	}

	// 4. Blacklist Device Fingerprint if requested
	if req.BlacklistDevice && shop.DeviceFingerprint != "" {
		_ = s.modRepo.BanEntity(
			ctx,
			model.EntityTypeDeviceID,
			shop.DeviceFingerprint,
			fmt.Sprintf("Banned shop (%s) Device ID: %s", shop.Name, req.Reason),
			&adminID,
		)
	}

	// 5. Blacklist phone number
	if shop.Phone != "" {
		_ = s.modRepo.BanEntity(
			ctx,
			model.EntityTypePhone,
			shop.Phone,
			fmt.Sprintf("Banned shop (%s) Phone: %s", shop.Name, req.Reason),
			&adminID,
		)
	}

	return nil
}

// AddBannedEntity adds an arbitrary blacklist entry.
func (s *ModerationService) AddBannedEntity(ctx context.Context, req dto.AddBannedEntityRequest, adminID string) error {
	return s.modRepo.BanEntity(ctx, req.EntityType, req.EntityValue, req.Reason, &adminID)
}

// UnbanEntity removes a record from the blacklist.
func (s *ModerationService) UnbanEntity(ctx context.Context, id string) error {
	return s.modRepo.UnbanEntity(ctx, id)
}

// ListBannedEntities lists all active blacklist entries.
func (s *ModerationService) ListBannedEntities(ctx context.Context, page, limit int) ([]model.BannedEntity, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit
	return s.modRepo.ListBannedEntities(ctx, limit, offset)
}

// ListReports retrieves content moderation reports for review.
func (s *ModerationService) ListReports(ctx context.Context, status string, page, limit int) ([]model.ContentReport, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit
	return s.modRepo.ListReports(ctx, status, limit, offset)
}

// ResolveReport handles administrative action or dismissal on a report.
func (s *ModerationService) ResolveReport(ctx context.Context, reportID string, req dto.ResolveReportRequest, adminID string) error {
	return s.modRepo.ResolveReport(ctx, reportID, req.Status, req.ActionTaken, adminID)
}

// GetAdminStats returns aggregate metrics for the dashboard.
func (s *ModerationService) GetAdminStats(ctx context.Context) (*dto.AdminStatsResponse, error) {
	return s.modRepo.GetAdminStats(ctx)
}

// ListShopsForAdmin retrieves shops with pagination and filters for admin review.
func (s *ModerationService) ListShopsForAdmin(ctx context.Context, status, search string, page, limit int) ([]*model.Shop, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit
	return s.modRepo.ListShopsForAdmin(ctx, status, search, limit, offset)
}

// UpdateShopStatus allows an admin to approve, suspend, or update a shop's status.
func (s *ModerationService) UpdateShopStatus(ctx context.Context, shopID, newStatus, reason, adminID string) error {
	return s.modRepo.SetShopStatus(ctx, shopID, newStatus, reason)
}

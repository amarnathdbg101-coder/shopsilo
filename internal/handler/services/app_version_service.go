// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"errors"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

// AppVersionService manages version lifecycle and update eligibility checks.
type AppVersionService struct {
	versionRepo *repository.AppVersionRepo
	logger      *zap.Logger
}

// NewAppVersionService constructs a new AppVersionService.
func NewAppVersionService(versionRepo *repository.AppVersionRepo, logger *zap.Logger) *AppVersionService {
	return &AppVersionService{
		versionRepo: versionRepo,
		logger:      logger,
	}
}

// CheckUpdate evaluates client's current version code against database releases.
// It determines if an update is available and whether it is an optional or forced update.
func (s *AppVersionService) CheckUpdate(ctx context.Context, appType, platform string, clientVersionCode int) (*model.VersionCheckResponse, error) {
	if appType == "" {
		appType = "merchant_pos"
	}
	if platform == "" {
		platform = "android"
	}

	latest, err := s.versionRepo.GetLatestActiveVersion(ctx, appType, platform)
	if err != nil {
		if errors.Is(err, repository.ErrVersionNotFound) {
			// No active version found in database, client is assumed up-to-date
			return &model.VersionCheckResponse{
				UpdateAvailable: false,
				CurrentCode:     clientVersionCode,
			}, nil
		}
		s.logger.Error("failed checking app update eligibility", zap.Error(err))
		return nil, err
	}

	// If client version code is already equal to or higher than latest, no update is required
	if clientVersionCode >= latest.VersionCode {
		return &model.VersionCheckResponse{
			UpdateAvailable:         false,
			LatestVersion:           latest.VersionName,
			LatestCode:              latest.VersionCode,
			CurrentCode:             clientVersionCode,
			MinSupportedVersionCode: latest.MinSupportedVersionCode,
		}, nil
	}

	// Client is running an older build; determine force update condition:
	// 1. Database explicitly marked this release as is_force_update = true, OR
	// 2. Client build is older than the min_supported_version_code threshold (e.g. breaking API changes).
	isForce := latest.IsForceUpdate || (clientVersionCode < latest.MinSupportedVersionCode)

	return &model.VersionCheckResponse{
		UpdateAvailable:         true,
		LatestVersion:           latest.VersionName,
		LatestCode:              latest.VersionCode,
		CurrentCode:             clientVersionCode,
		IsForceUpdate:           isForce,
		DownloadURL:             latest.ApkURL,
		FileSizeBytes:           latest.FileSizeBytes,
		ReleaseNotes:            latest.ReleaseNotes,
		MinSupportedVersionCode: latest.MinSupportedVersionCode,
	}, nil
}

// PublishRelease creates and persists a new release record.
func (s *AppVersionService) PublishRelease(ctx context.Context, v *model.AppVersion) (*model.AppVersion, error) {
	if v.VersionCode <= 0 {
		return nil, errors.New("version_code must be positive integer")
	}
	if v.VersionName == "" {
		return nil, errors.New("version_name is required")
	}
	if v.ApkURL == "" {
		return nil, errors.New("apk_url is required")
	}

	return s.versionRepo.PublishVersion(ctx, v)
}

// Package repository handles database queries for OTA application versions.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrVersionNotFound = errors.New("app version not found")
)

// AppVersionRepo manages database queries for published app releases and version checks.
type AppVersionRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewAppVersionRepo initializes an AppVersionRepo with a pgx connection pool and logger.
func NewAppVersionRepo(db *pgxpool.Pool, logger *zap.Logger) *AppVersionRepo {
	return &AppVersionRepo{
		db:     db,
		logger: logger,
	}
}

// GetLatestActiveVersion fetches the newest active release record for a given app type and platform.
func (r *AppVersionRepo) GetLatestActiveVersion(ctx context.Context, appType, platform string) (*model.AppVersion, error) {
	query := `
		SELECT id, app_type, platform, version_code, version_name, min_supported_version_code,
		       apk_url, file_size_bytes, release_notes, is_force_update, is_active, created_at
		FROM app_versions
		WHERE app_type = $1 AND platform = $2 AND is_active = TRUE
		ORDER BY version_code DESC
		LIMIT 1;
	`

	var v model.AppVersion
	var rawNotes []byte

	err := r.db.QueryRow(ctx, query, appType, platform).Scan(
		&v.ID,
		&v.AppType,
		&v.Platform,
		&v.VersionCode,
		&v.VersionName,
		&v.MinSupportedVersionCode,
		&v.ApkURL,
		&v.FileSizeBytes,
		&rawNotes,
		&v.IsForceUpdate,
		&v.IsActive,
		&v.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVersionNotFound
		}
		r.logger.Error("failed to query latest app version",
			zap.String("app_type", appType),
			zap.String("platform", platform),
			zap.Error(err),
		)
		return nil, fmt.Errorf("querying latest app version failed: %w", err)
	}

	// Unmarshal JSONB release notes array
	if len(rawNotes) > 0 {
		if err := json.Unmarshal(rawNotes, &v.ReleaseNotes); err != nil {
			r.logger.Warn("failed to unmarshal release_notes JSONB, defaulting to empty list", zap.Error(err))
			v.ReleaseNotes = []string{}
		}
	} else {
		v.ReleaseNotes = []string{}
	}

	return &v, nil
}

// PublishVersion inserts a new release version record into app_versions.
func (r *AppVersionRepo) PublishVersion(ctx context.Context, v *model.AppVersion) (*model.AppVersion, error) {
	notesJSON, err := json.Marshal(v.ReleaseNotes)
	if err != nil {
		notesJSON = []byte("[]")
	}

	query := `
		INSERT INTO app_versions (
			app_type, platform, version_code, version_name, min_supported_version_code,
			apk_url, file_size_bytes, release_notes, is_force_update, is_active, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, created_at;
	`

	err = r.db.QueryRow(ctx, query,
		v.AppType,
		v.Platform,
		v.VersionCode,
		v.VersionName,
		v.MinSupportedVersionCode,
		v.ApkURL,
		v.FileSizeBytes,
		notesJSON,
		v.IsForceUpdate,
		v.IsActive,
	).Scan(&v.ID, &v.CreatedAt)

	if err != nil {
		r.logger.Error("failed to insert new app version", zap.Error(err))
		return nil, fmt.Errorf("publishing app version failed: %w", err)
	}

	return v, nil
}

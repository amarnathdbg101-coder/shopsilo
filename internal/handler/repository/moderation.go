// Package repository handles database operations.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrReportNotFound = errors.New("report not found")
	ErrEntityAlreadyBanned = errors.New("entity is already banned")
)

type ModerationRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewModerationRepo(db *pgxpool.Pool, logger *zap.Logger) *ModerationRepo {
	return &ModerationRepo{
		db:     db,
		logger: logger,
	}
}

// IsEntityBanned checks whether a specific entity (ip, device, phone, hash) is in the blacklist.
func (r *ModerationRepo) IsEntityBanned(ctx context.Context, entityType, entityValue string) (bool, error) {
	if strings.TrimSpace(entityValue) == "" {
		return false, nil
	}
	query := `SELECT 1 FROM banned_entities WHERE entity_type = $1 AND entity_value = $2 LIMIT 1`
	var exists int
	err := r.db.QueryRow(ctx, query, entityType, strings.TrimSpace(entityValue)).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsAnyEntityBanned checks multiple candidate identifiers (ip, device_id, phone, etc.) in a single query.
func (r *ModerationRepo) IsAnyEntityBanned(ctx context.Context, candidates map[string]string) (bool, string, error) {
	if len(candidates) == 0 {
		return false, "", nil
	}

	for entityType, entityVal := range candidates {
		val := strings.TrimSpace(entityVal)
		if val == "" {
			continue
		}
		banned, err := r.IsEntityBanned(ctx, entityType, val)
		if err != nil {
			return false, "", err
		}
		if banned {
			return true, entityType + ":" + val, nil
		}
	}
	return false, "", nil
}

// BanEntity inserts an entity into the banned_entities blacklist.
func (r *ModerationRepo) BanEntity(ctx context.Context, entityType, entityValue, reason string, bannedBy *string) error {
	query := `
		INSERT INTO banned_entities (entity_type, entity_value, reason, banned_by, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (entity_value) DO UPDATE
		SET reason = EXCLUDED.reason, banned_by = EXCLUDED.banned_by
	`
	_, err := r.db.Exec(ctx, query, entityType, strings.TrimSpace(entityValue), reason, bannedBy)
	return err
}

// UnbanEntity removes an entity from the blacklist.
func (r *ModerationRepo) UnbanEntity(ctx context.Context, id string) error {
	query := `DELETE FROM banned_entities WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// ListBannedEntities lists all active blacklist entries.
func (r *ModerationRepo) ListBannedEntities(ctx context.Context, limit, offset int) ([]model.BannedEntity, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `
		SELECT id, entity_type, entity_value, reason, banned_by, created_at
		FROM banned_entities
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.BannedEntity
	for rows.Next() {
		var b model.BannedEntity
		if err := rows.Scan(&b.ID, &b.EntityType, &b.EntityValue, &b.Reason, &b.BannedBy, &b.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, nil
}

// GetBannedImageHashes fetches all banned perceptual image hashes for rapid memory comparison.
func (r *ModerationRepo) GetBannedImageHashes(ctx context.Context) ([]string, error) {
	query := `SELECT entity_value FROM banned_entities WHERE entity_type = $1`
	rows, err := r.db.Query(ctx, query, model.EntityTypeImageHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err == nil {
			hashes = append(hashes, h)
		}
	}
	return hashes, nil
}

// CreateReport records a new grievance / moderation report.
func (r *ModerationRepo) CreateReport(ctx context.Context, rep *model.ContentReport) (*model.ContentReport, error) {
	query := `
		INSERT INTO content_reports (
			reporter_user_id, target_type, target_id, reason, details, status,
			reporter_ip, reporter_user_agent, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, reporter_user_id, target_type, target_id, reason, details, status,
		          reporter_ip, reporter_user_agent, action_taken, reviewed_by, reviewed_at, created_at
	`
	created := &model.ContentReport{}
	err := r.db.QueryRow(
		ctx,
		query,
		rep.ReporterUserID,
		rep.TargetType,
		rep.TargetID,
		rep.Reason,
		rep.Details,
		model.ReportStatusPending,
		rep.ReporterIP,
		rep.ReporterUserAgent,
	).Scan(
		&created.ID,
		&created.ReporterUserID,
		&created.TargetType,
		&created.TargetID,
		&created.Reason,
		&created.Details,
		&created.Status,
		&created.ReporterIP,
		&created.ReporterUserAgent,
		&created.ActionTaken,
		&created.ReviewedBy,
		&created.ReviewedAt,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// CountRecentSevereReports counts reports in the last 24 hours for sexual content or illegal items.
func (r *ModerationRepo) CountRecentSevereReports(ctx context.Context, targetType, targetID string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT COALESCE(reporter_user_id::text, reporter_ip))
		FROM content_reports
		WHERE target_type = $1 
		  AND target_id = $2 
		  AND reason IN ($3, $4)
		  AND created_at >= NOW() - INTERVAL '24 hours'
	`
	var count int
	err := r.db.QueryRow(ctx, query, targetType, targetID, model.ReasonSexualContent, model.ReasonIllegalSubstance).Scan(&count)
	return count, err
}

// IncrementShopFlagCount increments flagged count on a shop.
func (r *ModerationRepo) IncrementShopFlagCount(ctx context.Context, shopID string) error {
	query := `UPDATE shops SET flagged_count = flagged_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, shopID)
	return err
}

// SetShopStatus updates the status and suspension reason of a shop.
func (r *ModerationRepo) SetShopStatus(ctx context.Context, shopID, status, reason string) error {
	query := `
		UPDATE shops
		SET status = $1, suspension_reason = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, status, reason, shopID)
	return err
}

// ListReports returns reports with optional status filter.
func (r *ModerationRepo) ListReports(ctx context.Context, status string, limit, offset int) ([]model.ContentReport, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows pgx.Rows
	var err error

	if status != "" {
		query := `
			SELECT id, reporter_user_id, target_type, target_id, reason, details, status,
			       reporter_ip, reporter_user_agent, action_taken, reviewed_by, reviewed_at, created_at
			FROM content_reports
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		rows, err = r.db.Query(ctx, query, status, limit, offset)
	} else {
		query := `
			SELECT id, reporter_user_id, target_type, target_id, reason, details, status,
			       reporter_ip, reporter_user_agent, action_taken, reviewed_by, reviewed_at, created_at
			FROM content_reports
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		rows, err = r.db.Query(ctx, query, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.ContentReport
	for rows.Next() {
		var rep model.ContentReport
		if err := rows.Scan(
			&rep.ID,
			&rep.ReporterUserID,
			&rep.TargetType,
			&rep.TargetID,
			&rep.Reason,
			&rep.Details,
			&rep.Status,
			&rep.ReporterIP,
			&rep.ReporterUserAgent,
			&rep.ActionTaken,
			&rep.ReviewedBy,
			&rep.ReviewedAt,
			&rep.CreatedAt,
		); err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	return reports, nil
}

// ResolveReport updates report resolution status and action taken.
func (r *ModerationRepo) ResolveReport(ctx context.Context, reportID, status, actionTaken, adminID string) error {
	now := time.Now()
	query := `
		UPDATE content_reports
		SET status = $1, action_taken = $2, reviewed_by = $3, reviewed_at = $4
		WHERE id = $5
	`
	tag, err := r.db.Exec(ctx, query, status, actionTaken, adminID, now, reportID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrReportNotFound
	}
	return nil
}

// GetAdminStats calculates high-level operational statistics for the admin control center.
func (r *ModerationRepo) GetAdminStats(ctx context.Context) (*dto.AdminStatsResponse, error) {
	query := `
		SELECT 
			COUNT(*) AS total_shops,
			COUNT(*) FILTER (WHERE status = 'active') AS active_shops,
			COUNT(*) FILTER (WHERE status = 'pending_review') AS pending_shops,
			COUNT(*) FILTER (WHERE status = 'flagged') AS flagged_shops,
			COUNT(*) FILTER (WHERE status = 'banned') AS banned_shops,
			(SELECT COUNT(*) FROM content_reports WHERE status = 'pending') AS pending_reports,
			(SELECT COUNT(*) FROM content_reports) AS total_reports,
			(SELECT COUNT(*) FROM banned_entities) AS banned_entities,
			(SELECT COUNT(*) FROM users) AS total_users
		FROM shops;
	`
	stats := &dto.AdminStatsResponse{}
	err := r.db.QueryRow(ctx, query).Scan(
		&stats.TotalShops,
		&stats.ActiveShops,
		&stats.PendingShops,
		&stats.FlaggedShops,
		&stats.BannedShops,
		&stats.PendingReports,
		&stats.TotalReports,
		&stats.BannedEntities,
		&stats.TotalUsers,
	)
	if err != nil {
		r.logger.Error("failed to fetch admin stats", zap.Error(err))
		return nil, err
	}
	return stats, nil
}

// ListShopsForAdmin lists all shops with audit metadata for admin oversight.
func (r *ModerationRepo) ListShopsForAdmin(ctx context.Context, status, search string, limit, offset int) ([]*model.Shop, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if status != "" && status != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, strings.TrimSpace(status))
		argIdx++
	}

	if search != "" {
		searchTerm := "%" + strings.TrimSpace(search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR slug ILIKE $%d OR city ILIKE $%d OR phone ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		       latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		       COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		       COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		       is_open, is_active, status, flagged_count, COALESCE(suspension_reason, ''),
		       COALESCE(creation_ip, ''), COALESCE(creation_user_agent, ''), COALESCE(device_fingerprint, ''),
		       created_at, updated_at,
		       COUNT(*) OVER() AS total_count
		FROM shops
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query admin shops", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var shops []*model.Shop
	totalCount := 0

	for rows.Next() {
		s := &model.Shop{}
		var bannersBytes []byte
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Name,
			&s.Slug,
			&s.Description,
			&s.Category,
			&s.Phone,
			&s.Address,
			&s.Latitude,
			&s.Longitude,
			&s.City,
			&s.Pincode,
			&s.WhatsAppNumber,
			&s.LogoURL,
			&bannersBytes,
			&s.Timing,
			&s.OpeningTime,
			&s.ClosingTime,
			&s.WeeklyOff,
			&s.IsOpen,
			&s.IsActive,
			&s.Status,
			&s.FlaggedCount,
			&s.SuspensionReason,
			&s.CreationIP,
			&s.CreationUserAgent,
			&s.DeviceFingerprint,
			&s.CreatedAt,
			&s.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan admin shop row", zap.Error(err))
			return nil, 0, err
		}

		s.Banners = []string{}
		if len(bannersBytes) > 0 {
			_ = json.Unmarshal(bannersBytes, &s.Banners)
		}
		shops = append(shops, s)
	}

	return shops, totalCount, nil
}

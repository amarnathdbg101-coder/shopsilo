package repository

import (
	"context"
	"shopMe/internal/handler/dto"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type TelemetryRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewTelemetryRepo(db *pgxpool.Pool, logger *zap.Logger) *TelemetryRepo {
	return &TelemetryRepo{
		db:     db,
		logger: logger,
	}
}

func (r *TelemetryRepo) LogError(ctx context.Context, req dto.TelemetryErrorRequest) error {
	query := `
		INSERT INTO admin_system_errors (
			error_code, user_friendly_msg, developer_stack_trace, request_path, user_id, user_role, created_at, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW() + INTERVAL '24 hours')
	`
	code := strings.TrimSpace(req.ErrorCode)
	if code == "" {
		code = "SYS_ERR_500"
	}
	msg := strings.TrimSpace(req.UserFriendlyMsg)
	if msg == "" {
		msg = "Slight network glitch. Please tap retry!"
	}

	_, err := r.db.Exec(
		ctx,
		query,
		code,
		msg,
		strings.TrimSpace(req.DeveloperStackTrace),
		strings.TrimSpace(req.RequestPath),
		strings.TrimSpace(req.UserID),
		strings.TrimSpace(req.UserRole),
	)
	if err != nil {
		r.logger.Error("failed to record telemetry system error", zap.Error(err))
	}
	return err
}

func (r *TelemetryRepo) GetActive24HourErrors(ctx context.Context) ([]*dto.SystemErrorRecord, error) {
	// Auto-Purge expired logs older than 24 hours
	_, _ = r.db.Exec(ctx, `DELETE FROM admin_system_errors WHERE expires_at < NOW()`)

	query := `
		SELECT id, error_code, user_friendly_msg, developer_stack_trace, COALESCE(request_path, ''), COALESCE(user_id, ''), COALESCE(user_role, ''), created_at, expires_at
		FROM admin_system_errors
		WHERE expires_at >= NOW()
		ORDER BY created_at DESC
		LIMIT 150
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query active 24h system errors", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var records []*dto.SystemErrorRecord
	for rows.Next() {
		var rec dto.SystemErrorRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.ErrorCode,
			&rec.UserFriendlyMsg,
			&rec.DeveloperStackTrace,
			&rec.RequestPath,
			&rec.UserID,
			&rec.UserRole,
			&rec.CreatedAt,
			&rec.ExpiresAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}

	if records == nil {
		records = []*dto.SystemErrorRecord{}
	}

	return records, nil
}

func (r *TelemetryRepo) ClearAllErrors(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM admin_system_errors`)
	return err
}

package repository

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/model"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrPhoneVerificationNotFound = errors.New("phone verification record not found")
)

type PhoneVerificationRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPhoneVerificationRepo(db *pgxpool.Pool, logger *zap.Logger) *PhoneVerificationRepo {
	return &PhoneVerificationRepo{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new phone verification record with the hashed OTP and expiration.
func (r *PhoneVerificationRepo) Create(ctx context.Context, pv *model.PhoneVerification) error {
	query := `
		INSERT INTO phone_verifications (
			phone, otp_hash, attempts, max_attempts, is_verified, is_consumed, expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(
		ctx, query,
		pv.Phone,
		pv.OTPHash,
		pv.Attempts,
		pv.MaxAttempts,
		pv.IsVerified,
		pv.IsConsumed,
		pv.ExpiresAt,
	).Scan(&pv.ID, &pv.CreatedAt, &pv.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create phone verification record", zap.Error(err), zap.String("phone", pv.Phone))
		return fmt.Errorf("failed to create phone verification record: %w", err)
	}
	return nil
}

// GetLatestByPhone retrieves the most recent verification record for the given phone.
func (r *PhoneVerificationRepo) GetLatestByPhone(ctx context.Context, phone string) (*model.PhoneVerification, error) {
	query := `
		SELECT id, phone, otp_hash, attempts, max_attempts, is_verified, is_consumed,
		       expires_at, verified_at, consumed_at, created_at, updated_at
		FROM phone_verifications
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	pv := &model.PhoneVerification{}
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&pv.ID,
		&pv.Phone,
		&pv.OTPHash,
		&pv.Attempts,
		&pv.MaxAttempts,
		&pv.IsVerified,
		&pv.IsConsumed,
		&pv.ExpiresAt,
		&pv.VerifiedAt,
		&pv.ConsumedAt,
		&pv.CreatedAt,
		&pv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPhoneVerificationNotFound
		}
		r.logger.Error("failed to get latest phone verification", zap.Error(err), zap.String("phone", phone))
		return nil, err
	}
	return pv, nil
}

// CountRecentInWindow counts how many verification requests were created for the phone within the given time window.
// Used to enforce hourly throttling (e.g. max 5 OTP requests per hour).
func (r *PhoneVerificationRepo) CountRecentInWindow(ctx context.Context, phone string, window time.Duration) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM phone_verifications
		WHERE phone = $1 AND created_at >= NOW() - $2::interval
	`
	intervalStr := fmt.Sprintf("%d seconds", int(window.Seconds()))
	var count int
	err := r.db.QueryRow(ctx, query, phone, intervalStr).Scan(&count)
	if err != nil {
		r.logger.Error("failed to count recent phone verifications", zap.Error(err), zap.String("phone", phone))
		return 0, err
	}
	return count, nil
}

// IncrementAttempts increments the failed OTP attempt counter and returns the updated count.
func (r *PhoneVerificationRepo) IncrementAttempts(ctx context.Context, id string) (int, error) {
	query := `
		UPDATE phone_verifications
		SET attempts = attempts + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING attempts
	`
	var newAttempts int
	err := r.db.QueryRow(ctx, query, id).Scan(&newAttempts)
	if err != nil {
		r.logger.Error("failed to increment verification attempts", zap.Error(err), zap.String("id", id))
		return 0, err
	}
	return newAttempts, nil
}

// MarkVerified flags the record as verified and sets the verified_at timestamp.
func (r *PhoneVerificationRepo) MarkVerified(ctx context.Context, id string) error {
	query := `
		UPDATE phone_verifications
		SET is_verified = TRUE, verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to mark phone verification verified", zap.Error(err), zap.String("id", id))
		return err
	}
	return nil
}

// ConsumeVerification atomically marks a verified token record as consumed during user registration.
// Guarantees that each verified token can only ever be used once, preventing replay / double registration.
func (r *PhoneVerificationRepo) ConsumeVerification(ctx context.Context, id, phone string) (bool, error) {
	query := `
		UPDATE phone_verifications
		SET is_consumed = TRUE, consumed_at = NOW(), updated_at = NOW()
		WHERE id = $1
		  AND phone = $2
		  AND is_verified = TRUE
		  AND is_consumed = FALSE
		  AND expires_at > NOW()
	`
	cmdTag, err := r.db.Exec(ctx, query, id, phone)
	if err != nil {
		r.logger.Error("failed to consume phone verification", zap.Error(err), zap.String("id", id), zap.String("phone", phone))
		return false, err
	}
	return cmdTag.RowsAffected() > 0, nil
}

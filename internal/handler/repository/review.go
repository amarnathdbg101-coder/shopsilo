// Package repository handle db query.
package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrReviewNotFound = errors.New("review not found")
)

type ReviewRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewReviewRepo(db *pgxpool.Pool, logger *zap.Logger) *ReviewRepo {
	return &ReviewRepo{
		db:     db,
		logger: logger,
	}
}

// UpsertReview creates or updates a review and automatically checks if user is a verified in-store visitor in a single atomic query.
func (r *ReviewRepo) UpsertReview(ctx context.Context, shopID, userID string, rating int, comment string) (*model.ShopReview, error) {
	query := `
		WITH inserted AS (
			INSERT INTO shop_reviews (shop_id, user_id, rating, comment, is_verified_visitor, created_at, updated_at)
			VALUES (
				$1, 
				$2, 
				$3, 
				$4, 
				EXISTS (SELECT 1 FROM reservations WHERE shop_id = $1 AND user_id = $2 AND status = 'completed'), 
				NOW(), 
				NOW()
			)
			ON CONFLICT (shop_id, user_id)
			DO UPDATE SET
				rating = EXCLUDED.rating,
				comment = EXCLUDED.comment,
				is_verified_visitor = EXCLUDED.is_verified_visitor,
				updated_at = NOW()
			RETURNING id, shop_id, user_id, rating, COALESCE(comment, '') AS comment, is_verified_visitor, created_at, updated_at
		)
		SELECT i.id, i.shop_id, i.user_id, i.rating, i.comment, i.is_verified_visitor, i.created_at, i.updated_at, COALESCE(u.name, '')
		FROM inserted i
		LEFT JOIN users u ON u.id = i.user_id;
	`
	rev := &model.ShopReview{}
	err := r.db.QueryRow(ctx, query, shopID, userID, rating, strings.TrimSpace(comment)).Scan(
		&rev.ID,
		&rev.ShopID,
		&rev.UserID,
		&rev.Rating,
		&rev.Comment,
		&rev.IsVerifiedVisitor,
		&rev.CreatedAt,
		&rev.UpdatedAt,
		&rev.UserName,
	)
	if err != nil {
		r.logger.Error("failed to upsert review", zap.Error(err), zap.String("shop_id", shopID), zap.String("user_id", userID))
		return nil, err
	}

	return rev, nil
}

func (r *ReviewRepo) FindByShopID(ctx context.Context, shopID string, page, limit int) ([]*model.ShopReview, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := `
		SELECT r.id, r.shop_id, r.user_id, r.rating, COALESCE(r.comment, ''), r.is_verified_visitor,
		       r.created_at, r.updated_at, COALESCE(u.name, 'Customer') AS user_name,
		       COUNT(*) OVER() AS total_count
		FROM shop_reviews r
		JOIN users u ON r.user_id = u.id
		WHERE r.shop_id = $1
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, shopID, limit, offset)
	if err != nil {
		r.logger.Error("failed to query shop reviews", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var reviews []*model.ShopReview
	totalCount := 0

	for rows.Next() {
		rev := &model.ShopReview{}
		err := rows.Scan(
			&rev.ID,
			&rev.ShopID,
			&rev.UserID,
			&rev.Rating,
			&rev.Comment,
			&rev.IsVerifiedVisitor,
			&rev.CreatedAt,
			&rev.UpdatedAt,
			&rev.UserName,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan review row", zap.Error(err))
			return nil, 0, err
		}
		reviews = append(reviews, rev)
	}

	return reviews, totalCount, nil
}

func (r *ReviewRepo) GetShopRatingStats(ctx context.Context, shopID string) (*model.ShopRatingStats, error) {
	query := `
		SELECT COALESCE(ROUND(AVG(rating)::numeric, 1), 0.0), COUNT(*)
		FROM shop_reviews
		WHERE shop_id = $1
	`
	stats := &model.ShopRatingStats{}
	err := r.db.QueryRow(ctx, query, shopID).Scan(&stats.AverageRating, &stats.TotalReviews)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.ShopRatingStats{AverageRating: 0, TotalReviews: 0}, nil
		}
		return nil, err
	}
	return stats, nil
}

func (r *ReviewRepo) DeleteReview(ctx context.Context, shopID, userID string) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM shop_reviews WHERE shop_id = $1 AND user_id = $2`, shopID, userID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrReviewNotFound
	}
	return nil
}

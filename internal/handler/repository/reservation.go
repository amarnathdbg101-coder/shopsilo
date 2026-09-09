// Package repository handle db query.
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
	ErrReservationNotFound  = errors.New("reservation not found")
	ErrReservationExpired   = errors.New("reservation has expired")
	ErrReservationNotActive = errors.New("reservation is not active")
	ErrInsufficientStock    = errors.New("insufficient stock to reserve product")
)

type ReservationRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewReservationRepo(db *pgxpool.Pool, logger *zap.Logger) *ReservationRepo {
	return &ReservationRepo{
		db:     db,
		logger: logger,
	}
}

func (r *ReservationRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// ExpireStaleReservations updates all past-due active reservations and releases reserved inventory stock.
func (r *ReservationRepo) ExpireStaleReservations(ctx context.Context) error {
	query := `
		WITH expired_items AS (
			UPDATE reservations
			SET status = 'expired', updated_at = NOW()
			WHERE status = 'active' AND expires_at < NOW()
			RETURNING product_id, quantity
		),
		aggregated AS (
			SELECT product_id, SUM(quantity) AS total_qty
			FROM expired_items
			GROUP BY product_id
		)
		UPDATE inventory inv
		SET reserved_quantity = GREATEST(0, inv.reserved_quantity - agg.total_qty)
		FROM aggregated agg
		WHERE inv.product_id = agg.product_id;
	`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("failed to expire stale reservations", zap.Error(err))
		return err
	}
	return nil
}

func (r *ReservationRepo) CountActiveByUserID(ctx context.Context, userID string) (int, error) {
	_ = r.ExpireStaleReservations(ctx)
	query := `SELECT COUNT(*) FROM reservations WHERE user_id = $1 AND status = 'active'`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *ReservationRepo) CreateWithTx(ctx context.Context, tx pgx.Tx, res *model.Reservation) (*model.Reservation, error) {
	// 1. Atomically increment reserved inventory stock ONLY if available stock is sufficient
	invQuery := `
		UPDATE inventory
		SET reserved_quantity = reserved_quantity + $1, updated_at = NOW()
		WHERE product_id = $2 AND (quantity - reserved_quantity) >= $1
	`
	cmdTag, err := tx.Exec(ctx, invQuery, res.Quantity, res.ProductID)
	if err != nil {
		r.logger.Error("failed to atomically update reserved inventory", zap.Error(err), zap.String("product_id", res.ProductID))
		return nil, err
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, ErrInsufficientStock
	}

	query := `
		INSERT INTO reservations (
			reservation_number, user_id, shop_id, product_id, quantity, pickup_code,
			status, expires_at, notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, NOW(), NOW())
		RETURNING id, reservation_number, user_id, shop_id, product_id, quantity, pickup_code,
		          status, expires_at, completed_at, COALESCE(notes, ''), created_at, updated_at
	`
	created := &model.Reservation{}
	err = tx.QueryRow(
		ctx,
		query,
		res.ReservationNumber,
		res.UserID,
		res.ShopID,
		res.ProductID,
		res.Quantity,
		res.PickupCode,
		res.ExpiresAt,
		strings.TrimSpace(res.Notes),
	).Scan(
		&created.ID,
		&created.ReservationNumber,
		&created.UserID,
		&created.ShopID,
		&created.ProductID,
		&created.Quantity,
		&created.PickupCode,
		&created.Status,
		&created.ExpiresAt,
		&created.CompletedAt,
		&created.Notes,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("failed to insert reservation in tx", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (r *ReservationRepo) FindByID(ctx context.Context, id string) (*model.Reservation, error) {
	query := `
		SELECT r.id, r.reservation_number, r.user_id, r.shop_id, r.product_id, r.quantity, r.pickup_code,
		       r.status, r.expires_at, r.completed_at, COALESCE(r.notes, ''), r.created_at, r.updated_at,
		       s.name AS shop_name, s.slug AS shop_slug, s.phone AS shop_phone, s.address AS shop_address, s.latitude, s.longitude,
		       p.name AS product_name, p.slug AS product_slug, p.price AS product_price, p.images AS product_images
		FROM reservations r
		JOIN shops s ON r.shop_id = s.id
		JOIN products p ON r.product_id = p.id
		WHERE r.id = $1
		LIMIT 1
	`
	res := &model.Reservation{}
	res.Shop = &model.Shop{}
	res.Product = &model.Product{}
	var productImagesBytes []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.ID,
		&res.ReservationNumber,
		&res.UserID,
		&res.ShopID,
		&res.ProductID,
		&res.Quantity,
		&res.PickupCode,
		&res.Status,
		&res.ExpiresAt,
		&res.CompletedAt,
		&res.Notes,
		&res.CreatedAt,
		&res.UpdatedAt,
		&res.Shop.Name,
		&res.Shop.Slug,
		&res.Shop.Phone,
		&res.Shop.Address,
		&res.Shop.Latitude,
		&res.Shop.Longitude,
		&res.Product.Name,
		&res.Product.Slug,
		&res.Product.Price,
		&productImagesBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if len(productImagesBytes) > 0 {
		_ = json.Unmarshal(productImagesBytes, &res.Product.Images)
	}

	if res.Status == "active" {
		rem := int(time.Until(res.ExpiresAt).Minutes())
		if rem < 0 {
			rem = 0
		}
		res.TimeRemainingMinutes = rem
	}

	return res, nil
}

func (r *ReservationRepo) FindByUserID(ctx context.Context, userID string, filter dto.ReservationFilter) ([]*model.Reservation, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	var whereClauses []string
	args := []interface{}{userID}
	argIdx := 2

	whereClauses = append(whereClauses, "r.user_id = $1")

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Status))
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT r.id, r.reservation_number, r.user_id, r.shop_id, r.product_id, r.quantity, r.pickup_code,
		       r.status, r.expires_at, r.completed_at, COALESCE(r.notes, ''), r.created_at, r.updated_at,
		       s.name AS shop_name, s.slug AS shop_slug, s.phone AS shop_phone, s.address AS shop_address, s.latitude, s.longitude,
		       p.name AS product_name, p.slug AS product_slug, p.price AS product_price, p.images AS product_images,
		       COUNT(*) OVER() AS total_count
		FROM reservations r
		JOIN shops s ON r.shop_id = s.id
		JOIN products p ON r.product_id = p.id
		WHERE %s
		ORDER BY (r.status = 'active') DESC, r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query user reservations", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	reservations := make([]*model.Reservation, 0)
	totalCount := 0

	for rows.Next() {
		res := &model.Reservation{}
		res.Shop = &model.Shop{}
		res.Product = &model.Product{}
		var productImagesBytes []byte

		err := rows.Scan(
			&res.ID,
			&res.ReservationNumber,
			&res.UserID,
			&res.ShopID,
			&res.ProductID,
			&res.Quantity,
			&res.PickupCode,
			&res.Status,
			&res.ExpiresAt,
			&res.CompletedAt,
			&res.Notes,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.Shop.Name,
			&res.Shop.Slug,
			&res.Shop.Phone,
			&res.Shop.Address,
			&res.Shop.Latitude,
			&res.Shop.Longitude,
			&res.Product.Name,
			&res.Product.Slug,
			&res.Product.Price,
			&productImagesBytes,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan user reservation row", zap.Error(err))
			return nil, 0, err
		}

		if len(productImagesBytes) > 0 {
			_ = json.Unmarshal(productImagesBytes, &res.Product.Images)
		}

		if res.Status == "active" {
			rem := int(time.Until(res.ExpiresAt).Minutes())
			if rem < 0 {
				rem = 0
			}
			res.TimeRemainingMinutes = rem
		}

		reservations = append(reservations, res)
	}

	return reservations, totalCount, nil
}

func (r *ReservationRepo) FindByShopID(ctx context.Context, shopID string, filter dto.ReservationFilter) ([]*model.Reservation, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	var whereClauses []string
	args := []interface{}{shopID}
	argIdx := 2

	whereClauses = append(whereClauses, "r.shop_id = $1")

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("r.status = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Status))
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
		SELECT r.id, r.reservation_number, r.user_id, r.shop_id, r.product_id, r.quantity, r.pickup_code,
		       r.status, r.expires_at, r.completed_at, COALESCE(r.notes, ''), r.created_at, r.updated_at,
		       p.name AS product_name, p.slug AS product_slug, p.price AS product_price, p.images AS product_images,
		       COUNT(*) OVER() AS total_count
		FROM reservations r
		JOIN products p ON r.product_id = p.id
		WHERE %s
		ORDER BY (r.status = 'active') DESC, r.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query shop reservations", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	reservations := make([]*model.Reservation, 0)
	totalCount := 0

	for rows.Next() {
		res := &model.Reservation{}
		res.Product = &model.Product{}
		var productImagesBytes []byte

		err := rows.Scan(
			&res.ID,
			&res.ReservationNumber,
			&res.UserID,
			&res.ShopID,
			&res.ProductID,
			&res.Quantity,
			&res.PickupCode,
			&res.Status,
			&res.ExpiresAt,
			&res.CompletedAt,
			&res.Notes,
			&res.CreatedAt,
			&res.UpdatedAt,
			&res.Product.Name,
			&res.Product.Slug,
			&res.Product.Price,
			&productImagesBytes,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan shop reservation row", zap.Error(err))
			return nil, 0, err
		}

		if len(productImagesBytes) > 0 {
			_ = json.Unmarshal(productImagesBytes, &res.Product.Images)
		}

		if res.Status == "active" {
			rem := int(time.Until(res.ExpiresAt).Minutes())
			if rem < 0 {
				rem = 0
			}
			res.TimeRemainingMinutes = rem
		}

		reservations = append(reservations, res)
	}

	return reservations, totalCount, nil
}

func (r *ReservationRepo) VerifyAndCompleteWithTx(ctx context.Context, tx pgx.Tx, shopID, code, resNumber string) (*model.Reservation, error) {
	query := `
		UPDATE reservations
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE shop_id = $1
		  AND status = 'active'
		  AND expires_at >= NOW()
		  AND (pickup_code = $2 OR reservation_number = $3)
		RETURNING id, reservation_number, user_id, shop_id, product_id, quantity, pickup_code,
		          status, expires_at, completed_at, COALESCE(notes, ''), created_at, updated_at
	`
	res := &model.Reservation{}
	err := tx.QueryRow(ctx, query, shopID, strings.TrimSpace(code), strings.TrimSpace(resNumber)).Scan(
		&res.ID,
		&res.ReservationNumber,
		&res.UserID,
		&res.ShopID,
		&res.ProductID,
		&res.Quantity,
		&res.PickupCode,
		&res.Status,
		&res.ExpiresAt,
		&res.CompletedAt,
		&res.Notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	// Actual sale complete: reduce total inventory and reserved inventory
	invQuery := `
		UPDATE inventory
		SET quantity = GREATEST(0, quantity - $1),
		    reserved_quantity = GREATEST(0, reserved_quantity - $1)
		WHERE product_id = $2
	`
	_, _ = tx.Exec(ctx, invQuery, res.Quantity, res.ProductID)

	return res, nil
}

func (r *ReservationRepo) CancelWithTx(ctx context.Context, tx pgx.Tx, id string, userID, shopID string) (*model.Reservation, error) {
	var whereClause string
	args := []interface{}{id}

	if userID != "" {
		whereClause = "WHERE id = $1 AND user_id = $2 AND status = 'active'"
		args = append(args, userID)
	} else if shopID != "" {
		whereClause = "WHERE id = $1 AND shop_id = $2 AND status = 'active'"
		args = append(args, shopID)
	} else {
		whereClause = "WHERE id = $1 AND status = 'active'"
	}

	query := fmt.Sprintf(`
		UPDATE reservations
		SET status = 'cancelled', updated_at = NOW()
		%s
		RETURNING id, reservation_number, user_id, shop_id, product_id, quantity, pickup_code,
		          status, expires_at, completed_at, COALESCE(notes, ''), created_at, updated_at
	`, whereClause)

	res := &model.Reservation{}
	err := tx.QueryRow(ctx, query, args...).Scan(
		&res.ID,
		&res.ReservationNumber,
		&res.UserID,
		&res.ShopID,
		&res.ProductID,
		&res.Quantity,
		&res.PickupCode,
		&res.Status,
		&res.ExpiresAt,
		&res.CompletedAt,
		&res.Notes,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	// Release reserved inventory
	invQuery := `
		UPDATE inventory
		SET reserved_quantity = GREATEST(0, reserved_quantity - $1)
		WHERE product_id = $2
	`
	_, _ = tx.Exec(ctx, invQuery, res.Quantity, res.ProductID)

	return res, nil
}

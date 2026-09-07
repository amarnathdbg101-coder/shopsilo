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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrShopNotFound = errors.New("shop not found")
	ErrShopExists   = errors.New("user already owns a shop")
	ErrSlugTaken    = errors.New("shop slug is already in use")
)

func computeShopOpenStatus(s *model.Shop) {
	if s == nil {
		return
	}
	if !s.IsOpen || !s.IsActive {
		s.IsCurrentlyOpen = false
		return
	}

	now := time.Now()
	// Check weekly off day
	if s.WeeklyOff != "" && strings.EqualFold(now.Weekday().String(), strings.TrimSpace(s.WeeklyOff)) {
		s.IsCurrentlyOpen = false
		return
	}

	// Check opening and closing times if configured (expected format HH:MM, e.g. "09:00", "21:30")
	if s.OpeningTime != "" && s.ClosingTime != "" {
		curMinutes := now.Hour()*60 + now.Minute()
		var openH, openM, closeH, closeM int
		if n, _ := fmt.Sscanf(s.OpeningTime, "%d:%d", &openH, &openM); n == 2 {
			if n2, _ := fmt.Sscanf(s.ClosingTime, "%d:%d", &closeH, &closeM); n2 == 2 {
				openMinutes := openH*60 + openM
				closeMinutes := closeH*60 + closeM
				if openMinutes < closeMinutes {
					if curMinutes < openMinutes || curMinutes >= closeMinutes {
						s.IsCurrentlyOpen = false
						return
					}
				}
			}
		}
	}

	s.IsCurrentlyOpen = true
}

type ShopRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewShopRepo(db *pgxpool.Pool, logger *zap.Logger) *ShopRepo {
	return &ShopRepo{
		db:     db,
		logger: logger,
	}
}

func (r *ShopRepo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *ShopRepo) CreateWithTx(ctx context.Context, tx pgx.Tx, s *model.Shop) (*model.Shop, error) {
	bannersJSON, _ := json.Marshal(s.Banners)
	if s.Banners == nil {
		bannersJSON = []byte("[]")
	}

	openingTime := s.OpeningTime
	if openingTime == "" {
		openingTime = "09:00"
	}
	closingTime := s.ClosingTime
	if closingTime == "" {
		closingTime = "21:00"
	}

	query := `
		INSERT INTO shops (
			user_id, name, slug, description, category, phone, address,
			latitude, longitude, city, pincode, whatsapp_number,
			logo_url, banners, timing, opening_time, closing_time, weekly_off, is_open, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())
		RETURNING id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		          latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		          COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		          COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		          is_open, is_active, created_at, updated_at
	`
	created := &model.Shop{}
	var bannersBytes []byte

	err := tx.QueryRow(
		ctx,
		query,
		s.UserID,
		strings.TrimSpace(s.Name),
		strings.TrimSpace(s.Slug),
		strings.TrimSpace(s.Description),
		strings.TrimSpace(s.Category),
		strings.TrimSpace(s.Phone),
		strings.TrimSpace(s.Address),
		s.Latitude,
		s.Longitude,
		strings.TrimSpace(s.City),
		strings.TrimSpace(s.Pincode),
		strings.TrimSpace(s.WhatsAppNumber),
		strings.TrimSpace(s.LogoURL),
		bannersJSON,
		strings.TrimSpace(s.Timing),
		openingTime,
		closingTime,
		strings.TrimSpace(s.WeeklyOff),
		s.IsOpen,
		s.IsActive,
	).Scan(
		&created.ID,
		&created.UserID,
		&created.Name,
		&created.Slug,
		&created.Description,
		&created.Category,
		&created.Phone,
		&created.Address,
		&created.Latitude,
		&created.Longitude,
		&created.City,
		&created.Pincode,
		&created.WhatsAppNumber,
		&created.LogoURL,
		&bannersBytes,
		&created.Timing,
		&created.OpeningTime,
		&created.ClosingTime,
		&created.WeeklyOff,
		&created.IsOpen,
		&created.IsActive,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "slug") {
				return nil, ErrSlugTaken
			}
			return nil, ErrShopExists
		}
		r.logger.Error("failed to create shop in tx", zap.Error(err), zap.String("user_id", s.UserID))
		return nil, err
	}

	created.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &created.Banners)
	}

	computeShopOpenStatus(created)
	return created, nil
}

func (r *ShopRepo) FindByUserID(ctx context.Context, userID string) (*model.Shop, error) {
	query := `
		SELECT id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		       latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		       COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		       COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		       is_open, is_active, created_at, updated_at
		FROM shops
		WHERE user_id = $1
		LIMIT 1
	`
	s := &model.Shop{}
	var bannersBytes []byte

	err := r.db.QueryRow(ctx, query, userID).Scan(
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
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShopNotFound
		}
		r.logger.Error("failed to find shop by user_id", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}

	s.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &s.Banners)
	}

	computeShopOpenStatus(s)
	return s, nil
}

func (r *ShopRepo) FindByID(ctx context.Context, id string) (*model.Shop, error) {
	query := `
		SELECT id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		       latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		       COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		       COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		       is_open, is_active, created_at, updated_at
		FROM shops
		WHERE id = $1 AND is_active = true
		LIMIT 1
	`
	s := &model.Shop{}
	var bannersBytes []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
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
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShopNotFound
		}
		r.logger.Error("failed to find shop by id", zap.Error(err), zap.String("id", id))
		return nil, err
	}

	s.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &s.Banners)
	}

	computeShopOpenStatus(s)
	return s, nil
}

func (r *ShopRepo) FindBySlug(ctx context.Context, slug string) (*model.Shop, error) {
	query := `
		SELECT id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		       latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		       COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		       COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		       is_open, is_active, created_at, updated_at
		FROM shops
		WHERE slug = LOWER(TRIM($1)) AND is_active = true
		LIMIT 1
	`
	s := &model.Shop{}
	var bannersBytes []byte

	err := r.db.QueryRow(ctx, query, slug).Scan(
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
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShopNotFound
		}
		r.logger.Error("failed to find shop by slug", zap.Error(err), zap.String("slug", slug))
		return nil, err
	}

	s.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &s.Banners)
	}

	computeShopOpenStatus(s)
	return s, nil
}

func (r *ShopRepo) FindAll(ctx context.Context, filter dto.ShopFilter) ([]*model.Shop, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var innerWhereClauses []string
	var args []interface{}
	argIdx := 1

	innerWhereClauses = append(innerWhereClauses, "is_active = true")

	if filter.Search != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		innerWhereClauses = append(innerWhereClauses, fmt.Sprintf("(name ILIKE $%d OR category ILIKE $%d OR description ILIKE $%d OR slug ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	if filter.Category != "" {
		innerWhereClauses = append(innerWhereClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Category))
		argIdx++
	}

	if filter.City != "" {
		innerWhereClauses = append(innerWhereClauses, fmt.Sprintf("city ILIKE $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.City))
		argIdx++
	}

	if filter.Pincode != "" {
		innerWhereClauses = append(innerWhereClauses, fmt.Sprintf("pincode = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Pincode))
		argIdx++
	}

	hasGeo := filter.Lat != nil && filter.Lng != nil
	var distanceExpr string

	if hasGeo {
		latIdx := argIdx
		lngIdx := argIdx + 1
		distanceExpr = fmt.Sprintf(`
			CASE
				WHEN latitude IS NOT NULL AND longitude IS NOT NULL THEN
					(6371 * acos(LEAST(1.0, GREATEST(-1.0,
						cos(radians($%d)) * cos(radians(latitude)) * cos(radians(longitude) - radians($%d)) +
						sin(radians($%d)) * sin(radians(latitude))
					))))
				ELSE NULL
			END
		`, latIdx, lngIdx, latIdx)
		args = append(args, *filter.Lat, *filter.Lng)
		argIdx += 2
	} else {
		distanceExpr = "NULL::DOUBLE PRECISION"
	}

	innerWhereSQL := strings.Join(innerWhereClauses, " AND ")

	var outerWhereClauses []string
	if hasGeo && filter.RadiusKm != nil && *filter.RadiusKm > 0 {
		outerWhereClauses = append(outerWhereClauses, fmt.Sprintf("distance_km IS NOT NULL AND distance_km <= $%d", argIdx))
		args = append(args, *filter.RadiusKm)
		argIdx++
	}

	outerWhereSQL := ""
	if len(outerWhereClauses) > 0 {
		outerWhereSQL = "WHERE " + strings.Join(outerWhereClauses, " AND ")
	}

	orderBy := "is_open DESC, created_at DESC"
	if hasGeo {
		orderBy = "is_open DESC, (distance_km IS NULL) ASC, distance_km ASC, created_at DESC"
	}

	query := fmt.Sprintf(`
		WITH filtered_shops AS (
			SELECT id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
			       latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
			       COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
			       COALESCE(opening_time, '09:00') AS opening_time, COALESCE(closing_time, '21:00') AS closing_time, COALESCE(weekly_off, '') AS weekly_off,
			       is_open, is_active, created_at, updated_at,
			       %s AS distance_km
			FROM shops
			WHERE %s
		)
		SELECT id, user_id, name, slug, description, category, phone, address,
		       latitude, longitude, city, pincode, whatsapp_number,
		       logo_url, banners, timing, opening_time, closing_time, weekly_off,
		       is_open, is_active, created_at, updated_at,
		       distance_km,
		       COUNT(*) OVER() AS total_count
		FROM filtered_shops
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, distanceExpr, innerWhereSQL, outerWhereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query shops", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var shops []*model.Shop
	totalCount := 0

	for rows.Next() {
		s := &model.Shop{}
		var bannersBytes []byte
		var distKm *float64

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
			&s.CreatedAt,
			&s.UpdatedAt,
			&distKm,
			&totalCount,
		)
		if err != nil {
			r.logger.Error("failed to scan shop row", zap.Error(err))
			return nil, 0, err
		}

		s.DistanceKm = distKm
		s.Banners = []string{}
		if len(bannersBytes) > 0 {
			_ = json.Unmarshal(bannersBytes, &s.Banners)
		}

		computeShopOpenStatus(s)
		shops = append(shops, s)
	}

	return shops, totalCount, nil
}

func (r *ShopRepo) Update(ctx context.Context, s *model.Shop) (*model.Shop, error) {
	bannersJSON, _ := json.Marshal(s.Banners)
	if s.Banners == nil {
		bannersJSON = []byte("[]")
	}

	query := `
		UPDATE shops
		SET name = $1, slug = $2, description = $3, category = $4, phone = $5, address = $6,
		    latitude = $7, longitude = $8, city = $9, pincode = $10, whatsapp_number = $11,
		    logo_url = $12, banners = $13, timing = $14,
		    opening_time = CASE WHEN $15 != '' THEN $15 ELSE opening_time END,
		    closing_time = CASE WHEN $16 != '' THEN $16 ELSE closing_time END,
		    weekly_off = CASE WHEN $17 != '' THEN $17 ELSE weekly_off END,
		    is_open = $18, updated_at = NOW()
		WHERE id = $19 AND user_id = $20
		RETURNING id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		          latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		          COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		          COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		          is_open, is_active, created_at, updated_at
	`
	updated := &model.Shop{}
	var bannersBytes []byte

	err := r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(s.Name),
		strings.TrimSpace(s.Slug),
		strings.TrimSpace(s.Description),
		strings.TrimSpace(s.Category),
		strings.TrimSpace(s.Phone),
		strings.TrimSpace(s.Address),
		s.Latitude,
		s.Longitude,
		strings.TrimSpace(s.City),
		strings.TrimSpace(s.Pincode),
		strings.TrimSpace(s.WhatsAppNumber),
		strings.TrimSpace(s.LogoURL),
		bannersJSON,
		strings.TrimSpace(s.Timing),
		strings.TrimSpace(s.OpeningTime),
		strings.TrimSpace(s.ClosingTime),
		strings.TrimSpace(s.WeeklyOff),
		s.IsOpen,
		s.ID,
		s.UserID,
	).Scan(
		&updated.ID,
		&updated.UserID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.Category,
		&updated.Phone,
		&updated.Address,
		&updated.Latitude,
		&updated.Longitude,
		&updated.City,
		&updated.Pincode,
		&updated.WhatsAppNumber,
		&updated.LogoURL,
		&bannersBytes,
		&updated.Timing,
		&updated.OpeningTime,
		&updated.ClosingTime,
		&updated.WeeklyOff,
		&updated.IsOpen,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "slug") {
			return nil, ErrSlugTaken
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShopNotFound
		}
		r.logger.Error("failed to update shop", zap.Error(err), zap.String("shop_id", s.ID))
		return nil, err
	}

	updated.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &updated.Banners)
	}

	computeShopOpenStatus(updated)
	return updated, nil
}

func (r *ShopRepo) ToggleStatus(ctx context.Context, userID string, isOpen bool) (*model.Shop, error) {
	query := `
		UPDATE shops
		SET is_open = $1, updated_at = NOW()
		WHERE user_id = $2
		RETURNING id, user_id, name, slug, COALESCE(description, ''), COALESCE(category, ''), COALESCE(phone, ''), COALESCE(address, ''),
		          latitude, longitude, COALESCE(city, ''), COALESCE(pincode, ''), COALESCE(whatsapp_number, ''),
		          COALESCE(logo_url, ''), banners, COALESCE(timing, ''),
		          COALESCE(opening_time, '09:00'), COALESCE(closing_time, '21:00'), COALESCE(weekly_off, ''),
		          is_open, is_active, created_at, updated_at
	`
	updated := &model.Shop{}
	var bannersBytes []byte

	err := r.db.QueryRow(ctx, query, isOpen, userID).Scan(
		&updated.ID,
		&updated.UserID,
		&updated.Name,
		&updated.Slug,
		&updated.Description,
		&updated.Category,
		&updated.Phone,
		&updated.Address,
		&updated.Latitude,
		&updated.Longitude,
		&updated.City,
		&updated.Pincode,
		&updated.WhatsAppNumber,
		&updated.LogoURL,
		&bannersBytes,
		&updated.Timing,
		&updated.OpeningTime,
		&updated.ClosingTime,
		&updated.WeeklyOff,
		&updated.IsOpen,
		&updated.IsActive,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShopNotFound
		}
		r.logger.Error("failed to toggle shop status", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}

	updated.Banners = []string{}
	if len(bannersBytes) > 0 {
		_ = json.Unmarshal(bannersBytes, &updated.Banners)
	}

	computeShopOpenStatus(updated)
	return updated, nil
}

func (r *ShopRepo) DeleteWithTx(ctx context.Context, tx pgx.Tx, userID string) error {
	query := `
		DELETE FROM shops
		WHERE user_id = $1
	`
	cmdTag, err := tx.Exec(ctx, query, userID)
	if err != nil {
		r.logger.Error("failed to delete shop in tx", zap.Error(err), zap.String("user_id", userID))
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrShopNotFound
	}
	return nil
}

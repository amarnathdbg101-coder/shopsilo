// Package repository handles database queries for Scan & Go self checkout.
package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrScanSessionNotFound = errors.New("scan and go session not found or expired")
	ErrScanSessionInactive = errors.New("scan and go session is no longer active")
	ErrItemNotInCart       = errors.New("item not found in cart session")
)

// ScanAndGoRepo manages database operations for customer self-checkout carts.
type ScanAndGoRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewScanAndGoRepo initializes ScanAndGoRepo.
func NewScanAndGoRepo(db *pgxpool.Pool, logger *zap.Logger) *ScanAndGoRepo {
	return &ScanAndGoRepo{
		db:     db,
		logger: logger,
	}
}

// GenerateSessionCode generates a secure random 6-digit number string (e.g. 849201).
func (r *ScanAndGoRepo) GenerateSessionCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// GenerateQRToken generates a 32-byte hex token.
func (r *ScanAndGoRepo) GenerateQRToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateSession initializes a new scan and go session in the database.
func (r *ScanAndGoRepo) CreateSession(ctx context.Context, s *model.ScanAndGoSession) (*model.ScanAndGoSession, error) {
	if s.SessionCode == "" {
		s.SessionCode = r.GenerateSessionCode()
	}
	if s.QRToken == "" {
		s.QRToken = r.GenerateQRToken()
	}
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt = time.Now().Add(2 * time.Hour) // Valid for 2 hours while in store
	}
	s.Status = model.ScanAndGoActive

	query := `
		INSERT INTO scan_and_go_sessions (
			shop_id, customer_user_id, customer_name, customer_phone,
			session_code, qr_token, status, total_items, subtotal, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, 0.00, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`
	err := r.db.QueryRow(ctx, query,
		s.ShopID, s.CustomerUserID, s.CustomerName, s.CustomerPhone,
		s.SessionCode, s.QRToken, s.Status, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create scan and go session", zap.Error(err), zap.String("shop_id", s.ShopID))
		return nil, err
	}
	return s, nil
}

// GetSessionByID fetches session by UUID.
func (r *ScanAndGoRepo) GetSessionByID(ctx context.Context, sessionID string) (*model.ScanAndGoSession, error) {
	query := `
		SELECT id, shop_id, customer_user_id, customer_name, customer_phone,
		       session_code, qr_token, status, total_items, subtotal, expires_at,
		       converted_bill_id, created_at, updated_at
		FROM scan_and_go_sessions
		WHERE id = $1;
	`
	s := &model.ScanAndGoSession{}
	err := r.db.QueryRow(ctx, query, sessionID).Scan(
		&s.ID, &s.ShopID, &s.CustomerUserID, &s.CustomerName, &s.CustomerPhone,
		&s.SessionCode, &s.QRToken, &s.Status, &s.TotalItems, &s.Subtotal, &s.ExpiresAt,
		&s.ConvertedBillID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScanSessionNotFound
		}
		return nil, err
	}
	return s, nil
}

// GetSessionByCode looks up session by 6-digit code or QR token.
func (r *ScanAndGoRepo) GetSessionByCode(ctx context.Context, shopID, codeOrToken string) (*model.ScanAndGoSession, error) {
	query := `
		SELECT id, shop_id, customer_user_id, customer_name, customer_phone,
		       session_code, qr_token, status, total_items, subtotal, expires_at,
		       converted_bill_id, created_at, updated_at
		FROM scan_and_go_sessions
		WHERE shop_id = $1 AND (session_code = $2 OR qr_token = $2);
	`
	s := &model.ScanAndGoSession{}
	err := r.db.QueryRow(ctx, query, shopID, codeOrToken).Scan(
		&s.ID, &s.ShopID, &s.CustomerUserID, &s.CustomerName, &s.CustomerPhone,
		&s.SessionCode, &s.QRToken, &s.Status, &s.TotalItems, &s.Subtotal, &s.ExpiresAt,
		&s.ConvertedBillID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScanSessionNotFound
		}
		return nil, err
	}
	return s, nil
}

// AddOrUpdateItem adds an item or increments its quantity, recalculating session subtotal.
func (r *ScanAndGoRepo) AddOrUpdateItem(ctx context.Context, sessionID string, item *model.ScanAndGoItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Check if session is still active
	var status model.ScanAndGoStatus
	var expiresAt time.Time
	err = tx.QueryRow(ctx, "SELECT status, expires_at FROM scan_and_go_sessions WHERE id = $1 FOR UPDATE;", sessionID).
		Scan(&status, &expiresAt)
	if err != nil {
		return err
	}
	if status != model.ScanAndGoActive || time.Now().After(expiresAt) {
		return ErrScanSessionInactive
	}

	// Check if this product/variant already exists in cart
	var existingItemID string
	var existingQty int
	checkQuery := `
		SELECT id, quantity FROM scan_and_go_items 
		WHERE session_id = $1 AND product_id = $2 
		  AND ((variant_id IS NULL AND $3::uuid IS NULL) OR variant_id = $3::uuid)
		LIMIT 1;
	`
	err = tx.QueryRow(ctx, checkQuery, sessionID, item.ProductID, item.VariantID).Scan(&existingItemID, &existingQty)
	if err == nil {
		// Update existing item
		newQty := existingQty + item.Quantity
		newTotal := float64(newQty) * item.UnitPrice
		_, err = tx.Exec(ctx, `
			UPDATE scan_and_go_items 
			SET quantity = $1, total_price = $2 
			WHERE id = $3;
		`, newQty, newTotal, existingItemID)
		if err != nil {
			return err
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		// Insert new item
		itemTotal := float64(item.Quantity) * item.UnitPrice
		insertQuery := `
			INSERT INTO scan_and_go_items (
				session_id, product_id, variant_id, product_name, barcode, quantity, unit_price, total_price, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			RETURNING id, created_at;
		`
		err = tx.QueryRow(ctx, insertQuery,
			sessionID, item.ProductID, item.VariantID, item.ProductName, item.Barcode,
			item.Quantity, item.UnitPrice, itemTotal,
		).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return err
		}
	} else {
		return err
	}

	// Recalculate session totals
	_, err = tx.Exec(ctx, `
		UPDATE scan_and_go_sessions
		SET total_items = (SELECT COALESCE(SUM(quantity), 0) FROM scan_and_go_items WHERE session_id = $1),
		    subtotal = (SELECT COALESCE(SUM(total_price), 0.00) FROM scan_and_go_items WHERE session_id = $1),
		    updated_at = NOW()
		WHERE id = $1;
	`, sessionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateItemQuantity modifies or removes an item.
func (r *ScanAndGoRepo) UpdateItemQuantity(ctx context.Context, sessionID, itemID string, quantity int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if quantity <= 0 {
		_, err = tx.Exec(ctx, "DELETE FROM scan_and_go_items WHERE id = $1 AND session_id = $2;", itemID, sessionID)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE scan_and_go_items
			SET quantity = $1, total_price = $1 * unit_price
			WHERE id = $2 AND session_id = $3;
		`, quantity, itemID, sessionID)
		if err != nil {
			return err
		}
	}

	// Update session totals
	_, err = tx.Exec(ctx, `
		UPDATE scan_and_go_sessions
		SET total_items = (SELECT COALESCE(SUM(quantity), 0) FROM scan_and_go_items WHERE session_id = $1),
		    subtotal = (SELECT COALESCE(SUM(total_price), 0.00) FROM scan_and_go_items WHERE session_id = $1),
		    updated_at = NOW()
		WHERE id = $1;
	`, sessionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteItem removes an item from cart.
func (r *ScanAndGoRepo) DeleteItem(ctx context.Context, sessionID, itemID string) error {
	return r.UpdateItemQuantity(ctx, sessionID, itemID, 0)
}

// ListItems returns all items inside a cart session.
func (r *ScanAndGoRepo) ListItems(ctx context.Context, sessionID string) ([]*model.ScanAndGoItem, error) {
	query := `
		SELECT id, session_id, product_id, variant_id, product_name, barcode, quantity, unit_price, total_price, created_at
		FROM scan_and_go_items
		WHERE session_id = $1
		ORDER BY created_at ASC;
	`
	rows, err := r.db.Query(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.ScanAndGoItem, 0)
	for rows.Next() {
		it := &model.ScanAndGoItem{}
		err := rows.Scan(
			&it.ID, &it.SessionID, &it.ProductID, &it.VariantID, &it.ProductName,
			&it.Barcode, &it.Quantity, &it.UnitPrice, &it.TotalPrice, &it.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

// MarkConverted sets the session as converted to official POS bill.
func (r *ScanAndGoRepo) MarkConverted(ctx context.Context, sessionID, billID string) error {
	query := `
		UPDATE scan_and_go_sessions
		SET status = 'CONVERTED_TO_BILL', converted_bill_id = $1, updated_at = NOW()
		WHERE id = $2;
	`
	_, err := r.db.Exec(ctx, query, billID, sessionID)
	return err
}


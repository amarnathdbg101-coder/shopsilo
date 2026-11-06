// Package repository handles database queries for cash drawer sessions and events.
package repository

import (
	"context"
	"errors"
	"math"
	"time"

	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrNoActiveDrawerSession = errors.New("no active cash drawer session found")
	ErrDrawerSessionExists   = errors.New("a cash drawer session is already open for this counter")
)

// isValidUUID checks if a string is a standard 36-char UUID (8-4-4-4-12).
func isValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	for i, r := range u {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

// CashDrawerRepo manages cash drawer shifts and physical drawer telemetry.
type CashDrawerRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewCashDrawerRepo initializes CashDrawerRepo.
func NewCashDrawerRepo(db *pgxpool.Pool, logger *zap.Logger) *CashDrawerRepo {
	return &CashDrawerRepo{
		db:     db,
		logger: logger,
	}
}

// GetActiveSession finds the open shift session for a shop.
func (r *CashDrawerRepo) GetActiveSession(ctx context.Context, shopID, staffUserID string) (*model.CashDrawerSession, error) {
	query := `
		SELECT s.id, s.shop_id, s.staff_user_id, COALESCE(u.full_name, 'Staff'), s.shift_name,
		       s.opened_at, s.closed_at, s.opening_cash, s.cash_sales_total, s.safe_drop_total,
		       s.expenses_total, s.expected_cash, s.actual_cash, s.discrepancy_amount,
		       s.safe_drop_threshold, s.status, COALESCE(s.closing_notes, ''), s.created_at, s.updated_at
		FROM cash_drawer_sessions s
		JOIN users u ON u.id = s.staff_user_id
		WHERE s.shop_id = $1 AND s.status = 'OPEN'
		ORDER BY s.opened_at DESC
		LIMIT 1;
	`
	sess := &model.CashDrawerSession{}
	err := r.db.QueryRow(ctx, query, shopID).Scan(
		&sess.ID, &sess.ShopID, &sess.StaffUserID, &sess.StaffName, &sess.ShiftName,
		&sess.OpenedAt, &sess.ClosedAt, &sess.OpeningCash, &sess.CashSalesTotal, &sess.SafeDropTotal,
		&sess.ExpensesTotal, &sess.ExpectedCash, &sess.ActualCash, &sess.DiscrepancyAmount,
		&sess.SafeDropThreshold, &sess.Status, &sess.ClosingNotes, &sess.CreatedAt, &sess.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveDrawerSession
		}
		r.logger.Error("failed to find active cash drawer session", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	return sess, nil
}

// OpenSession creates a new shift and logs the initial opening float.
func (r *CashDrawerRepo) OpenSession(ctx context.Context, s *model.CashDrawerSession) (*model.CashDrawerSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Check if already open
	var existingID string
	err = tx.QueryRow(ctx, "SELECT id FROM cash_drawer_sessions WHERE shop_id = $1 AND status = 'OPEN' LIMIT 1;", s.ShopID).Scan(&existingID)
	if err == nil {
		return nil, ErrDrawerSessionExists
	}

	threshold := s.SafeDropThreshold
	if threshold <= 0 {
		threshold = 10000.00
	}
	shiftName := s.ShiftName
	if shiftName == "" {
		shiftName = "General Shift"
	}

	s.ExpectedCash = s.OpeningCash
	s.SafeDropThreshold = threshold
	s.ShiftName = shiftName
	s.Status = model.DrawerStatusOpen

	query := `
		INSERT INTO cash_drawer_sessions (
			shop_id, staff_user_id, shift_name, opened_at, opening_cash, cash_sales_total,
			safe_drop_total, expenses_total, expected_cash, safe_drop_threshold, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, NOW(), $4, 0, 0, 0, $4, $5, 'OPEN', NOW(), NOW())
		RETURNING id, opened_at, created_at, updated_at;
	`
	err = tx.QueryRow(ctx, query, s.ShopID, s.StaffUserID, s.ShiftName, s.OpeningCash, s.SafeDropThreshold).
		Scan(&s.ID, &s.OpenedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to insert cash drawer session", zap.Error(err))
		return nil, err
	}

	// Insert opening float event
	evQuery := `
		INSERT INTO cash_drawer_events (
			session_id, shop_id, staff_user_id, event_type, amount, category, reason, created_at
		)
		VALUES ($1, $2, $3, 'OPENING_FLOAT', $4, 'Opening Float', 'Counter Shift Started', NOW());
	`
	_, err = tx.Exec(ctx, evQuery, s.ID, s.ShopID, s.StaffUserID, s.OpeningCash)
	if err != nil {
		r.logger.Error("failed to record opening float event", zap.Error(err))
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// AddCashSale adds cash tender from a completed POS bill to the active drawer session.
func (r *CashDrawerRepo) AddCashSale(ctx context.Context, shopID string, billID string, staffUserID string, cashAmount float64) error {
	if cashAmount <= 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var sessionID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM cash_drawer_sessions 
		WHERE shop_id = $1 AND status = 'OPEN' 
		ORDER BY opened_at DESC LIMIT 1 FOR UPDATE;
	`, shopID).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No open drawer session currently, ignore gracefully
			return nil
		}
		return err
	}

	// Increment cash sales & expected cash
	_, err = tx.Exec(ctx, `
		UPDATE cash_drawer_sessions
		SET cash_sales_total = cash_sales_total + $1,
		    expected_cash = expected_cash + $1,
		    updated_at = NOW()
		WHERE id = $2;
	`, cashAmount, sessionID)
	if err != nil {
		return err
	}

	// Log event
	var billParam any
	if isValidUUID(billID) {
		billParam = billID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO cash_drawer_events (
			session_id, shop_id, staff_user_id, event_type, amount, bill_id, category, reason, created_at
		)
		VALUES ($1, $2, $3, 'CASH_SALE', $4, $5, 'POS Sale', 'Counter Cash Received', NOW());
	`, sessionID, shopID, staffUserID, cashAmount, billParam)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ApplySafeDrop registers a safe drop deduction and moves money to secure storage.
func (r *CashDrawerRepo) ApplySafeDrop(ctx context.Context, sessionID, shopID, staffUserID string, amount float64, notes string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE cash_drawer_sessions
		SET safe_drop_total = safe_drop_total + $1,
		    expected_cash = expected_cash - $1,
		    updated_at = NOW()
		WHERE id = $2 AND shop_id = $3 AND status = 'OPEN';
	`, amount, sessionID, shopID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoActiveDrawerSession
	}

	reason := notes
	if reason == "" {
		reason = "Surplus cash deposited into safe vault"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO cash_drawer_events (
			session_id, shop_id, staff_user_id, event_type, amount, category, reason, created_at
		)
		VALUES ($1, $2, $3, 'SAFE_DROP', $4, 'Safe Vault', $5, NOW());
	`, sessionID, shopID, staffUserID, amount, reason)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ApplyPettyExpense records an instant drawer withdrawal for store operational expenses.
func (r *CashDrawerRepo) ApplyPettyExpense(ctx context.Context, sessionID, shopID, staffUserID string, amount float64, category, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE cash_drawer_sessions
		SET expenses_total = expenses_total + $1,
		    expected_cash = expected_cash - $1,
		    updated_at = NOW()
		WHERE id = $2 AND shop_id = $3 AND status = 'OPEN';
	`, amount, sessionID, shopID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoActiveDrawerSession
	}

	if category == "" {
		category = "Miscellaneous"
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO cash_drawer_events (
			session_id, shop_id, staff_user_id, event_type, amount, category, reason, created_at
		)
		VALUES ($1, $2, $3, 'PETTY_EXPENSE', $4, $5, $6, NOW());
	`, sessionID, shopID, staffUserID, amount, category, reason)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CloseSession reconciles actual vs expected cash and closes the shift.
func (r *CashDrawerRepo) CloseSession(ctx context.Context, sessionID, shopID, staffUserID string, actualCash float64, notes string) (*model.CashDrawerSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	sess := &model.CashDrawerSession{}
	err = tx.QueryRow(ctx, `
		SELECT id, shop_id, staff_user_id, shift_name, opened_at, opening_cash,
		       cash_sales_total, safe_drop_total, expenses_total, expected_cash, safe_drop_threshold
		FROM cash_drawer_sessions
		WHERE id = $1 AND shop_id = $2 AND status = 'OPEN'
		FOR UPDATE;
	`, sessionID, shopID).Scan(
		&sess.ID, &sess.ShopID, &sess.StaffUserID, &sess.ShiftName, &sess.OpenedAt,
		&sess.OpeningCash, &sess.CashSalesTotal, &sess.SafeDropTotal, &sess.ExpensesTotal,
		&sess.ExpectedCash, &sess.SafeDropThreshold,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveDrawerSession
		}
		return nil, err
	}

	discrepancy := actualCash - sess.ExpectedCash
	status := model.DrawerStatusClosed
	if math.Abs(discrepancy) > 0.01 {
		status = model.DrawerStatusDiscrepancy
	}

	now := time.Now()
	sess.ActualCash = &actualCash
	sess.DiscrepancyAmount = discrepancy
	sess.Status = status
	sess.ClosingNotes = notes
	sess.ClosedAt = &now

	_, err = tx.Exec(ctx, `
		UPDATE cash_drawer_sessions
		SET actual_cash = $1,
		    discrepancy_amount = $2,
		    status = $3,
		    closing_notes = $4,
		    closed_at = $5,
		    updated_at = NOW()
		WHERE id = $6;
	`, actualCash, discrepancy, status, notes, now, sessionID)
	if err != nil {
		return nil, err
	}

	// Insert shift close event
	_, err = tx.Exec(ctx, `
		INSERT INTO cash_drawer_events (
			session_id, shop_id, staff_user_id, event_type, amount, category, reason, created_at
		)
		VALUES ($1, $2, $3, 'SHIFT_CLOSE', $4, 'Shift Reconciliation', $5, NOW());
	`, sessionID, shopID, staffUserID, actualCash, notes)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return sess, nil
}

// ListSessionEvents retrieves all cash movements in a shift session.
func (r *CashDrawerRepo) ListSessionEvents(ctx context.Context, sessionID string, limit int) ([]*model.CashDrawerEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT e.id, e.session_id, e.shop_id, e.staff_user_id, COALESCE(u.full_name, 'Staff'),
		       e.event_type, e.amount, e.bill_id, e.category, e.reason, e.created_at
		FROM cash_drawer_events e
		LEFT JOIN users u ON u.id = e.staff_user_id
		WHERE e.session_id = $1
		ORDER BY e.created_at DESC
		LIMIT $2;
	`
	rows, err := r.db.Query(ctx, query, sessionID, limit)
	if err != nil {
		r.logger.Error("failed to list cash drawer events", zap.Error(err), zap.String("session_id", sessionID))
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.CashDrawerEvent, 0)
	for rows.Next() {
		ev := &model.CashDrawerEvent{}
		err := rows.Scan(
			&ev.ID, &ev.SessionID, &ev.ShopID, &ev.StaffUserID, &ev.StaffName,
			&ev.EventType, &ev.Amount, &ev.BillID, &ev.Category, &ev.Reason, &ev.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

// ListHistory retrieves past shift drawer sessions.
func (r *CashDrawerRepo) ListHistory(ctx context.Context, shopID string, limit int) ([]*model.CashDrawerSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	query := `
		SELECT s.id, s.shop_id, s.staff_user_id, COALESCE(u.full_name, 'Staff'), s.shift_name,
		       s.opened_at, s.closed_at, s.opening_cash, s.cash_sales_total, s.safe_drop_total,
		       s.expenses_total, s.expected_cash, s.actual_cash, s.discrepancy_amount,
		       s.safe_drop_threshold, s.status, COALESCE(s.closing_notes, ''), s.created_at, s.updated_at
		FROM cash_drawer_sessions s
		JOIN users u ON u.id = s.staff_user_id
		WHERE s.shop_id = $1
		ORDER BY s.opened_at DESC
		LIMIT $2;
	`
	rows, err := r.db.Query(ctx, query, shopID, limit)
	if err != nil {
		r.logger.Error("failed to list cash drawer history", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*model.CashDrawerSession, 0)
	for rows.Next() {
		s := &model.CashDrawerSession{}
		err := rows.Scan(
			&s.ID, &s.ShopID, &s.StaffUserID, &s.StaffName, &s.ShiftName,
			&s.OpenedAt, &s.ClosedAt, &s.OpeningCash, &s.CashSalesTotal, &s.SafeDropTotal,
			&s.ExpensesTotal, &s.ExpectedCash, &s.ActualCash, &s.DiscrepancyAmount,
			&s.SafeDropThreshold, &s.Status, &s.ClosingNotes, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

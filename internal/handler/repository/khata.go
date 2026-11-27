// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrKhataNotFound          = errors.New("khata account not found")
	ErrInsufficientBalance    = errors.New("payment amount exceeds current balance")
	ErrInvalidTransactionType = errors.New("invalid transaction type")
	ErrCreditLimitExceeded    = errors.New("customer credit limit exceeded")
	ErrKhataBalanceNonZero    = errors.New("khata cannot be closed while balance is greater than zero. Please settle remaining balance first")
)

type KhataRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewKhataRepo(db *pgxpool.Pool, logger *zap.Logger) *KhataRepo {
	return &KhataRepo{
		db:     db,
		logger: logger,
	}
}

// GetOrCreateCustomer retrieves or creates a khata record for a customer in a shop.
func (r *KhataRepo) GetOrCreateCustomer(ctx context.Context, shopID, customerName, customerMobile string) (*model.CustomerKhata, error) {
	cleanMobile, err := utils.NormalizeIndianPhone(customerMobile)
	if err != nil {
		cleanMobile = strings.TrimSpace(customerMobile)
	}

	digitsMobile := ""
	for _, ch := range cleanMobile {
		if ch >= '0' && ch <= '9' {
			digitsMobile += string(ch)
		}
	}
	if len(digitsMobile) > 10 {
		digitsMobile = digitsMobile[len(digitsMobile)-10:]
	}

	var khata model.CustomerKhata
	query := `
		SELECT id, shop_id, customer_id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0),
		       COALESCE(closure_status, 'ACTIVE'), closure_otp, closure_requested_by, closure_requested_at, closed_at,
		       created_at, updated_at
		FROM customer_khata
		WHERE shop_id = $1 AND (
			customer_mobile = $2 OR
			customer_phone = $2 OR
			($3 != '' AND RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10) = $3)
		)
	`
	err = r.db.QueryRow(ctx, query, shopID, cleanMobile, digitsMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.ClosureStatus, &khata.ClosureOTP,
		&khata.ClosureRequestedBy, &khata.ClosureRequestedAt, &khata.ClosedAt,
		&khata.CreatedAt, &khata.UpdatedAt,
	)
	if err == nil {
		// If existing and name changed, update it
		if customerName != "" && customerName != khata.CustomerName {
			_, _ = r.db.Exec(ctx, `UPDATE customer_khata SET customer_name = $1, updated_at = NOW() WHERE id = $2`, customerName, khata.ID)
			khata.CustomerName = customerName
		}
		// If customer_id is null, attempt to link with registered users
		if khata.CustomerID == nil && digitsMobile != "" {
			var uID string
			userQuery := `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 OR phone = $1 OR phone = $2 LIMIT 1`
			if uErr := r.db.QueryRow(ctx, userQuery, digitsMobile, cleanMobile).Scan(&uID); uErr == nil {
				_, _ = r.db.Exec(ctx, `UPDATE customer_khata SET customer_id = $1, updated_at = NOW() WHERE id = $2`, uID, khata.ID)
				khata.CustomerID = &uID
			}
		}
		khata.IsRegistered = khata.CustomerID != nil
		return &khata, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to query customer khata: %w", err)
	}

	// Lookup if user is already registered on Shopsilo
	var matchedUserID *string
	if digitsMobile != "" {
		var uID string
		userQuery := `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 OR phone = $1 OR phone = $2 LIMIT 1`
		if uErr := r.db.QueryRow(ctx, userQuery, digitsMobile, cleanMobile).Scan(&uID); uErr == nil {
			matchedUserID = &uID
		}
	}

	// Insert new customer khata
	insertQuery := `
		INSERT INTO customer_khata (shop_id, customer_id, customer_name, customer_mobile, customer_phone, current_balance, credit_limit, closure_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, 0.00, 0.00, 'ACTIVE', NOW(), NOW())
		RETURNING id, shop_id, customer_id, customer_name, customer_mobile, current_balance, credit_limit, closure_status, created_at, updated_at
	`
	err = r.db.QueryRow(ctx, insertQuery, shopID, matchedUserID, customerName, cleanMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.ClosureStatus, &khata.CreatedAt, &khata.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer khata: %w", err)
	}
	khata.IsRegistered = khata.CustomerID != nil
	return &khata, nil
}

// RecordTransaction atomically records a credit or payment transaction and updates the customer balance.
func (r *KhataRepo) RecordTransaction(ctx context.Context, shopID, customerMobile, customerName, txType string, amount float64, notes, billNumber, paymentMode, parchiImageURL, itemsSummary string) (*model.KhataTransaction, error) {
	return r.RecordTransactionWithIdempotency(ctx, shopID, customerMobile, customerName, txType, amount, notes, billNumber, paymentMode, parchiImageURL, itemsSummary, "")
}

// RecordTransactionWithIdempotency atomically records a credit or payment transaction with optional client_tx_id idempotency.
func (r *KhataRepo) RecordTransactionWithIdempotency(ctx context.Context, shopID, customerMobile, customerName, txType string, amount float64, notes, billNumber, paymentMode, parchiImageURL, itemsSummary, clientTxID string) (*model.KhataTransaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	kTx, err := r.RecordTransactionWithTxIdempotency(ctx, tx, shopID, customerMobile, customerName, txType, amount, notes, billNumber, paymentMode, parchiImageURL, itemsSummary, clientTxID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return kTx, nil
}

// RecordTransactionWithTx executes a customer khata credit/payment transaction within an existing database transaction (ACID).
func (r *KhataRepo) RecordTransactionWithTx(ctx context.Context, tx pgx.Tx, shopID, customerMobile, customerName, txType string, amount float64, notes, billNumber, paymentMode, parchiImageURL, itemsSummary string) (*model.KhataTransaction, error) {
	return r.RecordTransactionWithTxIdempotency(ctx, tx, shopID, customerMobile, customerName, txType, amount, notes, billNumber, paymentMode, parchiImageURL, itemsSummary, "")
}

// RecordTransactionWithTxIdempotency executes a customer khata transaction within an existing tx with DB-level client_tx_id idempotency.
func (r *KhataRepo) RecordTransactionWithTxIdempotency(ctx context.Context, tx pgx.Tx, shopID, customerMobile, customerName, txType string, amount float64, notes, billNumber, paymentMode, parchiImageURL, itemsSummary, clientTxID string) (*model.KhataTransaction, error) {
	cleanClientTxID := strings.TrimSpace(clientTxID)
	if cleanClientTxID != "" {
		var existingTx model.KhataTransaction
		var pImg, itmSum string
		idemQuery := `
			SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
			       COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''),
			       COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''), created_at
			FROM khata_transactions
			WHERE shop_id = $1::uuid AND client_tx_id = $2
			LIMIT 1
		`
		if err := tx.QueryRow(ctx, idemQuery, shopID, cleanClientTxID).Scan(
			&existingTx.ID, &existingTx.KhataID, &existingTx.ShopID, &existingTx.Type, &existingTx.Amount, &existingTx.BalanceAfter,
			&existingTx.Notes, &existingTx.BillNumber, &existingTx.PaymentMode, &existingTx.Status, &existingTx.DisputeReason,
			&existingTx.DisputedAt, &existingTx.UPIRefNo, &pImg, &itmSum, &existingTx.CreatedAt,
		); err == nil {
			if pImg != "" {
				existingTx.ParchiImageURL = &pImg
			}
			if itmSum != "" {
				existingTx.ItemsSummary = &itmSum
			}
			existingTx.ClientTxID = &cleanClientTxID
			return &existingTx, nil
		}
	}
	cleanMobile, err := utils.NormalizeIndianPhone(customerMobile)
	if err != nil {
		cleanMobile = strings.TrimSpace(customerMobile)
	}

	// 1. Lock and fetch current customer khata
	digitsMobile := ""
	for _, ch := range cleanMobile {
		if ch >= '0' && ch <= '9' {
			digitsMobile += string(ch)
		}
	}
	if len(digitsMobile) > 10 {
		digitsMobile = digitsMobile[len(digitsMobile)-10:]
	}

	var khataID string
	var currentBalance, creditLimit float64
	var closureStatus string
	lockQuery := `
		SELECT id, current_balance, COALESCE(credit_limit, 0), COALESCE(closure_status, 'ACTIVE')
		FROM customer_khata
		WHERE shop_id = $1 AND (
			customer_mobile = $2 OR
			customer_phone = $2 OR
			($3 != '' AND RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10) = $3)
		)
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, shopID, cleanMobile, digitsMobile).Scan(&khataID, &currentBalance, &creditLimit, &closureStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		// Lookup if user is already registered on Shopsilo
		var matchedUserID *string
		if digitsMobile != "" {
			var uID string
			userQuery := `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 OR phone = $1 OR phone = $2 LIMIT 1`
			if uErr := tx.QueryRow(ctx, userQuery, digitsMobile, cleanMobile).Scan(&uID); uErr == nil {
				matchedUserID = &uID
			}
		}

		// Insert new khata inside this tx
		insertQuery := `
			INSERT INTO customer_khata (shop_id, customer_id, customer_name, customer_mobile, customer_phone, current_balance, credit_limit, closure_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $4, 0.00, 0.00, 'ACTIVE', NOW(), NOW())
			RETURNING id, current_balance, credit_limit
		`
		err = tx.QueryRow(ctx, insertQuery, shopID, matchedUserID, customerName, cleanMobile).Scan(&khataID, &currentBalance, &creditLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to insert khata account: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query khata account: %w", err)
	}

	// 1.1 Idempotency check: detect duplicate taps within 5 seconds
	var dupTx model.KhataTransaction
	dupQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		       COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''), created_at
		FROM khata_transactions
		WHERE khata_id = $1 AND type = $2 AND amount = $3 AND COALESCE(notes, '') = $4 AND created_at >= NOW() - INTERVAL '5 seconds'
		ORDER BY created_at DESC
		LIMIT 1
	`
	if err := tx.QueryRow(ctx, dupQuery, khataID, txType, amount, strings.TrimSpace(notes)).Scan(
		&dupTx.ID, &dupTx.KhataID, &dupTx.ShopID, &dupTx.Type, &dupTx.Amount, &dupTx.BalanceAfter,
		&dupTx.Notes, &dupTx.BillNumber, &dupTx.PaymentMode, &dupTx.Status, &dupTx.DisputeReason,
		&dupTx.DisputedAt, &dupTx.UPIRefNo, &dupTx.CreatedAt,
	); err == nil {
		return &dupTx, nil
	}

	// 2. Compute new balance and verify credit limit
	var newBalance float64
	if txType == model.KhataTxTypeGiveCredit {
		newBalance = currentBalance + amount
		if creditLimit > 0 && newBalance > creditLimit {
			return nil, ErrCreditLimitExceeded
		}
	} else if txType == model.KhataTxTypeReceivePayment {
		newBalance = currentBalance - amount
		if newBalance < 0 {
			newBalance = 0 // prevent negative balance if customer pays slightly extra
		}
	} else {
		return nil, ErrInvalidTransactionType
	}

	// 3. Update customer_khata balance, oldest_unpaid_at, and re-activate if previously closed or pending closure
	updateQuery := `
		UPDATE customer_khata
		SET current_balance = $1,
		    oldest_unpaid_at = CASE
		        WHEN $1 <= 0 THEN NULL
		        ELSE COALESCE(oldest_unpaid_at, NOW())
		    END,
		    closure_status = 'ACTIVE',
		    closure_otp = NULL,
		    closure_requested_by = NULL,
		    closed_at = NULL,
		    updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateQuery, newBalance, khataID); err != nil {
		return nil, fmt.Errorf("failed to update khata balance: %w", err)
	}

	// 4. Insert khata transaction
	var kTx model.KhataTransaction
	insertTxQuery := `
		INSERT INTO khata_transactions (
			khata_id, shop_id, type, amount, balance_after, notes, bill_number, payment_mode, parchi_image_url, items_summary, client_tx_id, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), 'CONFIRMED', NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		          COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''),
		          COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''), created_at
	`
	var pImg, itmSum string
	err = tx.QueryRow(ctx, insertTxQuery, khataID, shopID, txType, amount, newBalance, notes, billNumber, paymentMode, strings.TrimSpace(parchiImageURL), strings.TrimSpace(itemsSummary), cleanClientTxID).Scan(
		&kTx.ID, &kTx.KhataID, &kTx.ShopID, &kTx.Type, &kTx.Amount, &kTx.BalanceAfter, &kTx.Notes, &kTx.BillNumber, &kTx.PaymentMode,
		&kTx.Status, &kTx.DisputeReason, &kTx.DisputedAt, &kTx.UPIRefNo, &pImg, &itmSum, &kTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert khata transaction: %w", err)
	}
	if pImg != "" {
		kTx.ParchiImageURL = &pImg
	}
	if itmSum != "" {
		kTx.ItemsSummary = &itmSum
	}
	if cleanClientTxID != "" {
		kTx.ClientTxID = &cleanClientTxID
	}

	return &kTx, nil
}

// UpdateCreditLimit updates the credit cap for a customer in a shop.
func (r *KhataRepo) UpdateCreditLimit(ctx context.Context, shopID, customerMobile string, limit float64) error {
	cleanMobile, err := utils.NormalizeIndianPhone(customerMobile)
	if err != nil {
		cleanMobile = strings.TrimSpace(customerMobile)
	}
	digitsMobile := ""
	for _, ch := range cleanMobile {
		if ch >= '0' && ch <= '9' {
			digitsMobile += string(ch)
		}
	}
	if len(digitsMobile) > 10 {
		digitsMobile = digitsMobile[len(digitsMobile)-10:]
	}

	query := `
		UPDATE customer_khata
		SET credit_limit = $1, updated_at = NOW()
		WHERE shop_id = $2 AND (
			customer_mobile = $3 OR
			customer_phone = $3 OR
			($4 != '' AND RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10) = $4)
		)
	`
	res, err := r.db.Exec(ctx, query, limit, shopID, cleanMobile, digitsMobile)
	if err != nil {
		return fmt.Errorf("failed to update credit limit: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrKhataNotFound
	}
	return nil
}

// GetAgingReport calculates bad-debt aging buckets and lists overdue credit balances.
func (r *KhataRepo) GetAgingReport(ctx context.Context, shopID string) (*dto.KhataAgingReport, error) {
	report := &dto.KhataAgingReport{
		OverdueList: make([]dto.OverdueCustomerItem, 0),
	}

	summaryQuery := `
		SELECT
			COALESCE(SUM(current_balance), 0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN (NOW() - COALESCE(oldest_unpaid_at, updated_at)) <= INTERVAL '30 days' THEN current_balance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (NOW() - COALESCE(oldest_unpaid_at, updated_at)) > INTERVAL '30 days' AND (NOW() - COALESCE(oldest_unpaid_at, updated_at)) <= INTERVAL '60 days' THEN current_balance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (NOW() - COALESCE(oldest_unpaid_at, updated_at)) > INTERVAL '60 days' THEN current_balance ELSE 0 END), 0)
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance > 0
	`
	err := r.db.QueryRow(ctx, summaryQuery, shopID).Scan(
		&report.TotalOutstanding,
		&report.TotalCustomers,
		&report.Bucket0To30,
		&report.Bucket31To60,
		&report.Bucket60Plus,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query khata aging summary: %w", err)
	}

	listQuery := `
		SELECT 
			id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0),
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(oldest_unpaid_at, updated_at))) / 86400))::int AS days_overdue,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS last_activity_at
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance > 0
		ORDER BY days_overdue DESC, current_balance DESC
		LIMIT 100
	`
	rows, err := r.db.Query(ctx, listQuery, shopID)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue customers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item dto.OverdueCustomerItem
		if err := rows.Scan(
			&item.KhataID, &item.CustomerName, &item.CustomerMobile,
			&item.CurrentBalance, &item.CreditLimit, &item.DaysOverdue, &item.LastActivityAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan overdue item: %w", err)
		}
		report.OverdueList = append(report.OverdueList, item)
	}

	return report, nil
}

// ListCustomers returns all customers with their balances, optionally filtering by search, only outstanding dues, and pagination.
func (r *KhataRepo) ListCustomers(ctx context.Context, shopID string, search string, onlyWithBalance bool, pagination ...int) ([]model.CustomerKhata, error) {
	limit := 0
	offset := 0
	if len(pagination) > 0 && pagination[0] > 0 {
		limit = pagination[0]
		if limit > 200 {
			limit = 200
		}
	}
	if len(pagination) > 1 && pagination[1] > 0 {
		offset = pagination[1]
	}

	query := `
		SELECT ck.id, ck.shop_id, ck.customer_id, ck.customer_name, ck.customer_mobile, ck.current_balance, COALESCE(ck.credit_limit, 0),
		       COALESCE(ck.credit_otp_required, false), COALESCE(ck.credit_otp_threshold, 0),
		       COALESCE(ck.closure_status, 'ACTIVE'), ck.closure_otp, ck.closure_requested_by, ck.closure_requested_at, ck.closed_at,
		       ck.promise_to_pay_date, COALESCE(ck.installment_target, 0),
		       GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(ck.oldest_unpaid_at, ck.updated_at))) / 86400))::int AS days_overdue,
		       (ck.customer_id IS NOT NULL) AS is_registered, ck.created_at, ck.updated_at
		FROM customer_khata ck
		WHERE ck.shop_id = $1
	`
	args := []interface{}{shopID}
	argIdx := 2

	if onlyWithBalance {
		query += " AND ck.current_balance > 0"
	}

	cleanSearch := strings.TrimSpace(search)
	if cleanSearch != "" {
		query += fmt.Sprintf(" AND (LOWER(ck.customer_name) LIKE LOWER($%d) OR ck.customer_mobile LIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+cleanSearch+"%")
		argIdx++
	}

	query += " ORDER BY ck.current_balance DESC, ck.updated_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, limit, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query khata customers: %w", err)
	}
	defer rows.Close()

	var customers []model.CustomerKhata
	for rows.Next() {
		var c model.CustomerKhata
		var daysOverdue int
		if err := rows.Scan(
			&c.ID, &c.ShopID, &c.CustomerID, &c.CustomerName, &c.CustomerMobile,
			&c.CurrentBalance, &c.CreditLimit, &c.CreditOTPRequired, &c.CreditOTPThreshold,
			&c.ClosureStatus, &c.ClosureOTP,
			&c.ClosureRequestedBy, &c.ClosureRequestedAt, &c.ClosedAt,
			&c.PromiseToPayDate, &c.InstallmentTarget, &daysOverdue,
			&c.IsRegistered, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer khata: %w", err)
		}
		c.TrustScore, c.TrustBadge = computeQuickTrustScore(c.CurrentBalance, daysOverdue)
		customers = append(customers, c)
	}
	return customers, nil
}

// GetCustomerHistory retrieves customer info along with their transaction history (with optional limit/offset).
func (r *KhataRepo) GetCustomerHistory(ctx context.Context, shopID, customerMobile string, pagination ...int) (*model.CustomerKhata, []model.KhataTransaction, error) {
	limit := 0
	offset := 0
	if len(pagination) > 0 && pagination[0] > 0 {
		limit = pagination[0]
		if limit > 200 {
			limit = 200
		}
	}
	if len(pagination) > 1 && pagination[1] > 0 {
		offset = pagination[1]
	}
	cleanMobile, err := utils.NormalizeIndianPhone(customerMobile)
	if err != nil {
		cleanMobile = strings.TrimSpace(customerMobile)
	}

	var khata model.CustomerKhata
	var daysOverdue int
	query := `
		SELECT ck.id, ck.shop_id, ck.customer_id, ck.customer_name, ck.customer_mobile, ck.current_balance, COALESCE(ck.credit_limit, 0),
		       COALESCE(ck.credit_otp_required, false), COALESCE(ck.credit_otp_threshold, 0),
		       COALESCE(ck.closure_status, 'ACTIVE'), ck.closure_otp, ck.closure_requested_by, ck.closure_requested_at, ck.closed_at,
		       ck.promise_to_pay_date, COALESCE(ck.installment_target, 0),
		       GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(ck.oldest_unpaid_at, ck.updated_at))) / 86400))::int AS days_overdue,
		       (ck.customer_id IS NOT NULL) AS is_registered, ck.created_at, ck.updated_at
		FROM customer_khata ck
		WHERE ck.shop_id = $1 AND (
			ck.customer_mobile = $2 OR
			RIGHT(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g'), 10) = $2
		)
	`
	err = r.db.QueryRow(ctx, query, shopID, cleanMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.CreditOTPRequired, &khata.CreditOTPThreshold,
		&khata.ClosureStatus, &khata.ClosureOTP,
		&khata.ClosureRequestedBy, &khata.ClosureRequestedAt, &khata.ClosedAt,
		&khata.PromiseToPayDate, &khata.InstallmentTarget, &daysOverdue,
		&khata.IsRegistered, &khata.CreatedAt, &khata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrKhataNotFound
		}
		return nil, nil, fmt.Errorf("failed to query khata: %w", err)
	}

	// Dynamic Trust Score
	khata.TrustScore, khata.TrustBadge = computeQuickTrustScore(khata.CurrentBalance, daysOverdue)

	txQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		       COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, resolution_action, resolution_notes, resolved_at, reversal_of_id, COALESCE(upi_ref_no, ''),
		       COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''),
		       COALESCE(is_deleted, false), deleted_at, COALESCE(deleted_by, ''), COALESCE(delete_reason, ''),
		       original_amount, COALESCE(edit_reason, ''), COALESCE(modified_by, ''), updated_at,
		       created_at
		FROM khata_transactions
		WHERE khata_id = $1
		ORDER BY created_at DESC
	`
	txArgs := []interface{}{khata.ID}
	if limit > 0 {
		txQuery += " LIMIT $2 OFFSET $3"
		txArgs = append(txArgs, limit, offset)
	}
	rows, err := r.db.Query(ctx, txQuery, txArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query khata transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.KhataTransaction
	for rows.Next() {
		var t model.KhataTransaction
		var pImg, itmSum, delBy, delReas, edReas, modBy string
		if err := rows.Scan(&t.ID, &t.KhataID, &t.ShopID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Notes, &t.BillNumber, &t.PaymentMode,
			&t.Status, &t.DisputeReason, &t.DisputedAt, &t.ResolutionAction, &t.ResolutionNotes, &t.ResolvedAt, &t.ReversalOfID, &t.UPIRefNo,
			&pImg, &itmSum,
			&t.IsDeleted, &t.DeletedAt, &delBy, &delReas,
			&t.OriginalAmount, &edReas, &modBy, &t.UpdatedAt,
			&t.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if pImg != "" {
			t.ParchiImageURL = &pImg
		}
		if itmSum != "" {
			t.ItemsSummary = &itmSum
		}
		if delBy != "" {
			t.DeletedBy = &delBy
		}
		if delReas != "" {
			t.DeleteReason = &delReas
		}
		if edReas != "" {
			t.EditReason = &edReas
		}
		if modBy != "" {
			t.ModifiedBy = &modBy
		}
		transactions = append(transactions, t)
	}

	return &khata, transactions, nil
}

// GetSummary returns total market credit outstanding and indebted customers count.
func (r *KhataRepo) GetSummary(ctx context.Context, shopID string) (*model.KhataSummary, error) {
	query := `
		SELECT 
			COALESCE(SUM(current_balance), 0),
			COUNT(CASE WHEN current_balance > 0 THEN 1 END)
		FROM customer_khata
		WHERE shop_id = $1
	`
	var summary model.KhataSummary
	err := r.db.QueryRow(ctx, query, shopID).Scan(&summary.TotalOutstandingAmount, &summary.TotalCustomers)
	if err != nil {
		return nil, fmt.Errorf("failed to query khata summary: %w", err)
	}

	txCountQuery := `SELECT COUNT(*) FROM khata_transactions WHERE shop_id = $1`
	_ = r.db.QueryRow(ctx, txCountQuery, shopID).Scan(&summary.TotalTransactionsCount)

	return &summary, nil
}

// FindCustomerKhatasByUserIDOrPhone retrieves all khata accounts across shops where the customer owes or has credit.
func (r *KhataRepo) FindCustomerKhatasByUserIDOrPhone(ctx context.Context, userID, phone string) ([]dto.CustomerShopKhataItem, error) {
	digitsPhone := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digitsPhone += string(ch)
		}
	}
	if len(digitsPhone) > 10 {
		digitsPhone = digitsPhone[len(digitsPhone)-10:]
	}
	cleanPhone := digitsPhone

	// Proactively link customer_id if not yet linked
	if userID != "" && cleanPhone != "" {
		linkQuery := `
			UPDATE customer_khata
			SET customer_id = $1::uuid, updated_at = NOW()
			WHERE customer_id IS NULL
			  AND (
				  customer_mobile = $2
				  OR customer_phone = $2
				  OR (
					  LENGTH(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g')) >= 10
					  AND RIGHT(REGEXP_REPLACE(COALESCE(customer_mobile, customer_phone, ''), '[^0-9]', '', 'g'), 10) = $2
				  )
			  )
		`
		_, _ = r.db.Exec(ctx, linkQuery, userID, cleanPhone)
	}

	query := `
		SELECT 
			ck.id,
			ck.shop_id,
			s.name,
			s.slug,
			COALESCE(s.phone, ''),
			COALESCE(s.whatsapp_number, ''),
			COALESCE(s.address, ''),
			COALESCE(s.logo_url, ''),
			COALESCE(s.upi_id, ''),
			ck.current_balance,
			COALESCE(ck.credit_limit, 0),
			COALESCE(ck.credit_otp_required, false),
			COALESCE(ck.credit_otp_threshold, 0),
			COALESCE(ck.closure_status, 'ACTIVE'),
			COALESCE(ck.closure_requested_by, ''),
			CASE WHEN ck.closure_requested_by = 'SHOP' THEN COALESCE(ck.closure_otp, '') ELSE '' END,
			COALESCE(to_char(ck.promise_to_pay_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			COALESCE(ck.installment_target, 0),
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - ck.updated_at)) / 86400))::int AS days_overdue,
			COALESCE(to_char(ck.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM customer_khata ck
		JOIN shops s ON s.id = ck.shop_id
		WHERE (
			ck.customer_id = $1::uuid
			OR (
				$2 != '' 
				AND (
					ck.customer_mobile = $2
					OR ck.customer_phone = $2
					OR (
						LENGTH(REGEXP_REPLACE(COALESCE(ck.customer_mobile, ck.customer_phone, ''), '[^0-9]', '', 'g')) >= 10
						AND RIGHT(REGEXP_REPLACE(COALESCE(ck.customer_mobile, ck.customer_phone, ''), '[^0-9]', '', 'g'), 10) = $2
					)
				)
			)
		)
		ORDER BY ck.current_balance DESC, ck.updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, cleanPhone)
	if err != nil {
		return nil, fmt.Errorf("failed to query customer khatas: %w", err)
	}
	defer rows.Close()

	items := make([]dto.CustomerShopKhataItem, 0)
	for rows.Next() {
		var item dto.CustomerShopKhataItem
		var daysOverdue int
		if err := rows.Scan(
			&item.KhataID, &item.ShopID, &item.ShopName, &item.ShopSlug,
			&item.ShopPhone, &item.ShopWhatsApp, &item.ShopAddress, &item.ShopLogo, &item.ShopUPIID,
			&item.CurrentBalance, &item.CreditLimit, &item.CreditOTPRequired, &item.CreditOTPThreshold,
			&item.ClosureStatus, &item.ClosureRequestedBy,
			&item.ClosureOTP, &item.PromiseToPayDate, &item.InstallmentTarget, &daysOverdue, &item.LastActivityAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan customer khata item: %w", err)
		}
		item.TrustScore, item.TrustBadge = computeQuickTrustScore(item.CurrentBalance, daysOverdue)
		items = append(items, item)
	}

	return items, nil
}

// GetCustomerKhataWithShop returns a specific shop khata and full transactions ledger for an authenticated customer.
func (r *KhataRepo) GetCustomerKhataWithShop(ctx context.Context, khataID, userID, phone string) (*dto.CustomerShopKhataItem, string, []model.KhataTransaction, error) {
	digitsPhone := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digitsPhone += string(ch)
		}
	}
	if len(digitsPhone) > 10 {
		digitsPhone = digitsPhone[len(digitsPhone)-10:]
	}
	cleanPhone := digitsPhone

	query := `
		SELECT 
			ck.id,
			ck.shop_id,
			ck.customer_name,
			s.name,
			s.slug,
			COALESCE(s.phone, ''),
			COALESCE(s.whatsapp_number, ''),
			COALESCE(s.address, ''),
			COALESCE(s.logo_url, ''),
			COALESCE(s.upi_id, ''),
			ck.current_balance,
			COALESCE(ck.credit_limit, 0),
			COALESCE(ck.credit_otp_required, false),
			COALESCE(ck.credit_otp_threshold, 0),
			COALESCE(ck.closure_status, 'ACTIVE'),
			COALESCE(ck.closure_requested_by, ''),
			CASE WHEN ck.closure_requested_by = 'SHOP' THEN COALESCE(ck.closure_otp, '') ELSE '' END,
			COALESCE(to_char(ck.promise_to_pay_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			COALESCE(ck.installment_target, 0),
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - ck.updated_at)) / 86400))::int AS days_overdue,
			COALESCE(to_char(ck.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM customer_khata ck
		JOIN shops s ON s.id = ck.shop_id
		WHERE ck.id = $1::uuid
		  AND (
			  ck.customer_id = $2::uuid
			  OR (
				  $3 != '' 
				  AND (
					  ck.customer_mobile = $3
					  OR (
						  LENGTH(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g')) >= 10
						  AND RIGHT(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g'), 10) = $3
					  )
				  )
			  )
		  )
	`
	var item dto.CustomerShopKhataItem
	var customerName string
	var daysOverdue int
	err := r.db.QueryRow(ctx, query, khataID, userID, cleanPhone).Scan(
		&item.KhataID, &item.ShopID, &customerName, &item.ShopName, &item.ShopSlug,
		&item.ShopPhone, &item.ShopWhatsApp, &item.ShopAddress, &item.ShopLogo, &item.ShopUPIID,
		&item.CurrentBalance, &item.CreditLimit, &item.CreditOTPRequired, &item.CreditOTPThreshold,
		&item.ClosureStatus, &item.ClosureRequestedBy,
		&item.ClosureOTP, &item.PromiseToPayDate, &item.InstallmentTarget, &daysOverdue, &item.LastActivityAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil, ErrKhataNotFound
		}
		return nil, "", nil, fmt.Errorf("failed to query customer khata: %w", err)
	}

	item.TrustScore, item.TrustBadge = computeQuickTrustScore(item.CurrentBalance, daysOverdue)

	txQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		       COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, resolution_action, resolution_notes, resolved_at, reversal_of_id, COALESCE(upi_ref_no, ''),
		       COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''),
		       COALESCE(is_deleted, false), deleted_at, COALESCE(deleted_by, ''), COALESCE(delete_reason, ''),
		       original_amount, COALESCE(edit_reason, ''), COALESCE(modified_by, ''), updated_at,
		       created_at
		FROM khata_transactions
		WHERE khata_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, txQuery, khataID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to query khata transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]model.KhataTransaction, 0)
	for rows.Next() {
		var t model.KhataTransaction
		var pImg, itmSum, delBy, delReas, edReas, modBy string
		if err := rows.Scan(&t.ID, &t.KhataID, &t.ShopID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Notes, &t.BillNumber, &t.PaymentMode,
			&t.Status, &t.DisputeReason, &t.DisputedAt, &t.ResolutionAction, &t.ResolutionNotes, &t.ResolvedAt, &t.ReversalOfID, &t.UPIRefNo,
			&pImg, &itmSum,
			&t.IsDeleted, &t.DeletedAt, &delBy, &delReas,
			&t.OriginalAmount, &edReas, &modBy, &t.UpdatedAt,
			&t.CreatedAt); err != nil {
			return nil, "", nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if pImg != "" {
			t.ParchiImageURL = &pImg
		}
		if itmSum != "" {
			t.ItemsSummary = &itmSum
		}
		if delBy != "" {
			t.DeletedBy = &delBy
		}
		if delReas != "" {
			t.DeleteReason = &delReas
		}
		if edReas != "" {
			t.EditReason = &edReas
		}
		if modBy != "" {
			t.ModifiedBy = &modBy
		}
		transactions = append(transactions, t)
	}

	return &item, customerName, transactions, nil
}

// DisputeTransaction flags an incorrect credit entry and records the dispute reason.
func (r *KhataRepo) DisputeTransaction(ctx context.Context, khataID, txID, reason string) error {
	query := `
		UPDATE khata_transactions
		SET status = 'DISPUTED',
		    dispute_reason = $1,
		    disputed_at = NOW()
		WHERE id = $2::uuid AND khata_id = $3::uuid AND type = 'GIVE_CREDIT'
	`
	res, err := r.db.Exec(ctx, query, strings.TrimSpace(reason), txID, khataID)
	if err != nil {
		return fmt.Errorf("failed to dispute transaction: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errors.New("transaction not found or not eligible for dispute")
	}
	return nil
}

// DeleteCustomer removes a settled customer khata account if balance is 0.
func (r *KhataRepo) DeleteCustomer(ctx context.Context, shopID, khataID string, force bool) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx for delete customer: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance float64
	err = tx.QueryRow(ctx, `SELECT current_balance FROM customer_khata WHERE id = $1 AND shop_id = $2 FOR UPDATE`, khataID, shopID).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrKhataNotFound
		}
		return err
	}

	if balance != 0 && !force {
		return ErrKhataBalanceNonZero
	}

	// Delete transactions
	_, _ = tx.Exec(ctx, `DELETE FROM khata_transactions WHERE khata_id = $1`, khataID)

	// Delete customer account
	cmdTag, err := tx.Exec(ctx, `DELETE FROM customer_khata WHERE id = $1 AND shop_id = $2`, khataID, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete customer khata: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrKhataNotFound
	}

	return tx.Commit(ctx)
}

// UpdateTransactionNotes updates the notes string on an existing ledger entry.
func (r *KhataRepo) UpdateTransactionNotes(ctx context.Context, shopID, transactionID, notes string) error {
	query := `
		UPDATE khata_transactions
		SET notes = $1
		WHERE id = $2 AND shop_id = $3
	`
	cmdTag, err := r.db.Exec(ctx, query, notes, transactionID, shopID)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("transaction not found")
	}
	return nil
}

// SubmitUPIPayment records a customer's direct UPI settlement proof with UTR reference as PENDING_APPROVAL.
// Balance is only deducted once the shopkeeper verifies and approves the UPI receipt.

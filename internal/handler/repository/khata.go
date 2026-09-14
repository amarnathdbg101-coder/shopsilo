// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"strings"
	"time"

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

	var khata model.CustomerKhata
	query := `
		SELECT id, shop_id, customer_id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0),
		       COALESCE(closure_status, 'ACTIVE'), closure_otp, closure_requested_by, closure_requested_at, closed_at,
		       created_at, updated_at
		FROM customer_khata
		WHERE shop_id = $1 AND (
			customer_mobile = $2 OR
			RIGHT(REGEXP_REPLACE(customer_mobile, '[^0-9]', '', 'g'), 10) = $2
		)
	`
	err = r.db.QueryRow(ctx, query, shopID, cleanMobile).Scan(
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
		if khata.CustomerID == nil {
			var uID string
			if uErr := r.db.QueryRow(ctx, `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 LIMIT 1`, cleanMobile).Scan(&uID); uErr == nil {
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
	var uID string
	if uErr := r.db.QueryRow(ctx, `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 LIMIT 1`, cleanMobile).Scan(&uID); uErr == nil {
		matchedUserID = &uID
	}

	// Insert new customer khata
	insertQuery := `
		INSERT INTO customer_khata (shop_id, customer_id, customer_name, customer_mobile, current_balance, credit_limit, closure_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0.00, 0.00, 'ACTIVE', NOW(), NOW())
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
	cleanMobile, err := utils.NormalizeIndianPhone(customerMobile)
	if err != nil {
		cleanMobile = strings.TrimSpace(customerMobile)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and fetch current customer khata
	var khataID string
	var currentBalance, creditLimit float64
	var closureStatus string
	lockQuery := `
		SELECT id, current_balance, COALESCE(credit_limit, 0), COALESCE(closure_status, 'ACTIVE')
		FROM customer_khata
		WHERE shop_id = $1 AND (
			customer_mobile = $2 OR
			RIGHT(REGEXP_REPLACE(customer_mobile, '[^0-9]', '', 'g'), 10) = $2
		)
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, shopID, cleanMobile).Scan(&khataID, &currentBalance, &creditLimit, &closureStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		// Lookup if user is already registered on Shopsilo
		var matchedUserID *string
		var uID string
		if uErr := tx.QueryRow(ctx, `SELECT id FROM users WHERE RIGHT(REGEXP_REPLACE(phone, '[^0-9]', '', 'g'), 10) = $1 LIMIT 1`, cleanMobile).Scan(&uID); uErr == nil {
			matchedUserID = &uID
		}

		// Insert new khata inside this tx
		insertQuery := `
			INSERT INTO customer_khata (shop_id, customer_id, customer_name, customer_mobile, current_balance, credit_limit, closure_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 0.00, 0.00, 'ACTIVE', NOW(), NOW())
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
		// Found identical transaction within 5 seconds, commit and return existing
		_ = tx.Commit(ctx)
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

	// 3. Update customer_khata balance and re-activate if previously closed or pending closure
	updateQuery := `
		UPDATE customer_khata
		SET current_balance = $1,
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
			khata_id, shop_id, type, amount, balance_after, notes, bill_number, payment_mode, parchi_image_url, items_summary, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), NULLIF($10, ''), 'CONFIRMED', NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		          COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''),
		          COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''), created_at
	`
	var pImg, itmSum string
	err = tx.QueryRow(ctx, insertTxQuery, khataID, shopID, txType, amount, newBalance, notes, billNumber, paymentMode, strings.TrimSpace(parchiImageURL), strings.TrimSpace(itemsSummary)).Scan(
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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &kTx, nil
}

// UpdateCreditLimit updates the credit cap for a customer in a shop.
func (r *KhataRepo) UpdateCreditLimit(ctx context.Context, shopID, customerMobile string, limit float64) error {
	query := `
		UPDATE customer_khata
		SET credit_limit = $1, updated_at = NOW()
		WHERE shop_id = $2 AND customer_mobile = $3
	`
	res, err := r.db.Exec(ctx, query, limit, shopID, strings.TrimSpace(customerMobile))
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
			COALESCE(SUM(CASE WHEN (NOW() - updated_at) <= INTERVAL '30 days' THEN current_balance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (NOW() - updated_at) > INTERVAL '30 days' AND (NOW() - updated_at) <= INTERVAL '60 days' THEN current_balance ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN (NOW() - updated_at) > INTERVAL '60 days' THEN current_balance ELSE 0 END), 0)
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
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - updated_at)) / 86400))::int AS days_overdue,
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

// ListCustomers returns all customers with their balances, optionally filtering by search or only outstanding dues.
func (r *KhataRepo) ListCustomers(ctx context.Context, shopID string, search string, onlyWithBalance bool) ([]model.CustomerKhata, error) {
	query := `
		SELECT ck.id, ck.shop_id, ck.customer_id, ck.customer_name, ck.customer_mobile, ck.current_balance, COALESCE(ck.credit_limit, 0),
		       COALESCE(ck.credit_otp_required, false), COALESCE(ck.credit_otp_threshold, 0),
		       COALESCE(ck.closure_status, 'ACTIVE'), ck.closure_otp, ck.closure_requested_by, ck.closure_requested_at, ck.closed_at,
		       ck.promise_to_pay_date, COALESCE(ck.installment_target, 0),
		       GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - ck.updated_at)) / 86400))::int AS days_overdue,
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

// GetCustomerHistory retrieves customer info along with their transaction history.
func (r *KhataRepo) GetCustomerHistory(ctx context.Context, shopID, customerMobile string) (*model.CustomerKhata, []model.KhataTransaction, error) {
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
		       GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - ck.updated_at)) / 86400))::int AS days_overdue,
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
		       COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''), created_at
		FROM khata_transactions
		WHERE khata_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, txQuery, khata.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query khata transactions: %w", err)
	}
	defer rows.Close()

	var transactions []model.KhataTransaction
	for rows.Next() {
		var t model.KhataTransaction
		var pImg, itmSum string
		if err := rows.Scan(&t.ID, &t.KhataID, &t.ShopID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Notes, &t.BillNumber, &t.PaymentMode,
			&t.Status, &t.DisputeReason, &t.DisputedAt, &t.ResolutionAction, &t.ResolutionNotes, &t.ResolvedAt, &t.ReversalOfID, &t.UPIRefNo,
			&pImg, &itmSum, &t.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if pImg != "" {
			t.ParchiImageURL = &pImg
		}
		if itmSum != "" {
			t.ItemsSummary = &itmSum
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
	cleanPhone, _ := utils.NormalizeIndianPhone(phone)
	if cleanPhone == "" {
		cleanPhone = strings.TrimSpace(phone)
	}

	// Proactively link customer_id if not yet linked
	if userID != "" && cleanPhone != "" {
		linkQuery := `
			UPDATE customer_khata
			SET customer_id = $1::uuid, updated_at = NOW()
			WHERE customer_id IS NULL
			  AND (
				  customer_mobile = $2
				  OR (
					  LENGTH(REGEXP_REPLACE(customer_mobile, '[^0-9]', '', 'g')) >= 10
					  AND RIGHT(REGEXP_REPLACE(customer_mobile, '[^0-9]', '', 'g'), 10) = $2
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
					OR (
						LENGTH(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g')) >= 10
						AND RIGHT(REGEXP_REPLACE(ck.customer_mobile, '[^0-9]', '', 'g'), 10) = $2
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
	cleanPhone, _ := utils.NormalizeIndianPhone(phone)
	if cleanPhone == "" {
		cleanPhone = strings.TrimSpace(phone)
	}

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
		       COALESCE(parchi_image_url, ''), COALESCE(items_summary, ''), created_at
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
		var pImg, itmSum string
		if err := rows.Scan(&t.ID, &t.KhataID, &t.ShopID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Notes, &t.BillNumber, &t.PaymentMode,
			&t.Status, &t.DisputeReason, &t.DisputedAt, &t.ResolutionAction, &t.ResolutionNotes, &t.ResolvedAt, &t.ReversalOfID, &t.UPIRefNo,
			&pImg, &itmSum, &t.CreatedAt); err != nil {
			return nil, "", nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if pImg != "" {
			t.ParchiImageURL = &pImg
		}
		if itmSum != "" {
			t.ItemsSummary = &itmSum
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

// SubmitUPIPayment records a customer's direct UPI settlement with UTR reference.
func (r *KhataRepo) SubmitUPIPayment(ctx context.Context, khataID string, amount float64, upiRef, notes string) (*model.KhataTransaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentBalance float64
	var shopID string
	err = tx.QueryRow(ctx, `SELECT current_balance, shop_id FROM customer_khata WHERE id = $1 FOR UPDATE`, khataID).Scan(&currentBalance, &shopID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrKhataNotFound
		}
		return nil, fmt.Errorf("failed to query khata: %w", err)
	}

	newBalance := currentBalance - amount
	if newBalance < 0 {
		newBalance = 0
	}

	_, err = tx.Exec(ctx, `UPDATE customer_khata SET current_balance = $1, updated_at = NOW() WHERE id = $2`, newBalance, khataID)
	if err != nil {
		return nil, fmt.Errorf("failed to update khata balance: %w", err)
	}

	var kTx model.KhataTransaction
	insertQuery := `
		INSERT INTO khata_transactions (
			khata_id, shop_id, type, amount, balance_after, notes, payment_mode, status, upi_ref_no, created_at
		) VALUES ($1, $2, 'RECEIVE_PAYMENT', $3, $4, $5, 'upi', 'CONFIRMED', $6, NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		          COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''), created_at
	`
	err = tx.QueryRow(ctx, insertQuery, khataID, shopID, amount, newBalance, notes, strings.TrimSpace(upiRef)).Scan(
		&kTx.ID, &kTx.KhataID, &kTx.ShopID, &kTx.Type, &kTx.Amount, &kTx.BalanceAfter, &kTx.Notes, &kTx.BillNumber, &kTx.PaymentMode,
		&kTx.Status, &kTx.DisputeReason, &kTx.DisputedAt, &kTx.UPIRefNo, &kTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record UPI payment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &kTx, nil
}

// RequestKhataClosure initiates a dual-OTP mutual handshake closure request.
// Only accounts with current_balance == 0 can request closure.
func (r *KhataRepo) RequestKhataClosure(ctx context.Context, khataID, requestedBy string) (string, error) {
	var currentBalance float64
	var closureStatus string
	query := `SELECT current_balance, COALESCE(closure_status, 'ACTIVE') FROM customer_khata WHERE id = $1::uuid`
	err := r.db.QueryRow(ctx, query, khataID).Scan(&currentBalance, &closureStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrKhataNotFound
		}
		return "", fmt.Errorf("failed to query khata for closure: %w", err)
	}

	if currentBalance > 0 {
		return "", ErrKhataBalanceNonZero
	}

	// Generate secure 6-digit OTP
	otp := fmt.Sprintf("%06d", rand.Intn(900000)+100000)

	updateQuery := `
		UPDATE customer_khata
		SET closure_status = 'PENDING_OTP',
		    closure_otp = $1,
		    closure_requested_by = $2,
		    closure_requested_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3::uuid
	`
	_, err = r.db.Exec(ctx, updateQuery, otp, requestedBy, khataID)
	if err != nil {
		return "", fmt.Errorf("failed to initiate khata closure: %w", err)
	}

	return otp, nil
}

// VerifyKhataClosureOTP validates the shared 6-digit OTP and marks the khata CLOSED.
func (r *KhataRepo) VerifyKhataClosureOTP(ctx context.Context, khataID, otp, verifyingParty string) error {
	var currentBalance float64
	var closureStatus string
	var savedOTP, requestedBy *string
	query := `
		SELECT current_balance, COALESCE(closure_status, 'ACTIVE'), closure_otp, closure_requested_by
		FROM customer_khata
		WHERE id = $1::uuid
	`
	err := r.db.QueryRow(ctx, query, khataID).Scan(&currentBalance, &closureStatus, &savedOTP, &requestedBy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrKhataNotFound
		}
		return fmt.Errorf("failed to query khata: %w", err)
	}

	if currentBalance > 0 {
		return ErrKhataBalanceNonZero
	}

	if closureStatus != model.KhataClosureStatusPendingOTP || savedOTP == nil {
		return errors.New("is khate par koi pending closure request nahi hai")
	}

	cleanInputOTP := strings.TrimSpace(otp)
	if cleanInputOTP != *savedOTP {
		return errors.New("galat OTP! Kripya dusre paksh se mila 6-digit closure OTP enter karein")
	}

	updateQuery := `
		UPDATE customer_khata
		SET closure_status = 'CLOSED',
		    closure_otp = NULL,
		    closed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1::uuid
	`
	_, err = r.db.Exec(ctx, updateQuery, khataID)
	if err != nil {
		return fmt.Errorf("failed to close khata: %w", err)
	}

	return nil
}

// AutoLinkCustomerKhatas links all existing offline khata accounts to a newly registered/updated user.
func (r *KhataRepo) AutoLinkCustomerKhatas(ctx context.Context, userID, rawPhone string) (int64, error) {
	cleanPhone, err := utils.NormalizeIndianPhone(rawPhone)
	if err != nil {
		cleanPhone = strings.TrimSpace(rawPhone)
	}
	if len(cleanPhone) < 10 {
		return 0, nil
	}

	query := `
		UPDATE customer_khata
		SET customer_id = $1::uuid, updated_at = NOW()
		WHERE customer_id IS NULL
		  AND (
			  customer_mobile = $2 OR
			  RIGHT(REGEXP_REPLACE(customer_mobile, '[^0-9]', '', 'g'), 10) = $2
		  )
	`
	res, err := r.db.Exec(ctx, query, userID, cleanPhone)
	if err != nil {
		return 0, fmt.Errorf("failed to auto-link customer khatas: %w", err)
	}
	return res.RowsAffected(), nil
}

// ReverseTransaction records an official audit reversal for an erroneous credit or payment entry.
func (r *KhataRepo) ReverseTransaction(ctx context.Context, shopID, khataID, txID, reason string) (*model.KhataTransaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch target transaction
	var origTx model.KhataTransaction
	fetchQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		       COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at,
		       COALESCE(upi_ref_no, ''), created_at
		FROM khata_transactions
		WHERE id = $1::uuid AND khata_id = $2::uuid AND shop_id = $3::uuid
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, fetchQuery, txID, khataID, shopID).Scan(
		&origTx.ID, &origTx.KhataID, &origTx.ShopID, &origTx.Type, &origTx.Amount, &origTx.BalanceAfter,
		&origTx.Notes, &origTx.BillNumber, &origTx.PaymentMode, &origTx.Status, &origTx.DisputeReason,
		&origTx.DisputedAt, &origTx.UPIRefNo, &origTx.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to query original transaction: %w", err)
	}

	if origTx.Type == model.KhataTxTypeReversal {
		return nil, errors.New("cannot reverse a reversal transaction")
	}

	// Fetch khata balance
	var currentBalance float64
	err = tx.QueryRow(ctx, `SELECT current_balance FROM customer_khata WHERE id = $1::uuid FOR UPDATE`, khataID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to lock customer khata: %w", err)
	}

	// Compute reversed balance
	var newBalance float64
	if origTx.Type == model.KhataTxTypeGiveCredit {
		// Udhar was given: reversal subtracts amount
		newBalance = currentBalance - origTx.Amount
		if newBalance < 0 {
			newBalance = 0
		}
	} else if origTx.Type == model.KhataTxTypeReceivePayment {
		// Payment was recorded: reversal adds amount back
		newBalance = currentBalance + origTx.Amount
	} else {
		return nil, errors.New("unsupported transaction type for reversal")
	}

	// Update customer_khata balance
	updateQuery := `UPDATE customer_khata SET current_balance = $1, updated_at = NOW() WHERE id = $2::uuid`
	if _, err := tx.Exec(ctx, updateQuery, newBalance, khataID); err != nil {
		return nil, fmt.Errorf("failed to update customer balance: %w", err)
	}

	// Mark original transaction as resolved
	_, _ = tx.Exec(ctx, `UPDATE khata_transactions SET status = 'RESOLVED_ACCEPTED', resolution_action = 'ACCEPT', resolution_notes = $1, resolved_at = NOW() WHERE id = $2::uuid`, reason, txID)

	// Insert reversal transaction
	var revTx model.KhataTransaction
	insertQuery := `
		INSERT INTO khata_transactions (
			khata_id, shop_id, type, amount, balance_after, notes, reversal_of_id, status, created_at
		) VALUES ($1, $2, 'REVERSAL', $3, $4, $5, $6, 'CONFIRMED', NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		          COALESCE(status, 'CONFIRMED'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''), created_at
	`
	revNotes := fmt.Sprintf("Galti Sudhar (Reversal): %s", strings.TrimSpace(reason))
	err = tx.QueryRow(ctx, insertQuery, khataID, shopID, origTx.Amount, newBalance, revNotes, txID).Scan(
		&revTx.ID, &revTx.KhataID, &revTx.ShopID, &revTx.Type, &revTx.Amount, &revTx.BalanceAfter,
		&revTx.Notes, &revTx.BillNumber, &revTx.PaymentMode, &revTx.Status, &revTx.DisputeReason,
		&revTx.DisputedAt, &revTx.UPIRefNo, &revTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert reversal entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit reversal: %w", err)
	}

	return &revTx, nil
}

// ResolveDispute resolves an open dispute on a credit transaction.
func (r *KhataRepo) ResolveDispute(ctx context.Context, shopID, khataID, txID, action, resolutionNotes string) error {
	cleanAction := strings.ToUpper(strings.TrimSpace(action))
	if cleanAction == "ACCEPT" {
		_, err := r.ReverseTransaction(ctx, shopID, khataID, txID, fmt.Sprintf("Dispute accepted by merchant: %s", resolutionNotes))
		return err
	} else if cleanAction == "REJECT" {
		query := `
			UPDATE khata_transactions
			SET status = 'RESOLVED_REJECTED',
			    resolution_action = 'REJECT',
			    resolution_notes = $1,
			    resolved_at = NOW()
			WHERE id = $2::uuid AND khata_id = $3::uuid AND shop_id = $4::uuid
		`
		res, err := r.db.Exec(ctx, query, strings.TrimSpace(resolutionNotes), txID, khataID, shopID)
		if err != nil {
			return fmt.Errorf("failed to reject dispute: %w", err)
		}
		if res.RowsAffected() == 0 {
			return errors.New("transaction not found")
		}
		return nil
	}
	return errors.New("invalid dispute resolution action: must be ACCEPT or REJECT")
}

// SetCreditOTPProtection configures customer OTP requirement for high-value credit.
func (r *KhataRepo) SetCreditOTPProtection(ctx context.Context, khataID string, required bool, threshold float64) error {
	query := `
		UPDATE customer_khata
		SET credit_otp_required = $1,
		    credit_otp_threshold = $2,
		    updated_at = NOW()
		WHERE id = $3::uuid
	`
	res, err := r.db.Exec(ctx, query, required, threshold, khataID)
	if err != nil {
		return fmt.Errorf("failed to update credit OTP protection: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrKhataNotFound
	}
	return nil
}

// computeQuickTrustScore evaluates a quick 300-900 score from current balance and days overdue.
func computeQuickTrustScore(currentBal float64, daysOverdue int) (int, string) {
	if currentBal == 0 {
		return 820, "TRUSTED"
	}
	if daysOverdue <= 15 {
		return 780, "TRUSTED"
	}
	if daysOverdue <= 30 {
		return 670, "MODERATE"
	}
	if daysOverdue <= 60 {
		return 540, "HIGH_RISK"
	}
	return 380, "HIGH_RISK"
}

// CalculateTrustScore computes an in-depth 300-900 score for a customer khata based on their repayment history, disputes, and aging.
func (r *KhataRepo) CalculateTrustScore(ctx context.Context, khataID string) (score int, badge string, summary string, onTimeRate float64, avgDays int, totalTx int, err error) {
	score = 650
	badge = "MODERATE"
	summary = "Normal payment terms laagu hain."

	query := `
		SELECT 
			COUNT(*) FILTER (WHERE type = 'GIVE_CREDIT') AS total_credits,
			COUNT(*) FILTER (WHERE type = 'RECEIVE_PAYMENT') AS total_payments,
			COUNT(*) FILTER (WHERE status = 'DISPUTED') AS active_disputes,
			COUNT(*) FILTER (WHERE status = 'RESOLVED_ACCEPTED') AS accepted_disputes,
			COALESCE(SUM(amount) FILTER (WHERE type = 'GIVE_CREDIT'), 0) AS total_borrowed,
			COALESCE(SUM(amount) FILTER (WHERE type = 'RECEIVE_PAYMENT'), 0) AS total_repaid
		FROM khata_transactions
		WHERE khata_id = $1::uuid
	`
	var totalCredits, totalPayments, activeDisputes, acceptedDisputes int
	var totalBorrowed, totalRepaid float64
	if qErr := r.db.QueryRow(ctx, query, khataID).Scan(
		&totalCredits, &totalPayments, &activeDisputes, &acceptedDisputes,
		&totalBorrowed, &totalRepaid,
	); qErr != nil {
		return 650, "MODERATE", "Default evaluation score", 100, 0, 0, nil
	}

	totalTx = totalCredits + totalPayments
	if totalTx == 0 {
		return 650, "MODERATE", "Abhi koi purani transaction nahi hai. Udhar shuru kar sakte hain.", 100, 0, 0, nil
	}

	if totalBorrowed > 0 {
		onTimeRate = (totalRepaid / totalBorrowed) * 100
		if onTimeRate > 100 {
			onTimeRate = 100
		}
	} else {
		onTimeRate = 100
	}

	var daysSinceLastCredit int
	var currentBal float64
	var pDate *time.Time
	_ = r.db.QueryRow(ctx, `
		SELECT 
			current_balance,
			promise_to_pay_date,
			GREATEST(0, FLOOR(EXTRACT(EPOCH FROM (NOW() - updated_at)) / 86400))::int
		FROM customer_khata WHERE id = $1::uuid
	`, khataID).Scan(&currentBal, &pDate, &daysSinceLastCredit)

	avgDays = daysSinceLastCredit

	if onTimeRate >= 90 {
		score += 80
	} else if onTimeRate >= 70 {
		score += 30
	} else if onTimeRate < 50 {
		score -= 70
	}

	if currentBal == 0 {
		score += 50
	} else {
		if daysSinceLastCredit <= 15 {
			score += 40
		} else if daysSinceLastCredit <= 30 {
			score += 10
		} else if daysSinceLastCredit <= 60 {
			score -= 60
		} else {
			score -= 150
		}
	}

	score -= (activeDisputes * 50)
	score -= (acceptedDisputes * 20)

	if score > 900 {
		score = 900
	}
	if score < 300 {
		score = 300
	}

	if score >= 750 {
		badge = "TRUSTED"
		summary = "Vishwasniya Grahak! Track record behtareen hai. Udhar dena safe hai."
	} else if score >= 600 {
		badge = "MODERATE"
		summary = "Madhyam Vishwas. Payment thoda samay leti hai, par chukta ho jati hai."
	} else {
		badge = "HIGH_RISK"
		summary = "Dhyan dein! Purani udhari lambe samay se baki hai ya dispute history hai."
	}

	return score, badge, summary, onTimeRate, avgDays, totalTx, nil
}

// SetPromiseToPay updates the agreed promise-to-pay date and installment target amount.
func (r *KhataRepo) SetPromiseToPay(ctx context.Context, khataID string, promiseDate *time.Time, target float64) error {
	query := `
		UPDATE customer_khata
		SET promise_to_pay_date = $1,
		    installment_target = $2,
		    updated_at = NOW()
		WHERE id = $3::uuid
	`
	res, err := r.db.Exec(ctx, query, promiseDate, target, khataID)
	if err != nil {
		return fmt.Errorf("failed to update promise to pay: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrKhataNotFound
	}
	return nil
}

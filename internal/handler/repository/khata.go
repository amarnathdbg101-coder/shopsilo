// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
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
	var khata model.CustomerKhata
	query := `
		SELECT id, shop_id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0), created_at, updated_at
		FROM customer_khata
		WHERE shop_id = $1 AND customer_mobile = $2
	`
	err := r.db.QueryRow(ctx, query, shopID, customerMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.CreatedAt, &khata.UpdatedAt,
	)
	if err == nil {
		// If existing, optionally update name if it changed
		if customerName != "" && customerName != khata.CustomerName {
			_, _ = r.db.Exec(ctx, `UPDATE customer_khata SET customer_name = $1, updated_at = NOW() WHERE id = $2`, customerName, khata.ID)
			khata.CustomerName = customerName
		}
		return &khata, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to query customer khata: %w", err)
	}

	// Insert new customer khata
	insertQuery := `
		INSERT INTO customer_khata (shop_id, customer_name, customer_mobile, current_balance, credit_limit, created_at, updated_at)
		VALUES ($1, $2, $3, 0.00, 0.00, NOW(), NOW())
		RETURNING id, shop_id, customer_name, customer_mobile, current_balance, credit_limit, created_at, updated_at
	`
	err = r.db.QueryRow(ctx, insertQuery, shopID, customerName, customerMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.CreatedAt, &khata.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer khata: %w", err)
	}
	return &khata, nil
}

// RecordTransaction atomically records a credit or payment transaction and updates the customer balance.
func (r *KhataRepo) RecordTransaction(ctx context.Context, shopID, customerMobile, customerName, txType string, amount float64, notes, billNumber, paymentMode string) (*model.KhataTransaction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and fetch current customer khata
	var khataID string
	var currentBalance, creditLimit float64
	lockQuery := `
		SELECT id, current_balance, COALESCE(credit_limit, 0)
		FROM customer_khata
		WHERE shop_id = $1 AND customer_mobile = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, shopID, customerMobile).Scan(&khataID, &currentBalance, &creditLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		// If doesn't exist, create it inside this tx
		insertQuery := `
			INSERT INTO customer_khata (shop_id, customer_name, customer_mobile, current_balance, credit_limit, created_at, updated_at)
			VALUES ($1, $2, $3, 0.00, 0.00, NOW(), NOW())
			RETURNING id, current_balance, credit_limit
		`
		err = tx.QueryRow(ctx, insertQuery, shopID, customerName, customerMobile).Scan(&khataID, &currentBalance, &creditLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to insert khata account: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to query khata account: %w", err)
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

	// 3. Update customer_khata balance
	updateQuery := `
		UPDATE customer_khata
		SET current_balance = $1, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(ctx, updateQuery, newBalance, khataID); err != nil {
		return nil, fmt.Errorf("failed to update khata balance: %w", err)
	}

	// 4. Insert khata transaction
	var kTx model.KhataTransaction
	insertTxQuery := `
		INSERT INTO khata_transactions (
			khata_id, shop_id, type, amount, balance_after, notes, bill_number, payment_mode, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''), created_at
	`
	err = tx.QueryRow(ctx, insertTxQuery, khataID, shopID, txType, amount, newBalance, notes, billNumber, paymentMode).Scan(
		&kTx.ID, &kTx.KhataID, &kTx.ShopID, &kTx.Type, &kTx.Amount, &kTx.BalanceAfter, &kTx.Notes, &kTx.BillNumber, &kTx.PaymentMode, &kTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert khata transaction: %w", err)
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
		SELECT id, shop_id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0), created_at, updated_at
		FROM customer_khata
		WHERE shop_id = $1
	`
	args := []interface{}{shopID}
	argIdx := 2

	if onlyWithBalance {
		query += " AND current_balance > 0"
	}

	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(customer_name) LIKE LOWER($%d) OR customer_mobile LIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	query += " ORDER BY current_balance DESC, updated_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query khata customers: %w", err)
	}
	defer rows.Close()

	var customers []model.CustomerKhata
	for rows.Next() {
		var c model.CustomerKhata
		if err := rows.Scan(&c.ID, &c.ShopID, &c.CustomerName, &c.CustomerMobile, &c.CurrentBalance, &c.CreditLimit, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan customer khata: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, nil
}

// GetCustomerHistory retrieves customer info along with their transaction history.
func (r *KhataRepo) GetCustomerHistory(ctx context.Context, shopID, customerMobile string) (*model.CustomerKhata, []model.KhataTransaction, error) {
	var khata model.CustomerKhata
	query := `
		SELECT id, shop_id, customer_name, customer_mobile, current_balance, COALESCE(credit_limit, 0), created_at, updated_at
		FROM customer_khata
		WHERE shop_id = $1 AND customer_mobile = $2
	`
	err := r.db.QueryRow(ctx, query, shopID, customerMobile).Scan(
		&khata.ID, &khata.ShopID, &khata.CustomerName, &khata.CustomerMobile,
		&khata.CurrentBalance, &khata.CreditLimit, &khata.CreatedAt, &khata.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrKhataNotFound
		}
		return nil, nil, fmt.Errorf("failed to query khata: %w", err)
	}

	txQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''), created_at
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
		if err := rows.Scan(&t.ID, &t.KhataID, &t.ShopID, &t.Type, &t.Amount, &t.BalanceAfter, &t.Notes, &t.BillNumber, &t.PaymentMode, &t.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("failed to scan transaction: %w", err)
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

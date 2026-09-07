// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrExpenseNotFound = errors.New("expense record not found")
)

type ExpenseRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewExpenseRepo(db *pgxpool.Pool, logger *zap.Logger) *ExpenseRepo {
	return &ExpenseRepo{
		db:     db,
		logger: logger,
	}
}

// CreateExpense records an operational store expense.
func (r *ExpenseRepo) CreateExpense(ctx context.Context, shopID string, expense *model.ShopExpense) (*model.ShopExpense, error) {
	query := `
		INSERT INTO shop_expenses (shop_id, category, amount, notes, payment_method, expense_date, created_at)
		VALUES ($1, $2, $3, $4, $5, 
			CASE WHEN $6 != '' THEN $6::DATE ELSE CURRENT_DATE END, 
			NOW())
		RETURNING id, shop_id, category, amount, COALESCE(notes, ''), payment_method, expense_date::TEXT, created_at
	`
	var res model.ShopExpense
	err := r.db.QueryRow(
		ctx, query,
		shopID, expense.Category, expense.Amount, expense.Notes, expense.PaymentMethod, expense.ExpenseDate,
	).Scan(
		&res.ID, &res.ShopID, &res.Category, &res.Amount, &res.Notes, &res.PaymentMethod, &res.ExpenseDate, &res.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert expense: %w", err)
	}
	return &res, nil
}

// ListExpenses returns all expenses for a given month and year, optionally filtered by category.
func (r *ExpenseRepo) ListExpenses(ctx context.Context, shopID string, year, month int, category string) ([]model.ShopExpense, float64, map[string]float64, error) {
	query := `
		SELECT id, shop_id, category, amount, COALESCE(notes, ''), payment_method, expense_date::TEXT, created_at
		FROM shop_expenses
		WHERE shop_id = $1 
		  AND EXTRACT(YEAR FROM expense_date) = $2
		  AND EXTRACT(MONTH FROM expense_date) = $3
	`
	args := []interface{}{shopID, year, month}
	if category != "" {
		query += " AND category = $4"
		args = append(args, category)
	}
	query += " ORDER BY expense_date DESC, created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to query expenses: %w", err)
	}
	defer rows.Close()

	var expenses []model.ShopExpense
	var total float64
	catTotals := make(map[string]float64)

	for rows.Next() {
		var e model.ShopExpense
		if err := rows.Scan(&e.ID, &e.ShopID, &e.Category, &e.Amount, &e.Notes, &e.PaymentMethod, &e.ExpenseDate, &e.CreatedAt); err != nil {
			return nil, 0, nil, fmt.Errorf("failed to scan expense: %w", err)
		}
		expenses = append(expenses, e)
		total += e.Amount
		catTotals[e.Category] += e.Amount
	}

	return expenses, total, catTotals, nil
}

// DeleteExpense deletes an expense belonging to the shop.
func (r *ExpenseRepo) DeleteExpense(ctx context.Context, shopID, expenseID string) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM shop_expenses WHERE id = $1 AND shop_id = $2`, expenseID, shopID)
	if err != nil {
		return fmt.Errorf("failed to delete expense: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrExpenseNotFound
	}
	return nil
}

// GetTotalExpensesForMonth calculates sum of expenses for a shop in a specific year and month.
func (r *ExpenseRepo) GetTotalExpensesForMonth(ctx context.Context, shopID string, year, month int) (float64, map[string]float64, error) {
	query := `
		SELECT category, COALESCE(SUM(amount), 0)
		FROM shop_expenses
		WHERE shop_id = $1
		  AND EXTRACT(YEAR FROM expense_date) = $2
		  AND EXTRACT(MONTH FROM expense_date) = $3
		GROUP BY category
	`
	rows, err := r.db.Query(ctx, query, shopID, year, month)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to query monthly expenses: %w", err)
	}
	defer rows.Close()

	var total float64
	breakdown := make(map[string]float64)
	for rows.Next() {
		var cat string
		var amt float64
		if err := rows.Scan(&cat, &amt); err != nil {
			return 0, nil, fmt.Errorf("failed to scan monthly expense row: %w", err)
		}
		breakdown[cat] = amt
		total += amt
	}
	return total, breakdown, nil
}

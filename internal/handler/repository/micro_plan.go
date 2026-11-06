// Package repository handles database queries for customer debt recovery and micro-installment plans.
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
	ErrMicroPlanNotFound = errors.New("khata micro-installment plan not found")
	ErrMicroPlanInactive = errors.New("khata micro-installment plan is not active")
)

// MicroPlanRepo manages database operations for micro-repayments.
type MicroPlanRepo struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewMicroPlanRepo initializes MicroPlanRepo.
func NewMicroPlanRepo(db *pgxpool.Pool, logger *zap.Logger) *MicroPlanRepo {
	return &MicroPlanRepo{
		db:     db,
		logger: logger,
	}
}

// CreatePlan initializes a new micro-installment schedule.
func (r *MicroPlanRepo) CreatePlan(ctx context.Context, p *model.KhataMicroPlan) (*model.KhataMicroPlan, error) {
	if p.Frequency == "" {
		p.Frequency = "DAILY"
	}
	if p.TotalInstallments <= 0 && p.DailyAmount > 0 {
		p.TotalInstallments = int(math.Ceil(p.TotalDebt / p.DailyAmount))
	}
	p.RemainingBalance = p.TotalDebt
	p.Status = model.PlanStatusActive
	if p.NextDueDate.IsZero() {
		p.NextDueDate = time.Now().Add(24 * time.Hour)
	}

	query := `
		INSERT INTO khata_micro_plans (
			shop_id, customer_id, customer_phone, customer_name, total_debt,
			daily_amount, frequency, total_installments, paid_installments,
			total_paid_amount, remaining_balance, next_due_date, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 0.00, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at;
	`
	err := r.db.QueryRow(ctx, query,
		p.ShopID, p.CustomerID, p.CustomerPhone, p.CustomerName, p.TotalDebt,
		p.DailyAmount, p.Frequency, p.TotalInstallments, p.RemainingBalance,
		p.NextDueDate, p.Status,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to create khata micro plan", zap.Error(err), zap.String("shop_id", p.ShopID))
		return nil, err
	}
	return p, nil
}

// GetPlanByID retrieves a single plan by its UUID.
func (r *MicroPlanRepo) GetPlanByID(ctx context.Context, planID string) (*model.KhataMicroPlan, error) {
	query := `
		SELECT id, shop_id, customer_id, customer_phone, customer_name, total_debt,
		       daily_amount, frequency, total_installments, paid_installments,
		       total_paid_amount, remaining_balance, next_due_date, status, created_at, updated_at
		FROM khata_micro_plans
		WHERE id = $1;
	`
	p := &model.KhataMicroPlan{}
	err := r.db.QueryRow(ctx, query, planID).Scan(
		&p.ID, &p.ShopID, &p.CustomerID, &p.CustomerPhone, &p.CustomerName, &p.TotalDebt,
		&p.DailyAmount, &p.Frequency, &p.TotalInstallments, &p.PaidInstallments,
		&p.TotalPaidAmount, &p.RemainingBalance, &p.NextDueDate, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMicroPlanNotFound
		}
		return nil, err
	}
	return p, nil
}

// RecordPayment processes an installment payment atomically.
func (r *MicroPlanRepo) RecordPayment(ctx context.Context, payment *model.KhataMicroPayment) (*model.KhataMicroPlan, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	p := &model.KhataMicroPlan{}
	query := `
		SELECT id, shop_id, customer_id, customer_phone, customer_name, total_debt,
		       daily_amount, frequency, total_installments, paid_installments,
		       total_paid_amount, remaining_balance, next_due_date, status
		FROM khata_micro_plans
		WHERE id = $1
		FOR UPDATE;
	`
	err = tx.QueryRow(ctx, query, payment.PlanID).Scan(
		&p.ID, &p.ShopID, &p.CustomerID, &p.CustomerPhone, &p.CustomerName, &p.TotalDebt,
		&p.DailyAmount, &p.Frequency, &p.TotalInstallments, &p.PaidInstallments,
		&p.TotalPaidAmount, &p.RemainingBalance, &p.NextDueDate, &p.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMicroPlanNotFound
		}
		return nil, err
	}

	if p.Status != model.PlanStatusActive {
		return nil, ErrMicroPlanInactive
	}

	// Update calculations
	newPaidInstallments := p.PaidInstallments + 1
	newPaidTotal := p.TotalPaidAmount + payment.Amount
	newRemaining := p.RemainingBalance - payment.Amount

	newStatus := model.PlanStatusActive
	if newRemaining <= 0.01 {
		newRemaining = 0.00
		newStatus = model.PlanStatusCompleted
	}

	// Advance next due date
	nextDue := p.NextDueDate
	switch p.Frequency {
	case "WEEKLY":
		nextDue = nextDue.Add(7 * 24 * time.Hour)
	case "MONTHLY":
		nextDue = nextDue.Add(30 * 24 * time.Hour)
	default: // DAILY
		nextDue = nextDue.Add(24 * time.Hour)
	}

	// 1. Insert Payment Log
	payQuery := `
		INSERT INTO khata_micro_payments (
			plan_id, shop_id, installment_number, amount, payment_mode,
			transaction_reference, notes, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING id, created_at;
	`
	payment.InstallmentNumber = newPaidInstallments
	payment.ShopID = p.ShopID
	err = tx.QueryRow(ctx, payQuery,
		payment.PlanID, payment.ShopID, payment.InstallmentNumber, payment.Amount,
		payment.PaymentMode, payment.TransactionReference, payment.Notes,
	).Scan(&payment.ID, &payment.CreatedAt)
	if err != nil {
		return nil, err
	}

	// 2. Update Plan State
	planUpdateQuery := `
		UPDATE khata_micro_plans
		SET paid_installments = $1,
		    total_paid_amount = $2,
		    remaining_balance = $3,
		    next_due_date = $4,
		    status = $5,
		    updated_at = NOW()
		WHERE id = $6;
	`
	_, err = tx.Exec(ctx, planUpdateQuery, newPaidInstallments, newPaidTotal, newRemaining, nextDue, newStatus, p.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	p.PaidInstallments = newPaidInstallments
	p.TotalPaidAmount = newPaidTotal
	p.RemainingBalance = newRemaining
	p.NextDueDate = nextDue
	p.Status = newStatus

	return p, nil
}

// ListPlans queries all micro-repayment plans for a shop.
func (r *MicroPlanRepo) ListPlans(ctx context.Context, shopID, status string, limit int) ([]*model.KhataMicroPlan, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, shop_id, customer_id, customer_phone, customer_name, total_debt,
		       daily_amount, frequency, total_installments, paid_installments,
		       total_paid_amount, remaining_balance, next_due_date, status, created_at, updated_at
		FROM khata_micro_plans
		WHERE shop_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3;
	`
	rows, err := r.db.Query(ctx, query, shopID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plans := make([]*model.KhataMicroPlan, 0)
	for rows.Next() {
		p := &model.KhataMicroPlan{}
		err := rows.Scan(
			&p.ID, &p.ShopID, &p.CustomerID, &p.CustomerPhone, &p.CustomerName, &p.TotalDebt,
			&p.DailyAmount, &p.Frequency, &p.TotalInstallments, &p.PaidInstallments,
			&p.TotalPaidAmount, &p.RemainingBalance, &p.NextDueDate, &p.Status,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// ListPayments returns all historical payments made under a plan.
func (r *MicroPlanRepo) ListPayments(ctx context.Context, planID string) ([]*model.KhataMicroPayment, error) {
	query := `
		SELECT id, plan_id, shop_id, installment_number, amount, payment_mode,
		       transaction_reference, notes, created_at
		FROM khata_micro_payments
		WHERE plan_id = $1
		ORDER BY installment_number ASC;
	`
	rows, err := r.db.Query(ctx, query, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]*model.KhataMicroPayment, 0)
	for rows.Next() {
		pay := &model.KhataMicroPayment{}
		err := rows.Scan(
			&pay.ID, &pay.PlanID, &pay.ShopID, &pay.InstallmentNumber, &pay.Amount,
			&pay.PaymentMode, &pay.TransactionReference, &pay.Notes, &pay.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, pay)
	}
	return payments, nil
}


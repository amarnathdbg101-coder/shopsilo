// Package repository handles database queries.
package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *KhataRepo) SubmitUPIPayment(ctx context.Context, khataID string, amount float64, upiRef, notes string) (*model.KhataTransaction, error) {
	cleanRef := strings.TrimSpace(upiRef)
	if cleanRef == "" {
		return nil, errors.New("UPI reference number is required")
	}

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

	// Prevent duplicate UTR submissions on the same khata
	var dupID string
	dupCheckQuery := `
		SELECT id FROM khata_transactions
		WHERE khata_id = $1 AND LOWER(TRIM(upi_ref_no)) = LOWER($2) AND COALESCE(status, 'CONFIRMED') <> 'REJECTED'
		LIMIT 1
	`
	if err := tx.QueryRow(ctx, dupCheckQuery, khataID, cleanRef).Scan(&dupID); err == nil {
		return nil, errors.New("ye UPI reference number pehle se submit ho chuka hai")
	}

	var kTx model.KhataTransaction
	insertQuery := `
		INSERT INTO khata_transactions (
			khata_id, shop_id, type, amount, balance_after, notes, payment_mode, status, upi_ref_no, created_at
		) VALUES ($1, $2, 'RECEIVE_PAYMENT', $3, $4, $5, 'upi', 'PENDING_APPROVAL', $6, NOW())
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''), COALESCE(payment_mode, ''),
		          COALESCE(status, 'PENDING_APPROVAL'), COALESCE(dispute_reason, ''), disputed_at, COALESCE(upi_ref_no, ''), created_at
	`
	err = tx.QueryRow(ctx, insertQuery, khataID, shopID, amount, currentBalance, notes, cleanRef).Scan(
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

// VerifyUPIPayment allows the merchant to APPROVE or REJECT a customer's pending UPI payment proof.
func (r *KhataRepo) VerifyUPIPayment(ctx context.Context, shopID, khataID, txID, action, notes string) (*model.KhataTransaction, error) {
	cleanAction := strings.ToUpper(strings.TrimSpace(action))
	if cleanAction != "APPROVE" && cleanAction != "REJECT" {
		return nil, errors.New("invalid action: must be APPROVE or REJECT")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var origTx model.KhataTransaction
	fetchQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		       COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), COALESCE(upi_ref_no, ''), created_at
		FROM khata_transactions
		WHERE id = $1::uuid AND khata_id = $2::uuid AND shop_id = $3::uuid
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, fetchQuery, txID, khataID, shopID).Scan(
		&origTx.ID, &origTx.KhataID, &origTx.ShopID, &origTx.Type, &origTx.Amount, &origTx.BalanceAfter,
		&origTx.Notes, &origTx.BillNumber, &origTx.PaymentMode, &origTx.Status, &origTx.UPIRefNo, &origTx.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}

	if origTx.Status != model.KhataTxStatusPendingApproval {
		return nil, fmt.Errorf("transaction is not pending approval (current status: %s)", origTx.Status)
	}

	newStatus := model.KhataTxStatusRejected
	newBalanceAfter := origTx.BalanceAfter

	if cleanAction == "APPROVE" {
		var currentBalance float64
		err = tx.QueryRow(ctx, `SELECT current_balance FROM customer_khata WHERE id = $1::uuid FOR UPDATE`, khataID).Scan(&currentBalance)
		if err != nil {
			return nil, fmt.Errorf("failed to lock customer khata: %w", err)
		}

		newBalance := currentBalance - origTx.Amount
		if newBalance < 0 {
			newBalance = 0
		}
		newBalanceAfter = newBalance
		newStatus = model.KhataTxStatusConfirmed

		_, err = tx.Exec(ctx, `
			UPDATE customer_khata
			SET current_balance = $1,
			    oldest_unpaid_at = CASE WHEN $1 <= 0 THEN NULL ELSE oldest_unpaid_at END,
			    updated_at = NOW()
			WHERE id = $2::uuid
		`, newBalance, khataID)
		if err != nil {
			return nil, fmt.Errorf("failed to update customer khata balance: %w", err)
		}
	}

	var updatedTx model.KhataTransaction
	updateTxQuery := `
		UPDATE khata_transactions
		SET status = $1,
		    balance_after = $2,
		    resolution_action = $3,
		    resolution_notes = NULLIF($4, ''),
		    resolved_at = NOW(),
		    updated_at = NOW()
		WHERE id = $5::uuid
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		          COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), resolution_action, resolution_notes, resolved_at,
		          COALESCE(upi_ref_no, ''), created_at
	`
	err = tx.QueryRow(ctx, updateTxQuery, newStatus, newBalanceAfter, cleanAction, strings.TrimSpace(notes), txID).Scan(
		&updatedTx.ID, &updatedTx.KhataID, &updatedTx.ShopID, &updatedTx.Type, &updatedTx.Amount, &updatedTx.BalanceAfter,
		&updatedTx.Notes, &updatedTx.BillNumber, &updatedTx.PaymentMode, &updatedTx.Status, &updatedTx.ResolutionAction,
		&updatedTx.ResolutionNotes, &updatedTx.ResolvedAt, &updatedTx.UPIRefNo, &updatedTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update UPI verification status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit UPI verification: %w", err)
	}

	return &updatedTx, nil
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


// UpdateTransaction safely edits a credit or payment entry with atomic balance recalculation & audit logging.
// recalculateLedgerBalances replays all active confirmed transactions for a khata in chronological order
// and updates balance_after on any stale rows as well as customer_khata.current_balance and oldest_unpaid_at.
func (r *KhataRepo) recalculateLedgerBalances(ctx context.Context, tx pgx.Tx, khataID string) (float64, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, type, amount, balance_after, reversal_of_id
		FROM khata_transactions
		WHERE khata_id = $1::uuid
		  AND COALESCE(is_deleted, false) = false
		  AND COALESCE(status, 'CONFIRMED') NOT IN ('PENDING_APPROVAL', 'REJECTED', 'CANCELLED')
		ORDER BY created_at ASC, id ASC
		FOR UPDATE
	`, khataID)
	if err != nil {
		return 0, fmt.Errorf("failed to query ledger rows for recalculation: %w", err)
	}

	type ledgerRow struct {
		id           string
		txType       string
		amount       float64
		balanceAfter float64
		reversalOfID *string
	}
	var entries []ledgerRow
	for rows.Next() {
		var row ledgerRow
		if err := rows.Scan(&row.id, &row.txType, &row.amount, &row.balanceAfter, &row.reversalOfID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan ledger row: %w", err)
		}
		entries = append(entries, row)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	txTypes := make(map[string]string, len(entries))
	running := 0.0
	type balanceUpdate struct {
		id         string
		newBalance float64
	}
	var updates []balanceUpdate

	for _, e := range entries {
		txTypes[e.id] = e.txType
		switch e.txType {
		case model.KhataTxTypeGiveCredit:
			running += e.amount
		case model.KhataTxTypeReceivePayment:
			running -= e.amount
			if running < 0 {
				running = 0
			}
		case model.KhataTxTypeReversal:
			if e.reversalOfID != nil {
				origType := txTypes[*e.reversalOfID]
				if origType == model.KhataTxTypeGiveCredit {
					running -= e.amount
					if running < 0 {
						running = 0
					}
				} else if origType == model.KhataTxTypeReceivePayment {
					running += e.amount
				}
			}
		}
		if e.balanceAfter != running {
			updates = append(updates, balanceUpdate{id: e.id, newBalance: running})
		}
	}

	for _, u := range updates {
		if _, err := tx.Exec(ctx, `UPDATE khata_transactions SET balance_after = $1 WHERE id = $2::uuid`, u.newBalance, u.id); err != nil {
			return 0, fmt.Errorf("failed to update recalculated balance_after: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE customer_khata
		SET current_balance = $1,
		    oldest_unpaid_at = CASE WHEN $1 <= 0 THEN NULL ELSE COALESCE(oldest_unpaid_at, NOW()) END,
		    updated_at = NOW()
		WHERE id = $2::uuid
	`, running, khataID)
	if err != nil {
		return 0, fmt.Errorf("failed to update customer khata balance: %w", err)
	}

	return running, nil
}

// UpdateTransaction edits a credit or payment entry atomically with mandatory audit tracking & recalculates running balances.
func (r *KhataRepo) UpdateTransaction(ctx context.Context, shopID, khataID, txID string, newAmount float64, newNotes, newBillNumber, editReason, modifiedBy string) (*model.KhataTransaction, error) {
	if newAmount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if strings.TrimSpace(editReason) == "" {
		return nil, errors.New("edit reason is required for financial audit tracking")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and fetch target transaction
	var origTx model.KhataTransaction
	var isDel bool
	fetchQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		       COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), COALESCE(is_deleted, false)
		FROM khata_transactions
		WHERE id = $1::uuid AND khata_id = $2::uuid AND shop_id = $3::uuid
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, fetchQuery, txID, khataID, shopID).Scan(
		&origTx.ID, &origTx.KhataID, &origTx.ShopID, &origTx.Type, &origTx.Amount, &origTx.BalanceAfter,
		&origTx.Notes, &origTx.BillNumber, &origTx.PaymentMode, &origTx.Status, &isDel,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}

	if isDel {
		return nil, errors.New("cannot edit a cancelled or deleted transaction")
	}
	if origTx.Type == model.KhataTxTypeReversal {
		return nil, errors.New("cannot edit an automated reversal entry")
	}

	// 2. Lock customer_khata row
	var currentBalance float64
	err = tx.QueryRow(ctx, `SELECT current_balance FROM customer_khata WHERE id = $1::uuid FOR UPDATE`, khataID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to lock customer khata: %w", err)
	}

	if origTx.Type != model.KhataTxTypeGiveCredit && origTx.Type != model.KhataTxTypeReceivePayment {
		return nil, errors.New("unsupported transaction type for edit")
	}

	// 3. Update the transaction with audit trail first
	updateTxQuery := `
		UPDATE khata_transactions
		SET original_amount = COALESCE(original_amount, amount),
		    amount = $1,
		    notes = $2,
		    bill_number = $3,
		    edit_reason = $4,
		    modified_by = $5,
		    updated_at = NOW()
		WHERE id = $6::uuid
	`
	if _, err = tx.Exec(ctx, updateTxQuery,
		newAmount,
		strings.TrimSpace(newNotes),
		strings.TrimSpace(newBillNumber),
		strings.TrimSpace(editReason),
		strings.TrimSpace(modifiedBy),
		txID,
	); err != nil {
		return nil, fmt.Errorf("failed to update transaction entry: %w", err)
	}

	// 4. Recalculate running balance_after across this and all subsequent ledger rows + customer_khata.current_balance
	if _, err := r.recalculateLedgerBalances(ctx, tx, khataID); err != nil {
		return nil, err
	}

	// 5. Fetch updated transaction with its exact recalculated balance_after
	selectUpdatedQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		       COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), COALESCE(is_deleted, false),
		       deleted_at, COALESCE(deleted_by, ''), COALESCE(delete_reason, ''),
		       original_amount, COALESCE(edit_reason, ''), COALESCE(modified_by, ''), updated_at, created_at
		FROM khata_transactions
		WHERE id = $1::uuid
	`
	var updatedTx model.KhataTransaction
	var delBy, delReas, modByStr, edReas string
	err = tx.QueryRow(ctx, selectUpdatedQuery, txID).Scan(
		&updatedTx.ID, &updatedTx.KhataID, &updatedTx.ShopID, &updatedTx.Type, &updatedTx.Amount, &updatedTx.BalanceAfter,
		&updatedTx.Notes, &updatedTx.BillNumber, &updatedTx.PaymentMode, &updatedTx.Status, &updatedTx.IsDeleted,
		&updatedTx.DeletedAt, &delBy, &delReas,
		&updatedTx.OriginalAmount, &edReas, &modByStr, &updatedTx.UpdatedAt, &updatedTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction entry: %w", err)
	}
	if delBy != "" { updatedTx.DeletedBy = &delBy }
	if delReas != "" { updatedTx.DeleteReason = &delReas }
	if modByStr != "" { updatedTx.ModifiedBy = &modByStr }
	if edReas != "" { updatedTx.EditReason = &edReas }

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction update: %w", err)
	}

	return &updatedTx, nil
}

// DeleteTransaction soft-deletes a credit or payment entry with atomic balance reversal & audit logging.
func (r *KhataRepo) DeleteTransaction(ctx context.Context, shopID, khataID, txID string, deleteReason, deletedBy string) (*model.KhataTransaction, error) {
	if strings.TrimSpace(deleteReason) == "" {
		return nil, errors.New("delete reason is required for financial audit tracking")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock and fetch target transaction
	var origTx model.KhataTransaction
	var isDel bool
	fetchQuery := `
		SELECT id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		       COALESCE(payment_mode, ''), COALESCE(status, 'CONFIRMED'), COALESCE(is_deleted, false)
		FROM khata_transactions
		WHERE id = $1::uuid AND khata_id = $2::uuid AND shop_id = $3::uuid
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, fetchQuery, txID, khataID, shopID).Scan(
		&origTx.ID, &origTx.KhataID, &origTx.ShopID, &origTx.Type, &origTx.Amount, &origTx.BalanceAfter,
		&origTx.Notes, &origTx.BillNumber, &origTx.PaymentMode, &origTx.Status, &isDel,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}

	if isDel {
		return nil, errors.New("transaction is already deleted or cancelled")
	}
	if origTx.Type == model.KhataTxTypeReversal {
		return nil, errors.New("cannot delete an automated reversal entry")
	}

	// 2. Lock customer_khata before recalculating
	var currentBalance float64
	err = tx.QueryRow(ctx, `SELECT current_balance FROM customer_khata WHERE id = $1::uuid FOR UPDATE`, khataID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("failed to lock customer khata: %w", err)
	}

	if origTx.Type != model.KhataTxTypeGiveCredit && origTx.Type != model.KhataTxTypeReceivePayment {
		return nil, errors.New("unsupported transaction type for deletion")
	}

	// 3. Soft-delete the transaction with status CANCELLED and audit information
	updateTxQuery := `
		UPDATE khata_transactions
		SET is_deleted = true,
		    status = 'CANCELLED',
		    deleted_at = NOW(),
		    deleted_by = $1,
		    delete_reason = $2,
		    updated_at = NOW()
		WHERE id = $3::uuid
		RETURNING id, khata_id, shop_id, type, amount, balance_after, COALESCE(notes, ''), COALESCE(bill_number, ''),
		          COALESCE(payment_mode, ''), COALESCE(status, 'CANCELLED'), COALESCE(is_deleted, true),
		          deleted_at, COALESCE(deleted_by, ''), COALESCE(delete_reason, ''),
		          original_amount, COALESCE(edit_reason, ''), COALESCE(modified_by, ''), updated_at, created_at
	`
	var deletedTx model.KhataTransaction
	var delBy, delReas, modByStr, edReas string
	err = tx.QueryRow(ctx, updateTxQuery,
		strings.TrimSpace(deletedBy),
		strings.TrimSpace(deleteReason),
		txID,
	).Scan(
		&deletedTx.ID, &deletedTx.KhataID, &deletedTx.ShopID, &deletedTx.Type, &deletedTx.Amount, &deletedTx.BalanceAfter,
		&deletedTx.Notes, &deletedTx.BillNumber, &deletedTx.PaymentMode, &deletedTx.Status, &deletedTx.IsDeleted,
		&deletedTx.DeletedAt, &delBy, &delReas,
		&deletedTx.OriginalAmount, &edReas, &modByStr, &deletedTx.UpdatedAt, &deletedTx.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to soft-delete transaction: %w", err)
	}
	if delBy != "" { deletedTx.DeletedBy = &delBy }
	if delReas != "" { deletedTx.DeleteReason = &delReas }
	if modByStr != "" { deletedTx.ModifiedBy = &modByStr }
	if edReas != "" { deletedTx.EditReason = &edReas }

	// 4. Recalculate running balance_after across all remaining active entries + customer_khata.current_balance
	if _, err := r.recalculateLedgerBalances(ctx, tx, khataID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction deletion: %w", err)
	}

	return &deletedTx, nil
}

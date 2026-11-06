package services_test

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/migration"
	"shopMe/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func setupCashDrawerTest(t *testing.T) (*pgxpool.Pool, *repository.ShopRepo, *repository.CashDrawerRepo, *services.CashDrawerService) {
	for _, envPath := range []string{"../../../.env", "../../.env", "../.env", ".env"} {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					_ = os.Setenv("DATABASE_URL", strings.TrimPrefix(line, "DATABASE_URL="))
				}
			}
			break
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	pool, err := utils.ConnectDB(dbURL)
	if err != nil {
		t.Skipf("cannot connect to test db (%v); skipping integration test", err)
		return nil, nil, nil, nil
	}

	migration.EnsureSchemaColumns(pool)

	logger, _ := zap.NewDevelopment()
	shopRepo := repository.NewShopRepo(pool, logger)
	drawerRepo := repository.NewCashDrawerRepo(pool, logger)
	drawerService := services.NewCashDrawerService(shopRepo, drawerRepo, logger)

	return pool, shopRepo, drawerRepo, drawerService
}

func TestCashDrawerTelemetryLifecycle(t *testing.T) {
	pool, _, drawerRepo, drawerService := setupCashDrawerTest(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 1. Fetch an active shop and owner user
	var shopID, ownerUserID string
	err := pool.QueryRow(ctx, "SELECT id, user_id FROM shops WHERE is_active = true LIMIT 1;").Scan(&shopID, &ownerUserID)
	if err != nil {
		t.Skip("No active shop found for test")
	}

	// 2. Clean up any leftover open session for this shop from prior aborted test runs
	_, _ = pool.Exec(ctx, "UPDATE cash_drawer_sessions SET status = 'CLOSED', closed_at = NOW() WHERE shop_id = $1 AND status = 'OPEN';", shopID)

	// Step A: Open Cash Drawer Session with ₹1,000 Opening Float
	var sessionID string
	t.Run("Open Shift Session", func(t *testing.T) {
		safeDropLimit := 3000.00
		openReq := dto.OpenDrawerSessionRequest{
			OpeningCash:       1000.00,
			ShiftName:         "Morning Peak Shift",
			SafeDropThreshold: &safeDropLimit,
		}
		sess, err := drawerService.OpenSession(ctx, ownerUserID, openReq)
		if err != nil {
			t.Fatalf("OpenSession failed: %v", err)
		}
		if sess.OpeningCash != 1000.00 {
			t.Errorf("Expected opening cash 1000.00, got %.2f", sess.OpeningCash)
		}
		if sess.ExpectedCash != 1000.00 {
			t.Errorf("Expected expected_cash 1000.00, got %.2f", sess.ExpectedCash)
		}
		if sess.Status != model.DrawerStatusOpen {
			t.Errorf("Expected status OPEN, got %s", sess.Status)
		}
		sessionID = sess.ID
		t.Logf("Shift opened successfully! ID=%s, OpeningCash=₹%.2f", sess.ID, sess.OpeningCash)
	})

	// Step B: Simulate POS Cash Sales of ₹2,500
	t.Run("Add POS Cash Sale", func(t *testing.T) {
		err := drawerRepo.AddCashSale(ctx, shopID, "dummy-bill-101", ownerUserID, 2500.00)
		if err != nil {
			t.Fatalf("AddCashSale failed: %v", err)
		}

		status, err := drawerService.GetCurrentStatus(ctx, ownerUserID)
		if err != nil {
			t.Fatalf("GetCurrentStatus failed: %v", err)
		}
		expected := 1000.00 + 2500.00 // ₹3,500
		if status.CurrentDrawerCash != expected {
			t.Errorf("Expected current drawer cash ₹%.2f, got ₹%.2f", expected, status.CurrentDrawerCash)
		}
		t.Logf("Cash sale recorded! Current Drawer Cash=₹%.2f", status.CurrentDrawerCash)
	})

	// Step C: Record Petty Cash Expense (₹50 Tea & Refreshments)
	t.Run("Record Petty Cash Expense", func(t *testing.T) {
		expReq := dto.RecordPettyExpenseRequest{
			Amount:   50.00,
			Category: "Chai & Refreshments",
			Reason:   "Evening staff tea and biscuits",
		}
		sess, err := drawerService.RecordPettyExpense(ctx, ownerUserID, expReq)
		if err != nil {
			t.Fatalf("RecordPettyExpense failed: %v", err)
		}
		expected := 3500.00 - 50.00 // ₹3,450
		if sess.ExpectedCash != expected {
			t.Errorf("Expected expected_cash ₹%.2f, got ₹%.2f", expected, sess.ExpectedCash)
		}
		if sess.ExpensesTotal != 50.00 {
			t.Errorf("Expected expenses_total 50.00, got %.2f", sess.ExpensesTotal)
		}
		t.Logf("Petty expense logged! Expected Cash=₹%.2f, Total Expenses=₹%.2f", sess.ExpectedCash, sess.ExpensesTotal)
	})

	// Step D: Verify Safe Drop Recommendation Alert
	t.Run("Safe Drop Recommendation Alert", func(t *testing.T) {
		status, err := drawerService.GetCurrentStatus(ctx, ownerUserID)
		if err != nil {
			t.Fatalf("GetCurrentStatus failed: %v", err)
		}
		// Current cash = ₹3,450 > Threshold ₹3,000 -> Should recommend safe drop!
		if !status.SafeDropRecommended {
			t.Errorf("Expected SafeDropRecommended=true when cash ₹%.2f exceeds limit ₹3000", status.CurrentDrawerCash)
		}
		if status.SafeDropRecommendedAmount <= 0 {
			t.Errorf("Expected positive recommended drop amount, got %.2f", status.SafeDropRecommendedAmount)
		}
		t.Logf("Safe drop alert verified! Current=₹%.2f, Recommended Drop=₹%.2f", status.CurrentDrawerCash, status.SafeDropRecommendedAmount)
	})

	// Step E: Execute Safe Drop (Deposit ₹2,000 into secure tijori vault)
	t.Run("Execute Safe Drop Deposit", func(t *testing.T) {
		dropReq := dto.RecordSafeDropRequest{
			Amount: 2000.00,
			Notes:  "Surplus cash dropped into main counter safe tijori",
		}
		sess, err := drawerService.RecordSafeDrop(ctx, ownerUserID, dropReq)
		if err != nil {
			t.Fatalf("RecordSafeDrop failed: %v", err)
		}
		expected := 3450.00 - 2000.00 // ₹1,450
		if sess.ExpectedCash != expected {
			t.Errorf("Expected expected_cash ₹%.2f after safe drop, got ₹%.2f", expected, sess.ExpectedCash)
		}
		if sess.SafeDropTotal != 2000.00 {
			t.Errorf("Expected safe_drop_total 2000.00, got %.2f", sess.SafeDropTotal)
		}
		t.Logf("Safe drop executed! Remaining Drawer Float=₹%.2f", sess.ExpectedCash)
	})

	// Step F: Close Shift with Physical Cash Count Reconciled
	t.Run("Close Shift and Reconcile", func(t *testing.T) {
		// Cashier counts actual notes in drawer: exactly ₹1,450 (Zero Discrepancy!)
		closeReq := dto.CloseDrawerSessionRequest{
			ActualCash:   1450.00,
			ClosingNotes: "All physical currency notes and coins verified. ₹0 discrepancy.",
		}
		closedSess, err := drawerService.CloseSession(ctx, ownerUserID, closeReq)
		if err != nil {
			t.Fatalf("CloseSession failed: %v", err)
		}
		if closedSess.Status != model.DrawerStatusClosed {
			t.Errorf("Expected status CLOSED, got %s", closedSess.Status)
		}
		if math.Abs(closedSess.DiscrepancyAmount) > 0.01 {
			t.Errorf("Expected discrepancy 0.00, got %.2f", closedSess.DiscrepancyAmount)
		}
		t.Logf("Shift closed and verified! Status=%s, Discrepancy=₹%.2f", closedSess.Status, closedSess.DiscrepancyAmount)

		// Verify Shift History endpoint
		history, err := drawerService.ListHistory(ctx, ownerUserID)
		if err != nil {
			t.Fatalf("ListHistory failed: %v", err)
		}
		if history.Count == 0 {
			t.Errorf("Expected at least 1 session in history")
		}
		t.Logf("Shift history verified! Count=%d", history.Count)
	})
	_ = sessionID
}


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

type milestone4TestHarness struct {
	pool             *pgxpool.Pool
	shopRepo         *repository.ShopRepo
	productRepo      *repository.ProductRepo
	batchRepo        *repository.BatchRepo
	microPlanRepo    *repository.MicroPlanRepo
	batchService     *services.BatchService
	microPlanService *services.MicroPlanService
}

func setupMilestone4Test(t *testing.T) *milestone4TestHarness {
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
		return nil
	}

	migration.EnsureSchemaColumns(pool)

	logger, _ := zap.NewDevelopment()
	shopRepo := repository.NewShopRepo(pool, logger)
	productRepo := repository.NewProductRepo(pool, logger)
	userRepo := repository.NewUserRepo(pool, logger)
	khataRepo := repository.NewKhataRepo(pool, logger)

	batchRepo := repository.NewBatchRepo(pool, logger)
	batchService := services.NewBatchService(batchRepo, shopRepo, productRepo, logger)

	microPlanRepo := repository.NewMicroPlanRepo(pool, logger)
	microPlanService := services.NewMicroPlanService(microPlanRepo, shopRepo, userRepo, khataRepo, logger)

	return &milestone4TestHarness{
		pool:             pool,
		shopRepo:         shopRepo,
		productRepo:      productRepo,
		batchRepo:        batchRepo,
		microPlanRepo:    microPlanRepo,
		batchService:     batchService,
		microPlanService: microPlanService,
	}
}

func TestMilestone4CoreEngines(t *testing.T) {
	h := setupMilestone4Test(t)
	if h == nil || h.pool == nil {
		return
	}
	defer h.pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Find active shop and owner user
	var shopID, ownerUserID string
	err := h.pool.QueryRow(ctx, "SELECT id, user_id FROM shops WHERE is_active = true LIMIT 1;").Scan(&shopID, &ownerUserID)
	if err != nil {
		t.Skip("No active shop found for milestone 4 test")
	}

	// Fetch active product
	var productID, productName string
	var productPrice float64
	err = h.pool.QueryRow(ctx, "SELECT id, name, price FROM products WHERE shop_id = $1 AND is_active = true LIMIT 1;", shopID).
		Scan(&productID, &productName, &productPrice)
	if err != nil {
		t.Skip("No active product found for shop")
	}

	// =========================================================================
	// 1. Batch Expiry Radar & Near-Expiry Auto-Markdown Engine Lifecycle
	// =========================================================================
	t.Run("Batch Expiry & Auto-Markdown Lifecycle", func(t *testing.T) {
		// A: Create a lot expiring in 15 days (Cost ₹80, MRP ₹120, Markdown 25%)
		nearExpiryDate := time.Now().Add(15 * 24 * time.Hour).Format("2006-01-02")
		createReq := dto.CreateBatchRequest{
			ProductID:       productID,
			BatchNumber:     "LOT-2026-EXP15",
			ExpiryDate:      nearExpiryDate,
			Quantity:        20,
			CostPrice:       80.00,
			SellingPrice:    120.00,
			MarkdownPercent: 25.00,
		}
		batch, err := h.batchService.CreateBatch(ctx, ownerUserID, createReq)
		if err != nil {
			t.Fatalf("CreateBatch failed: %v", err)
		}
		if batch.BatchNumber != "LOT-2026-EXP15" {
			t.Errorf("Expected batch LOT-2026-EXP15, got %s", batch.BatchNumber)
		}
		t.Logf("Batch registered! ID=%s, Expiry=%s, Qty=%d, OriginalPrice=₹%.2f", batch.ID, nearExpiryDate, batch.Quantity, batch.OriginalSellingPrice)

		// B: Run near-expiry radar check
		radarList, err := h.batchService.ListNearExpiryBatches(ctx, ownerUserID, 30)
		if err != nil {
			t.Fatalf("ListNearExpiryBatches failed: %v", err)
		}
		if radarList.Count == 0 {
			t.Errorf("Expected at least 1 near-expiry batch in 30-day radar")
		}
		t.Logf("Radar scan detected %d batches expiring within 30 days", radarList.Count)

		// C: Execute Auto-Markdown Engine
		scanResp, err := h.batchService.RunMarkdownScan(ctx, ownerUserID, 30, 25.00)
		if err != nil {
			t.Fatalf("RunMarkdownScan failed: %v", err)
		}
		if scanResp.MarkdownsApplied == 0 {
			t.Errorf("Expected markdowns to be applied for near-expiry lot")
		}
		// 25% off ₹120 = ₹90 (which is above ₹80 cost price)
		expectedDiscountPrice := 90.00
		t.Logf("Auto-markdown executed! MarkdownsApplied=%d, CapitalProtected=₹%.2f", scanResp.MarkdownsApplied, scanResp.CapitalProtectedAmount)

		// D: Verify batch price update
		reloadedBatch, err := h.batchRepo.GetByID(ctx, batch.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if reloadedBatch.CurrentSellingPrice != expectedDiscountPrice {
			t.Errorf("Expected markdown price ₹%.2f, got ₹%.2f", expectedDiscountPrice, reloadedBatch.CurrentSellingPrice)
		}
		if !reloadedBatch.IsMarkdownApplied {
			t.Errorf("Expected is_markdown_applied=true")
		}
		if reloadedBatch.Status != model.BatchStatusNearExpiry {
			t.Errorf("Expected status NEAR_EXPIRY, got %s", reloadedBatch.Status)
		}
		t.Logf("Batch markdown verified! CurrentSellingPrice=₹%.2f, Status=%s", reloadedBatch.CurrentSellingPrice, reloadedBatch.Status)
	})

	// =========================================================================
	// 2. Khata Micro-Installment / Daily Chillar Settle Lifecycle
	// =========================================================================
	t.Run("Khata Micro-Installment Settle Lifecycle", func(t *testing.T) {
		// A: Start a daily micro-repayment plan for ₹1,500 debt at ₹50/day
		planReq := dto.CreateMicroPlanRequest{
			CustomerPhone: "9876543210",
			CustomerName:  "Ramesh Chand Gupta",
			TotalDebt:     1500.00,
			DailyAmount:   50.00,
			Frequency:     "DAILY",
		}
		plan, err := h.microPlanService.CreatePlan(ctx, ownerUserID, planReq)
		if err != nil {
			t.Fatalf("CreatePlan failed: %v", err)
		}
		if plan.TotalInstallments != 30 { // ₹1500 / ₹50 = 30 days
			t.Errorf("Expected 30 total installments, got %d", plan.TotalInstallments)
		}
		if plan.RemainingBalance != 1500.00 {
			t.Errorf("Expected initial remaining balance ₹1500.00, got %.2f", plan.RemainingBalance)
		}
		t.Logf("Micro plan created! ID=%s, TotalInstallments=%d, DailyAmount=₹%.2f", plan.ID, plan.TotalInstallments, plan.DailyAmount)

		// B: Record Installment 1: ₹50 in CASH
		payReq1 := dto.RecordMicroPaymentRequest{
			Amount:      50.00,
			PaymentMode: "CASH",
			Notes:       "Day 1 Counter Chillar Settle",
		}
		summary1, err := h.microPlanService.RecordPayment(ctx, ownerUserID, plan.ID, payReq1)
		if err != nil {
			t.Fatalf("RecordPayment 1 failed: %v", err)
		}
		if summary1.Plan.PaidInstallments != 1 {
			t.Errorf("Expected 1 paid installment, got %d", summary1.Plan.PaidInstallments)
		}
		if summary1.Plan.RemainingBalance != 1450.00 {
			t.Errorf("Expected remaining balance ₹1450.00, got %.2f", summary1.Plan.RemainingBalance)
		}
		t.Logf("Payment 1 logged! Paid=₹%.2f, Remaining=₹%.2f, Progress=%.2f%%",
			summary1.Plan.TotalPaidAmount, summary1.Plan.RemainingBalance, summary1.ProgressPercent)

		// C: Record Final Settlement: ₹1,450 via UPI
		payReq2 := dto.RecordMicroPaymentRequest{
			Amount:               1450.00,
			PaymentMode:          "UPI",
			TransactionReference: "UPI-AXIS-9920194821",
			Notes:                "Full closure via GPay QR",
		}
		summary2, err := h.microPlanService.RecordPayment(ctx, ownerUserID, plan.ID, payReq2)
		if err != nil {
			t.Fatalf("RecordPayment 2 failed: %v", err)
		}
		if summary2.Plan.RemainingBalance != 0.00 {
			t.Errorf("Expected remaining balance ₹0.00, got %.2f", summary2.Plan.RemainingBalance)
		}
		if summary2.Plan.Status != model.PlanStatusCompleted {
			t.Errorf("Expected status COMPLETED, got %s", summary2.Plan.Status)
		}
		if math.Abs(summary2.ProgressPercent-100.00) > 0.01 {
			t.Errorf("Expected progress 100.00%%, got %.2f%%", summary2.ProgressPercent)
		}
		t.Logf("Full debt recovered! Plan Status=%s, Progress=%.2f%%, PaymentsCount=%d",
			summary2.Plan.Status, summary2.ProgressPercent, len(summary2.Payments))
	})
}


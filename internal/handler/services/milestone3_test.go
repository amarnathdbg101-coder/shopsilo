package services_test

import (
	"context"
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

type milestone3TestHarness struct {
	pool          *pgxpool.Pool
	shopRepo      *repository.ShopRepo
	productRepo   *repository.ProductRepo
	scanRepo      *repository.ScanAndGoRepo
	claimRepo     *repository.DamageClaimRepo
	returnRepo    *repository.ReturnLookupRepo
	scanService   *services.ScanAndGoService
	claimService  *services.DamageClaimService
	returnService *services.ReturnLookupService
	posService    *services.POSService
}

func setupMilestone3Test(t *testing.T) *milestone3TestHarness {
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
	khataRepo := repository.NewKhataRepo(pool, logger)
	posRepo := repository.NewPOSRepo(pool, logger)
	drawerRepo := repository.NewCashDrawerRepo(pool, logger)

	posService := services.NewPOSService(posRepo, shopRepo, productRepo, khataRepo)
	posService.SetDrawerRepo(drawerRepo)

	scanRepo := repository.NewScanAndGoRepo(pool, logger)
	scanService := services.NewScanAndGoService(scanRepo, shopRepo, productRepo, posService, logger)

	claimRepo := repository.NewDamageClaimRepo(pool, logger)
	claimService := services.NewDamageClaimService(claimRepo, shopRepo, logger)

	returnRepo := repository.NewReturnLookupRepo(pool, logger)
	returnService := services.NewReturnLookupService(returnRepo, shopRepo, logger)

	return &milestone3TestHarness{
		pool:          pool,
		shopRepo:      shopRepo,
		productRepo:   productRepo,
		scanRepo:      scanRepo,
		claimRepo:     claimRepo,
		returnRepo:    returnRepo,
		scanService:   scanService,
		claimService:  claimService,
		returnService: returnService,
		posService:    posService,
	}
}

func TestMilestone3CoreEngines(t *testing.T) {
	h := setupMilestone3Test(t)
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
		t.Skip("No active shop found for milestone 3 test")
	}

	// Fetch or create an active product for testing
	var productID, productName string
	var productPrice float64
	err = h.pool.QueryRow(ctx, "SELECT id, name, price FROM products WHERE shop_id = $1 AND is_active = true LIMIT 1;", shopID).
		Scan(&productID, &productName, &productPrice)
	if err != nil {
		t.Skip("No active product found for shop")
	}

	// Ensure inventory exists so sale doesn't fail on stock check
	_, _ = h.pool.Exec(ctx, `
		INSERT INTO inventory (product_id, quantity, updated_at)
		VALUES ($1, 100, NOW())
		ON CONFLICT (product_id)
		DO UPDATE SET quantity = GREATEST(inventory.quantity, 50);
	`, productID)

	// =========================================================================
	// 1. Scan & Go Self-Checkout Queue Buster Flow
	// =========================================================================
	t.Run("Scan & Go Queue Buster Lifecycle", func(t *testing.T) {
		// A: Start mobile customer session
		startReq := dto.StartScanSessionRequest{
			ShopID:        shopID,
			CustomerName:  "Vikram Sharma",
			CustomerPhone: "9876543210",
		}
		sessionResp, err := h.scanService.StartSession(ctx, nil, startReq)
		if err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}
		if sessionResp.SessionCode == "" || len(sessionResp.SessionCode) != 6 {
			t.Errorf("Expected 6-digit session code, got '%s'", sessionResp.SessionCode)
		}
		if sessionResp.Status != model.ScanAndGoActive {
			t.Errorf("Expected status ACTIVE, got %s", sessionResp.Status)
		}
		t.Logf("Session started! Code=%s, Token=%s", sessionResp.SessionCode, sessionResp.QRToken)

		// B: Customer scans item in aisle
		addReq := dto.AddItemToCartRequest{
			ProductID: productID,
			Quantity:  2,
		}
		updatedCart, err := h.scanService.AddItemToCart(ctx, sessionResp.SessionID, addReq)
		if err != nil {
			t.Fatalf("AddItemToCart failed: %v", err)
		}
		if updatedCart.TotalItems != 2 {
			t.Errorf("Expected 2 total items, got %d", updatedCart.TotalItems)
		}
		expectedSubtotal := 2 * productPrice
		if updatedCart.Subtotal != expectedSubtotal {
			t.Errorf("Expected subtotal %.2f, got %.2f", expectedSubtotal, updatedCart.Subtotal)
		}
		t.Logf("Item scanned! Total items=%d, Subtotal=₹%.2f", updatedCart.TotalItems, updatedCart.Subtotal)

		// C: Cashier verifies cart at billing counter using 6-digit code
		counterCart, err := h.scanService.VerifySessionAtCounter(ctx, ownerUserID, sessionResp.SessionCode)
		if err != nil {
			t.Fatalf("VerifySessionAtCounter failed: %v", err)
		}
		if counterCart.SessionID != sessionResp.SessionID {
			t.Errorf("Counter fetched mismatched session ID")
		}
		t.Logf("Counter verified customer cart! Customer=%s, Items=%d", counterCart.CustomerName, len(counterCart.Items))

		// D: Cashier executes 1-tap checkout conversion to POS Bill
		convertReq := dto.ConvertToBillRequest{
			PaymentMethod: "cash",
			CustomerPhone: "9876543210",
			CustomerName:  "Vikram Sharma",
		}
		saleResp, err := h.scanService.ConvertToBill(ctx, ownerUserID, sessionResp.SessionCode, convertReq)
		if err != nil {
			t.Fatalf("ConvertToBill failed: %v", err)
		}
		if saleResp == nil || saleResp.Bill == nil {
			t.Fatalf("Expected valid POS bill response")
		}
		t.Logf("Scan & Go cart converted to POS Bill! BillNumber=%s, Total=₹%.2f", saleResp.Bill.BillNumber, saleResp.Bill.TotalAmount)

		// E: Verify session is marked CONVERTED_TO_BILL
		finalSession, err := h.scanService.GetSessionDetails(ctx, sessionResp.SessionID)
		if err != nil {
			t.Fatalf("GetSessionDetails failed: %v", err)
		}
		if finalSession.Status != model.ScanAndGoConvertedToBill {
			t.Errorf("Expected session status CONVERTED_TO_BILL, got %s", finalSession.Status)
		}
		t.Logf("Session marked converted with BillID=%v", finalSession.ConvertedBillID)
	})

	// =========================================================================
	// 2. Company Damage & Expiry Claim Vault Flow
	// =========================================================================
	t.Run("Company Damage Claim Vault Lifecycle", func(t *testing.T) {
		distributor := "Hindustan Unilever Distributor Varanasi"
		expiry := "2026-12-31"

		claimReq := dto.CreateDamageClaimRequest{
			ShopID:           shopID,
			ProductID:        &productID,
			ProductName:      productName,
			BatchNumber:      "HUL-B2026-99",
			ExpiryDate:       &expiry,
			Quantity:         5,
			UnitCost:         40.00,
			ClaimType:        model.ClaimTypeExpired,
			DistributorName:  distributor,
			DistributorPhone: "9988776655",
			SalesRepName:     "Amit Verma (HUL Area Exec)",
			PhotoURL:         "https://shopsilo.storage/claims/hul-expired.jpg",
		}

		claim, err := h.claimService.CreateClaim(ctx, ownerUserID, claimReq)
		if err != nil {
			t.Fatalf("CreateClaim failed: %v", err)
		}
		if claim.TotalLoss != 200.00 { // 5 * ₹40 = ₹200
			t.Errorf("Expected total loss ₹200.00, got ₹%.2f", claim.TotalLoss)
		}
		if claim.Status != model.ClaimStatusPendingClaim {
			t.Errorf("Expected status PENDING_CLAIM, got %s", claim.Status)
		}
		t.Logf("Damage claim filed! ID=%s, Loss=₹%.2f, Distributor=%s", claim.ID, claim.TotalLoss, claim.DistributorName)

		// Distributor Statement generation when Sales Rep arrives
		stmt, err := h.claimService.GetDistributorStatement(ctx, ownerUserID, distributor)
		if err != nil {
			t.Fatalf("GetDistributorStatement failed: %v", err)
		}
		if stmt.PendingClaimsCount == 0 || stmt.PendingLossAmount < 200.00 {
			t.Errorf("Expected at least 1 pending claim with >= ₹200 loss, got %d claims, ₹%.2f loss", stmt.PendingClaimsCount, stmt.PendingLossAmount)
		}
		t.Logf("Distributor statement generated! PendingClaims=%d, TotalLoss=₹%.2f", stmt.PendingClaimsCount, stmt.PendingLossAmount)

		// Resolution: SR issues credit note
		updateReq := dto.UpdateClaimStatusRequest{
			Status:          model.ClaimStatusCreditNoteIssued,
			ResolutionNotes: "Credit note #CN-HUL-482 issued by Amit Verma. Adjusted in invoice.",
		}
		resolved, err := h.claimService.UpdateClaimStatus(ctx, ownerUserID, claim.ID, updateReq)
		if err != nil {
			t.Fatalf("UpdateClaimStatus failed: %v", err)
		}
		if resolved.Status != model.ClaimStatusCreditNoteIssued {
			t.Errorf("Expected status CREDIT_NOTE_ISSUED, got %s", resolved.Status)
		}
		if resolved.ResolvedAt == nil {
			t.Errorf("Expected resolved_at timestamp to be populated")
		}
		t.Logf("Claim resolved! Status=%s, ResolvedAt=%v", resolved.Status, resolved.ResolvedAt)
	})

	// =========================================================================
	// 3. No-Bill Instant Return Lookup Flow
	// =========================================================================
	t.Run("No-Bill Instant Purchase Lookup", func(t *testing.T) {
		filter := dto.ReturnLookupFilter{
			CustomerPhone: "9876543210",
			ProductID:     productID,
			Days:          30,
		}
		res, err := h.returnService.LookupReturn(ctx, ownerUserID, filter)
		if err != nil {
			t.Fatalf("LookupReturn failed: %v", err)
		}
		if res.Count == 0 {
			t.Errorf("Expected at least 1 past purchase transaction for customer phone 9876543210")
		} else {
			firstItem := res.Results[0]
			t.Logf("Past purchase found! BillNumber=%s, BilledAt=%v, SoldUnitPrice=₹%.2f, Quantity=%d",
				firstItem.BillNumber, firstItem.BilledAt, firstItem.UnitPriceSold, firstItem.QuantitySold)
		}
	})
}


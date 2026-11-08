// Package services handles business logic and distributed workflow orchestration.
package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	saga "github.com/amarnathdbg101-coder/go/go-saga"
	"go.uber.org/zap"
)

// OrderCheckoutSagaInput defines the payload for an end-to-end checkout saga.
type OrderCheckoutSagaInput struct {
	ShopOwnerUserID     string
	CustomerPhone       string
	CustomerName        string
	PaymentMethod       string
	Items               []dto.POSSaleItemRequest
	SplitPayments       *dto.POSSplitPayment
	DiscountAmount      float64
	RedeemLoyaltyPoints int
	Notes               string
}

// OrderCheckoutSagaResult contains the completed sale response and saga telemetry.
type OrderCheckoutSagaResult struct {
	SaleResponse  *dto.POSSaleResponse
	ExecutedSteps []string
	SagaDuration  time.Duration
}

// OrderSagaOrchestrator coordinates multi-stage distributed transactions across
// catalog verification, inventory deduction, ledger accounting, and real-time alerts.
// Automatically executes reverse compensations if any downstream step fails.
type OrderSagaOrchestrator struct {
	posRepo     *repository.POSRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
	drawerRepo  *repository.CashDrawerRepo
	khataRepo   *repository.KhataRepo
	logger      *zap.Logger
}

// NewOrderSagaOrchestrator creates a new saga orchestrator instance.
func NewOrderSagaOrchestrator(
	posRepo *repository.POSRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
	drawerRepo *repository.CashDrawerRepo,
	khataRepo *repository.KhataRepo,
	logger *zap.Logger,
) *OrderSagaOrchestrator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &OrderSagaOrchestrator{
		posRepo:     posRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
		drawerRepo:  drawerRepo,
		khataRepo:   khataRepo,
		logger:      logger,
	}
}

// Execute runs the complete order checkout saga sequentially with reverse compensation defense.
func (o *OrderSagaOrchestrator) Execute(ctx context.Context, input OrderCheckoutSagaInput) (*OrderCheckoutSagaResult, error) {
	startTime := time.Now()
	var (
		mu           sync.Mutex
		shop         *model.Shop
		saleResponse *dto.POSSaleResponse
		billCreated  *model.POSBill
		cashRecorded float64
		stepLogs     []string
	)

	coord := saga.New()

	// Observability hook to trace forward and backward compensation events
	coord.OnStepEvent(func(name, phase string, err error) {
		mu.Lock()
		defer mu.Unlock()
		entry := fmt.Sprintf("[%s] step '%s'", phase, name)
		if err != nil {
			entry += fmt.Sprintf(" error: %v", err)
			o.logger.Error("Order checkout saga step event", zap.String("step", name), zap.String("phase", phase), zap.Error(err))
		} else {
			o.logger.Info("Order checkout saga step event", zap.String("step", name), zap.String("phase", phase))
		}
		stepLogs = append(stepLogs, entry)
	})

	// -------------------------------------------------------------
	// STEP 1: Verify Shop and Catalog Pre-conditions
	// -------------------------------------------------------------
	coord.AddStep(
		"VerifyCatalogAndShop",
		func(ctx context.Context) error {
			s, err := o.shopRepo.FindByUserID(ctx, input.ShopOwnerUserID)
			if err != nil {
				return fmt.Errorf("shop verification failed: %w", err)
			}
			if !s.IsActive {
				return errors.New("shop is currently inactive")
			}
			shop = s
			return nil
		},
		nil, // Read-only validation step, no compensation needed
	)

	// -------------------------------------------------------------
	// STEP 2: Execute Core Sale & Inventory Deduction
	// -------------------------------------------------------------
	coord.AddStep(
		"ExecuteSaleAndStockDeduction",
		func(ctx context.Context) error {
			saleReq := dto.CreatePOSSaleRequest{
				Items:               input.Items,
				PaymentMethod:       input.PaymentMethod,
				CustomerPhone:       input.CustomerPhone,
				CustomerName:        input.CustomerName,
				DiscountAmount:      input.DiscountAmount,
				SplitPayments:       input.SplitPayments,
				RedeemLoyaltyPoints: input.RedeemLoyaltyPoints,
			}
			posService := NewPOSService(o.posRepo, o.shopRepo, o.productRepo, o.khataRepo)
			if o.drawerRepo != nil {
				posService.SetDrawerRepo(o.drawerRepo)
			}
			resp, err := posService.CreateSale(ctx, input.ShopOwnerUserID, saleReq)
			if err != nil {
				return fmt.Errorf("sale creation failed: %w", err)
			}
			saleResponse = resp
			if resp != nil && resp.Bill != nil {
				billCreated = resp.Bill
			}
			return nil
		},
		func(ctx context.Context) error {
			// COMPENSATION: If subsequent steps fail, automatically cancel bill and restock inventory
			if billCreated != nil && shop != nil {
				o.logger.Warn("Compensating sale: cancelling bill and restocking inventory",
					zap.String("bill_number", billCreated.BillNumber),
					zap.String("shop_id", shop.ID),
				)
				return o.posRepo.CancelBill(ctx, shop.ID, billCreated.BillNumber, "Saga compensation: subsequent checkout step failed")
			}
			return nil
		},
	)

	// -------------------------------------------------------------
	// STEP 3: Synchronize Physical Cash Drawer Register
	// -------------------------------------------------------------
	coord.AddStep(
		"SyncCashDrawer",
		func(ctx context.Context) error {
			if o.drawerRepo == nil || billCreated == nil || shop == nil {
				return nil
			}
			if billCreated.CashAmount > 0 || billCreated.PaymentMethod == "cash" {
				cashToRecord := billCreated.CashAmount
				if cashToRecord <= 0 && billCreated.PaymentMethod == "cash" {
					cashToRecord = billCreated.TotalAmount
				}
				cashRecorded = cashToRecord
				return o.drawerRepo.AddCashSale(ctx, shop.ID, billCreated.ID, input.ShopOwnerUserID, cashToRecord)
			}
			return nil
		},
		func(ctx context.Context) error {
			// COMPENSATION: Deduct voided cash from drawer register
			if cashRecorded > 0 && shop != nil && billCreated != nil && o.drawerRepo != nil {
				o.logger.Warn("Compensating cash drawer: adjusting voided cash sale", zap.Float64("amount", cashRecorded))
				_ = o.drawerRepo.AddCashSale(ctx, shop.ID, billCreated.ID, input.ShopOwnerUserID, -cashRecorded)
			}
			return nil
		},
	)

	// -------------------------------------------------------------
	// STEP 4: Real-time Storefront & Counter Alerts
	// -------------------------------------------------------------
	coord.AddStep(
		"BroadcastRealtimeEvents",
		func(ctx context.Context) error {
			if billCreated == nil || shop == nil {
				return nil
			}
			wsHub := GetWebSocketHub(nil)
			wsHub.BroadcastToShop(shop.ID, "POS_SALE_COMPLETED", map[string]interface{}{
				"bill_number":    billCreated.BillNumber,
				"total_amount":   billCreated.TotalAmount,
				"payment_method": billCreated.PaymentMethod,
				"customer_phone": billCreated.CustomerPhone,
			})
			return nil
		},
		func(ctx context.Context) error {
			// COMPENSATION: Notify counter that sale was rolled back
			if billCreated != nil && shop != nil {
				wsHub := GetWebSocketHub(nil)
				wsHub.BroadcastToShop(shop.ID, "POS_SALE_CANCELLED", map[string]interface{}{
					"bill_number": billCreated.BillNumber,
					"reason":      "Saga compensation: checkout rollback",
				})
			}
			return nil
		},
	)

	// Run Saga Coordinator!
	if err := coord.Run(ctx); err != nil {
		return nil, fmt.Errorf("order saga execution failed: %w", err)
	}

	return &OrderCheckoutSagaResult{
		SaleResponse:  saleResponse,
		ExecutedSteps: stepLogs,
		SagaDuration:  time.Since(startTime),
	}, nil
}

package services_test

import (
	"context"
	"testing"

	"shopMe/internal/handler/services"

	gogemini "github.com/amarnathdbg101-coder/go/go-gemini"
)

func TestAIFinancialAuditor_NilDB(t *testing.T) {
	auditor := services.NewAIFinancialAuditor(nil, nil)
	ctx := context.Background()

	_, err := auditor.AuditCashReconciliation(ctx, "shop-123")
	if err == nil {
		t.Errorf("Expected error when db is nil for AuditCashReconciliation")
	}

	_, err = auditor.AuditGSTTaxBreakdown(ctx, "shop-123", "today")
	if err == nil {
		t.Errorf("Expected error when db is nil for AuditGSTTaxBreakdown")
	}

	_, err = auditor.AuditDeadStockWorkingCapital(ctx, "shop-123", 30)
	if err == nil {
		t.Errorf("Expected error when db is nil for AuditDeadStockWorkingCapital")
	}
}

func TestAIKhataNudgeEngine_Formatting(t *testing.T) {
	nudgeEngine := services.NewAIKhataNudgeEngine(nil, nil)
	ctx := context.Background()

	_, err := nudgeEngine.GenerateRecoveryNudge(ctx, "shop-123", "", "gentle", "Test Shop", "test@upi")
	if err == nil {
		t.Errorf("Expected error when customer identifier is empty")
	}
}

func TestAIToolDispatcher_AllToolsRegistered(t *testing.T) {
	dispatcher := services.NewAIToolDispatcher(nil, nil, nil, nil)
	tools := dispatcher.GetToolDeclarations()

	if len(tools) == 0 {
		t.Fatalf("Expected tools declaration list")
	}

	funcs := tools[0].FunctionDeclarations
	// We expect 11 tools:
	// 1. get_live_sales_audit
	// 2. get_inventory_health_radar
	// 3. get_overdue_khata_radar
	// 4. search_store_catalog
	// 5. propose_price_update
	// 6. propose_stock_restock
	// 7. propose_khata_entry
	// 8. audit_financial_cash_reconciliation
	// 9. get_tax_gst_breakdown
	// 10. generate_khata_payment_nudge
	// 11. get_dead_stock_clearance_radar
	expectedTools := map[string]bool{
		"get_live_sales_audit":                 false,
		"get_inventory_health_radar":           false,
		"get_overdue_khata_radar":              false,
		"search_store_catalog":                 false,
		"propose_price_update":                 false,
		"propose_stock_restock":                false,
		"propose_khata_entry":                  false,
		"audit_financial_cash_reconciliation": false,
		"get_tax_gst_breakdown":                false,
		"generate_khata_payment_nudge":         false,
		"get_dead_stock_clearance_radar":       false,
	}

	for _, fn := range funcs {
		if _, ok := expectedTools[fn.Name]; ok {
			expectedTools[fn.Name] = true
		}
	}

	for toolName, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool %q to be registered in GetToolDeclarations", toolName)
		}
	}
}

func TestAIToolDispatcher_ExecuteNewToolsWithNilDB(t *testing.T) {
	dispatcher := services.NewAIToolDispatcher(nil, nil, nil, nil)
	ctx := context.Background()

	// 1. Cash Audit
	res, err := dispatcher.ExecuteToolCall(ctx, "shop-1", "user-1", gogemini.FunctionCall{
		Name: "audit_financial_cash_reconciliation",
		Args: map[string]any{},
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall failed: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("Expected error response when db is nil, got: %v", res)
	}

	// 2. GST Tax Breakdown
	res, err = dispatcher.ExecuteToolCall(ctx, "shop-1", "user-1", gogemini.FunctionCall{
		Name: "get_tax_gst_breakdown",
		Args: map[string]any{"period": "today"},
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall failed: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("Expected error response when db is nil, got: %v", res)
	}

	// 3. Khata Nudge
	res, err = dispatcher.ExecuteToolCall(ctx, "shop-1", "user-1", gogemini.FunctionCall{
		Name: "generate_khata_payment_nudge",
		Args: map[string]any{"customer_name_or_phone": "9876543210", "tone": "gentle"},
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall failed: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("Expected error response when db is nil, got: %v", res)
	}

	// 4. Dead Stock Radar
	res, err = dispatcher.ExecuteToolCall(ctx, "shop-1", "user-1", gogemini.FunctionCall{
		Name: "get_dead_stock_clearance_radar",
		Args: map[string]any{"min_days_unsold": 30.0},
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall failed: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("Expected error response when db is nil, got: %v", res)
	}
}

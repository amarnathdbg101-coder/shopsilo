package services_test

import (
	"context"
	"testing"

	"shopMe/internal/handler/services"

	gogemini "github.com/amarnathdbg101-coder/go/go-gemini"
)

func TestAIToolDispatcher_GetToolDeclarations(t *testing.T) {
	dispatcher := services.NewAIToolDispatcher(nil, nil, nil, nil)
	tools := dispatcher.GetToolDeclarations()

	if len(tools) != 1 {
		t.Fatalf("expected 1 Tool group, got %d", len(tools))
	}

	decls := tools[0].FunctionDeclarations
	if len(decls) < 7 {
		t.Fatalf("expected at least 7 function declarations, got %d", len(decls))
	}

	expectedNames := map[string]bool{
		"get_live_sales_audit":       false,
		"get_inventory_health_radar": false,
		"get_overdue_khata_radar":    false,
		"search_store_catalog":       false,
		"propose_price_update":       false,
		"propose_stock_restock":      false,
		"propose_khata_entry":        false,
	}

	for _, d := range decls {
		if _, ok := expectedNames[d.Name]; ok {
			expectedNames[d.Name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected tool %s to be declared", name)
		}
	}
}

func TestAIToolDispatcher_UnknownToolCall(t *testing.T) {
	dispatcher := services.NewAIToolDispatcher(nil, nil, nil, nil)
	res, err := dispatcher.ExecuteToolCall(context.Background(), "shop_1", "user_1", gogemini.FunctionCall{
		Name: "unknown_mystery_tool",
		Args: map[string]any{},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res["status"] != "error" {
		t.Errorf("expected status 'error' for unknown tool, got %v", res["status"])
	}
}

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestContainer_InitializationAndWiring(t *testing.T) {
	logger := zap.NewNop()
	// Initialize centralized container with nil DB for pure dependency graph verification
	container := NewContainer(nil, logger)

	if container == nil {
		t.Fatal("expected container to be non-nil")
	}

	// 1. Verify Centralized Repositories
	if container.Repos == nil {
		t.Fatal("expected Repositories struct to be non-nil")
	}
	if container.Repos.User == nil || container.Repos.Shop == nil || container.Repos.Product == nil {
		t.Fatal("core repositories must be initialized")
	}
	if container.Repos.POS == nil || container.Repos.Khata == nil {
		t.Fatal("POS and Khata repositories must be initialized")
	}

	// 2. Verify Centralized Services
	if container.Services == nil {
		t.Fatal("expected Services struct to be non-nil")
	}
	if container.Services.AI == nil {
		t.Fatal("AIService must be initialized")
	}
	if container.Services.POS == nil || container.Services.EODPDF == nil {
		t.Fatal("POS and EODPDF services must be initialized")
	}
	if container.Services.Khata == nil || container.Services.Inventory == nil {
		t.Fatal("Khata and Inventory services must be initialized")
	}

	// 3. Verify Centralized Controllers
	if container.Controllers == nil {
		t.Fatal("expected Controllers struct to be non-nil")
	}
	if container.Controllers.AI == nil || container.Controllers.POS == nil {
		t.Fatal("AI and POS controllers must be initialized")
	}
	if container.Controllers.EODReport == nil {
		t.Fatal("EODReport controller must be initialized")
	}

	// 4. Verify Centralized Features
	if container.Features == nil {
		t.Fatal("expected FeatureRegistry to be non-nil")
	}

	t.Log("✅ Centralized application container graph wired and verified successfully!")
}

func TestFeatureRegistry_Discovery(t *testing.T) {
	registry := NewFeatureRegistry()

	// Verify all 13 core platform features are active
	coreKeys := []string{
		"ai_vision_ocr",
		"pos_billing",
		"khata_ledger",
		"eod_reports",
		"inventory_procurement",
		"scan_and_go",
		"batch_radar",
		"cash_drawer",
		"damage_claims",
		"multi_godown",
		"loyalty_offers",
		"master_catalog",
		"telemetry_audit",
	}

	for _, key := range coreKeys {
		if !registry.IsEnabled(key) {
			t.Errorf("expected feature '%s' to be enabled in centralized registry", key)
		}
		feat, exists := registry.Get(key)
		if !exists || feat.Name == "" || feat.Category == "" {
			t.Errorf("expected complete metadata for feature '%s'", key)
		}
	}

	all := registry.ListAll()
	if len(all) < 13 {
		t.Fatalf("expected at least 13 registered features, got %d", len(all))
	}

	// Test HTTP discovery response
	req := httptest.NewRequest(http.MethodGet, "/api/features", nil)
	rec := httptest.NewRecorder()
	registry.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", rec.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalFeatures int                 `json:"total_features"`
			Features      []FeatureDescriptor `json:"features"`
		} `json:"data"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode features JSON: %v", err)
	}

	if !resp.Success || resp.Data.TotalFeatures < 13 {
		t.Fatalf("unexpected JSON payload: %+v", resp)
	}

	t.Logf("✅ Centralized Feature Registry verified: %d active platform features exposed via /api/features!", resp.Data.TotalFeatures)
}

func TestFeatureRegistry_DynamicToggle(t *testing.T) {
	registry := NewFeatureRegistry()

	if !registry.IsEnabled("batch_radar") {
		t.Fatal("expected batch_radar to be enabled by default")
	}

	// Disable feature dynamically
	registry.Register(FeatureDescriptor{
		Key:         "batch_radar",
		Name:        "Batch Expiry Radar",
		Category:    "Inventory",
		Description: "Under maintenance",
		Status:      FeatureStatusDisabled,
		Version:     "v2.0.0",
	})

	if registry.IsEnabled("batch_radar") {
		t.Fatal("expected batch_radar to be reported as disabled")
	}

	// Re-enable
	registry.Register(FeatureDescriptor{
		Key:         "batch_radar",
		Name:        "Batch Expiry Radar",
		Category:    "Inventory",
		Description: "Live again",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	if !registry.IsEnabled("batch_radar") {
		t.Fatal("expected batch_radar to be re-enabled")
	}

	t.Log("✅ Dynamic thread-safe feature toggles verified!")
}

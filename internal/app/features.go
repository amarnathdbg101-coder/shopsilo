// Package app provides centralized application state, dependency container, feature registry, and worker coordination.
package app

import (
	"encoding/json"
	"net/http"
	"sync"
)

// FeatureStatus defines whether a feature is active, in preview, or disabled.
type FeatureStatus string

const (
	// FeatureStatusActive indicates the feature is live and operational.
	FeatureStatusActive FeatureStatus = "active"
	// FeatureStatusPreview indicates the feature is in rollout/beta.
	FeatureStatusPreview FeatureStatus = "preview"
	// FeatureStatusDisabled indicates the feature is temporarily disabled.
	FeatureStatusDisabled FeatureStatus = "disabled"
)

// FeatureDescriptor represents a centralized feature specification across ShopSilo.
type FeatureDescriptor struct {
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	Status      FeatureStatus `json:"status"`
	Version     string        `json:"version"`
}

// FeatureRegistry provides centralized discovery, metadata, and toggles for backend features.
type FeatureRegistry struct {
	mu       sync.RWMutex
	features map[string]FeatureDescriptor
}

// NewFeatureRegistry constructs and registers all core centralized ShopSilo platform features.
func NewFeatureRegistry() *FeatureRegistry {
	r := &FeatureRegistry{
		features: make(map[string]FeatureDescriptor),
	}

	// 1. Vision OCR & Multimodal AI
	r.Register(FeatureDescriptor{
		Key:         "ai_vision_ocr",
		Name:        "Smart Studio Vision & Voice Bill",
		Category:    "AI & Intelligence",
		Description: "Camera OCR, supplier bill scanning, Hinglish voice bill, and Parchi parser",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 3. Counter POS Billing
	r.Register(FeatureDescriptor{
		Key:         "pos_billing",
		Name:        "Counter POS Billing & Fast Hotkeys",
		Category:    "Point of Sale",
		Description: "High-speed counter checkout, touch hotkeys, live margin shield, and thermal receipt generation",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 4. Digital Khata & Micro-Installments
	r.Register(FeatureDescriptor{
		Key:         "khata_ledger",
		Name:        "Digital Udhar Ledger & Micro-Installments",
		Category:    "Accounting & Credit",
		Description: "Dual-entry customer passbook, micro-plans (Chillar settle), friendly WhatsApp clarification notes, and automated payment reminders",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 5. Automated End-of-Day P&L PDF
	r.Register(FeatureDescriptor{
		Key:         "eod_reports",
		Name:        "Automated End-of-Day P&L Statements",
		Category:    "Accounting & Audit",
		Description: "High-performance native pure Go gopdf financial audit and profit report generation (<15ms)",
		Status:      FeatureStatusActive,
		Version:     "v2.1.0",
	})

	// 6. Inventory & Wholesale Restock
	r.Register(FeatureDescriptor{
		Key:         "inventory_procurement",
		Name:        "Inventory Radar & Wholesale Procurement",
		Category:    "Inventory & Supply",
		Description: "Low-stock radar alerts, Mandi wholesale checklists, and supplier WhatsApp reorders",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 7. Express Pre-Cart & Counter Queue Buster
	r.Register(FeatureDescriptor{
		Key:         "scan_and_go",
		Name:        "Express Pre-Cart & Counter Queue Buster",
		Category:    "Customer & Counter Flow",
		Description: "Customer or store helper barcode pre-scan with 1-tap counter checkout, preventing long billing queues during peak rush hours",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 8. Mandi Daily Rates & Counter Pricing
	r.Register(FeatureDescriptor{
		Key:         "mandi_pricing",
		Name:        "Mandi Rates & Loose Grains Pricing",
		Category:    "Catalog & Pricing",
		Description: "Loose staples and volatile agri commodities show 'Mandi Daily Rates' or 'Price at Counter', eliminating online stale price disputes",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 8. Batch Expiry Radar & Markdown Engine
	r.Register(FeatureDescriptor{
		Key:         "batch_radar",
		Name:        "Batch Expiry Radar & Clearance Markdown",
		Category:    "Inventory & Loss Prevention",
		Description: "Automatic near-expiry tracking and smart discount clearance markdowns",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 9. Cash Drawer Shift Accountability
	r.Register(FeatureDescriptor{
		Key:         "cash_drawer",
		Name:        "Cash Drawer Shift Accountability",
		Category:    "Cash Management",
		Description: "Cashier open/close sessions, safe drops, petty expenses, and drawer discrepancy audit",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 10. Damage & Expiry Claims Vault
	r.Register(FeatureDescriptor{
		Key:         "damage_claims",
		Name:        "Distributor Damage Claims Vault",
		Category:    "Vendor Relations",
		Description: "Distributor return claim tickets, settlement tracking, and receiptless return lookup",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 11. Multi-Godown Locations & Variant Matrix
	r.Register(FeatureDescriptor{
		Key:         "multi_godown",
		Name:        "Multi-Location Warehouses & Stock Transfers",
		Category:    "Inventory & Warehousing",
		Description: "Godown to retail store requisitions, variant matrix (sizes/colors/shades), and stock transfer status machine",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 12. Customer Loyalty & Offers
	r.Register(FeatureDescriptor{
		Key:         "loyalty_offers",
		Name:        "Customer Loyalty & VIP Offers",
		Category:    "Marketing & Retention",
		Description: "Tiered customer loyalty points, store discount coupons, and counter return management",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 13. Crowdsourced FMCG Master Catalog
	r.Register(FeatureDescriptor{
		Key:         "master_catalog",
		Name:        "Crowdsourced FMCG Barcode Catalog",
		Category:    "Catalog Intelligence",
		Description: "Global FMCG product database with auto-complete for fast product onboarding",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	// 14. Observability & Audit Engine
	r.Register(FeatureDescriptor{
		Key:         "telemetry_audit",
		Name:        "24-Hour Developer Telemetry & Audit Vault",
		Category:    "Observability & Security",
		Description: "Asynchronous PostgreSQL audit logging, error telemetry vault, and live performance metrics",
		Status:      FeatureStatusActive,
		Version:     "v2.0.0",
	})

	return r
}

// Register adds or updates a feature descriptor in the centralized registry.
func (r *FeatureRegistry) Register(feat FeatureDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.features[feat.Key] = feat
}

// Get returns the descriptor for a feature by key.
func (r *FeatureRegistry) Get(key string) (FeatureDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, exists := r.features[key]
	return f, exists
}

// IsEnabled checks if a specific feature is currently active.
func (r *FeatureRegistry) IsEnabled(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, exists := r.features[key]
	return exists && f.Status == FeatureStatusActive
}

// ListAll returns all registered features in a slice.
func (r *FeatureRegistry) ListAll() []FeatureDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]FeatureDescriptor, 0, len(r.features))
	for _, f := range r.features {
		list = append(list, f)
	}
	return list
}

// ServeHTTP writes the registered features as JSON response for discovery.
func (r *FeatureRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	features := r.ListAll()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"total_features": len(features),
			"features":       features,
		},
	})
}

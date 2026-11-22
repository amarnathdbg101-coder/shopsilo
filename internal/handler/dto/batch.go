// Package dto defines request and response payloads.
package dto

import "shopMe/internal/handler/model"

// CreateBatchRequest adds a newly procured batch lot with expiry date.
type CreateBatchRequest struct {
	ProductID       string  `json:"product_id" validate:"required"`
	VariantID       *string `json:"variant_id,omitempty"`
	BatchNumber     string  `json:"batch_number" validate:"required"`
	MfgDate         *string `json:"mfg_date,omitempty"`              // YYYY-MM-DD
	ExpiryDate      string  `json:"expiry_date" validate:"required"` // YYYY-MM-DD
	Quantity        int     `json:"quantity" validate:"required,min=1"`
	CostPrice       float64 `json:"cost_price" validate:"required,min=0"`
	SellingPrice    float64 `json:"selling_price" validate:"required,min=0"`
	MarkdownPercent float64 `json:"markdown_percent"` // Optional default markdown discount (e.g. 20.0%)
}

// RunMarkdownScanResponse reports the outcome of the near-expiry markdown run.
type RunMarkdownScanResponse struct {
	BatchesScanned         int                   `json:"batches_scanned"`
	MarkdownsApplied       int                   `json:"markdowns_applied"`
	ExpiredBatchesCount    int                   `json:"expired_batches_count"`
	CapitalProtectedAmount float64               `json:"capital_protected_amount"`
	Batches                []*model.ProductBatch `json:"batches"`
}

// BatchListResponse contains a list of queried inventory batches.
type BatchListResponse struct {
	Count   int                   `json:"count"`
	Batches []*model.ProductBatch `json:"batches"`
}

// ExpiryRadarSummaryResponse aggregates industrial risk metrics for shop inventory.
type ExpiryRadarSummaryResponse struct {
	TotalLotsTracked      int     `json:"total_lots_tracked"`
	CapitalAtRisk30D      float64 `json:"capital_at_risk_30d"`      // Stock expiring in <= 30 days
	ExpiredCapitalLost    float64 `json:"expired_capital_lost"`     // Stock already expired (< 0 days)
	CriticalCapitalAtRisk float64 `json:"critical_capital_at_risk"` // Stock expiring in 0-7 days
	WarningCapitalAtRisk  float64 `json:"warning_capital_at_risk"`  // Stock expiring in 8-30 days
	ExpiredCount          int     `json:"expired_count"`
	CriticalCount         int     `json:"critical_count"`
	WarningCount          int     `json:"warning_count"`
	UpcomingCount         int     `json:"upcoming_count"`
}

// ApplyBatchMarkdownRequest defines payload for custom markdown discount.
type ApplyBatchMarkdownRequest struct {
	MarkdownPercent float64  `json:"markdown_percent" validate:"required,min=1,max=95"`
	CustomPrice     *float64 `json:"custom_price,omitempty"`
}

// BatchWriteOffRequest logs physical removal or vendor return of expired lots.
type BatchWriteOffRequest struct {
	Reason   string `json:"reason" validate:"required"` // "EXPIRED_DISPOSAL", "VENDOR_RETURN"
	Quantity int    `json:"quantity" validate:"required,min=1"`
}

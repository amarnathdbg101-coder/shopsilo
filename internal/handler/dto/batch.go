// Package dto defines request and response payloads.
package dto

import "shopMe/internal/handler/model"

// CreateBatchRequest adds a newly procured batch lot with expiry date.
type CreateBatchRequest struct {
	ProductID       string  `json:"product_id" validate:"required"`
	VariantID       *string `json:"variant_id,omitempty"`
	BatchNumber     string  `json:"batch_number" validate:"required"`
	MfgDate         *string `json:"mfg_date,omitempty"`   // YYYY-MM-DD
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


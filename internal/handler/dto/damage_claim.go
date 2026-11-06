// Package dto defines request and response payloads.
package dto

import (
	"shopMe/internal/handler/model"
)

// CreateDamageClaimRequest logs a damaged or expired product for credit note or replacement.
type CreateDamageClaimRequest struct {
	ShopID           string          `json:"shop_id"`
	ProductID        *string         `json:"product_id,omitempty"`
	VariantID        *string         `json:"variant_id,omitempty"`
	ProductName      string          `json:"product_name" validate:"required"`
	Barcode          string          `json:"barcode"`
	BatchNumber      string          `json:"batch_number"`
	ExpiryDate       *string         `json:"expiry_date,omitempty"` // YYYY-MM-DD
	Quantity         int             `json:"quantity" validate:"required,min=1"`
	UnitCost         float64         `json:"unit_cost" validate:"required,min=0"`
	ClaimType        model.ClaimType `json:"claim_type" validate:"required"`
	DistributorName  string          `json:"distributor_name" validate:"required"`
	DistributorPhone string          `json:"distributor_phone"`
	SalesRepName     string          `json:"sales_rep_name"`
	PhotoURL         string          `json:"photo_url"`
}

// UpdateClaimStatusRequest updates the claim lifecycle (e.g. SR replaced stock or issued credit note).
type UpdateClaimStatusRequest struct {
	Status          model.ClaimStatus `json:"status" validate:"required"`
	ResolutionNotes string            `json:"resolution_notes"`
}

// DamageClaimSummaryResponse aggregates total claims and recovered amounts per supplier.
type DamageClaimSummaryResponse struct {
	ShopID              string               `json:"shop_id"`
	DistributorName     string               `json:"distributor_name,omitempty"`
	TotalClaimsCount    int                  `json:"total_claims_count"`
	TotalLossAmount     float64              `json:"total_loss_amount"`
	PendingClaimsCount  int                  `json:"pending_claims_count"`
	PendingLossAmount   float64              `json:"pending_loss_amount"`
	RecoveredLossAmount float64              `json:"recovered_loss_amount"`
	Claims              []*model.DamageClaim `json:"claims"`
}

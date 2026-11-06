// Package model contains core database entity structs.
package model

import "time"

// ClaimType defines why an item is declared damaged or unsellable.
type ClaimType string

const (
	ClaimTypeExpired       ClaimType = "EXPIRED"
	ClaimTypeTransitDamage ClaimType = "TRANSIT_DAMAGE"
	ClaimTypeLeakageSpill  ClaimType = "LEAKAGE_SPILL"
	ClaimTypeRatPestBite   ClaimType = "RAT_PEST_BITE"
	ClaimTypeFactoryDefect ClaimType = "FACTORY_DEFECT"
	ClaimTypeOther         ClaimType = "OTHER"
)

// ClaimStatus tracks the recovery status with the company or distributor.
type ClaimStatus string

const (
	ClaimStatusPendingClaim     ClaimStatus = "PENDING_CLAIM"
	ClaimStatusSubmittedToSR    ClaimStatus = "SUBMITTED_TO_SR"
	ClaimStatusReplaced         ClaimStatus = "REPLACED"
	ClaimStatusCreditNoteIssued ClaimStatus = "CREDIT_NOTE_ISSUED"
	ClaimStatusRejected         ClaimStatus = "REJECTED"
)

// DamageClaim represents a logged damaged, broken, or expired merchandise claim.
type DamageClaim struct {
	ID              string      `json:"id"`
	ShopID          string      `json:"shop_id"`
	ProductID       *string     `json:"product_id,omitempty"`
	VariantID       *string     `json:"variant_id,omitempty"`
	ProductName     string      `json:"product_name"`
	Barcode         string      `json:"barcode"`
	BatchNumber     string      `json:"batch_number"`
	ExpiryDate      *time.Time  `json:"expiry_date,omitempty"`
	Quantity        int         `json:"quantity"`
	UnitCost        float64     `json:"unit_cost"`
	TotalLoss       float64     `json:"total_loss"` // Quantity * UnitCost
	ClaimType       ClaimType   `json:"claim_type"`
	DistributorName string      `json:"distributor_name"`
	DistributorPhone string     `json:"distributor_phone"`
	SalesRepName    string      `json:"sales_rep_name"`
	PhotoURL        string      `json:"photo_url"`
	Status          ClaimStatus `json:"status"`
	ResolutionNotes string      `json:"resolution_notes"`
	ResolvedAt      *time.Time  `json:"resolved_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}


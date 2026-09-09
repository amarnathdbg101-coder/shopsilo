package model

import "time"

// Moderation & Reporting Constants
const (
	TargetTypeShop    = "shop"
	TargetTypeProduct = "product"
	TargetTypeReview  = "review"

	ReasonSexualContent     = "sexual_content"
	ReasonCounterfeitFake   = "counterfeit_fake"
	ReasonScamFraud         = "scam_fraud"
	ReasonHarassment        = "harassment"
	ReasonIllegalSubstance  = "illegal_substance"
	ReasonOther             = "other"

	ReportStatusPending     = "pending"
	ReportStatusReviewed    = "reviewed"
	ReportStatusActionTaken = "action_taken"
	ReportStatusDismissed   = "dismissed"

	EntityTypeIP        = "ip"
	EntityTypeDeviceID  = "device_id"
	EntityTypePhone     = "phone"
	EntityTypeImageHash = "image_hash"

	ShopStatusActive        = "active"
	ShopStatusPendingReview = "pending_review"
	ShopStatusFlagged       = "flagged"
	ShopStatusSuspended     = "suspended"
	ShopStatusBanned        = "banned"
)

type BannedEntity struct {
	ID          string    `json:"id"`
	EntityType  string    `json:"entity_type"`
	EntityValue string    `json:"entity_value"`
	Reason      string    `json:"reason"`
	BannedBy    *string   `json:"banned_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ContentReport struct {
	ID                string     `json:"id"`
	ReporterUserID    *string    `json:"reporter_user_id,omitempty"`
	TargetType        string     `json:"target_type"`
	TargetID          string     `json:"target_id"`
	Reason            string     `json:"reason"`
	Details           string     `json:"details,omitempty"`
	Status            string     `json:"status"`
	ReporterIP        string     `json:"reporter_ip,omitempty"`
	ReporterUserAgent string     `json:"reporter_user_agent,omitempty"`
	ActionTaken       string     `json:"action_taken,omitempty"`
	ReviewedBy        *string    `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

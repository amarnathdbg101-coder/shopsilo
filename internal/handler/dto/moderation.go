// Package dto provides request and response structures.
package dto

type CreateReportRequest struct {
	TargetType string `json:"target_type" validate:"required,oneof=shop product review"`
	TargetID   string `json:"target_id" validate:"required,uuid4"`
	Reason     string `json:"reason" validate:"required,oneof=sexual_content counterfeit_fake scam_fraud harassment illegal_substance other"`
	Details    string `json:"details" validate:"omitempty,max=1000"`
}

type ResolveReportRequest struct {
	Status      string `json:"status" validate:"required,oneof=reviewed action_taken dismissed"`
	ActionTaken string `json:"action_taken" validate:"required,max=500"`
}

type BanShopRequest struct {
	Reason          string `json:"reason" validate:"required,min=3,max=500"`
	BlacklistIP      bool   `json:"blacklist_ip"`
	BlacklistDevice  bool   `json:"blacklist_device"`
	BlacklistImages  bool   `json:"blacklist_images"`
}

type AddBannedEntityRequest struct {
	EntityType  string `json:"entity_type" validate:"required,oneof=ip device_id phone image_hash"`
	EntityValue string `json:"entity_value" validate:"required,min=2,max=255"`
	Reason      string `json:"reason" validate:"required,min=3,max=500"`
}

type UpdateShopStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active pending_review flagged suspended banned"`
	Reason string `json:"reason" validate:"omitempty,max=500"`
}

type AdminStatsResponse struct {
	TotalShops      int `json:"total_shops"`
	ActiveShops     int `json:"active_shops"`
	PendingShops    int `json:"pending_shops"`
	FlaggedShops    int `json:"flagged_shops"`
	BannedShops     int `json:"banned_shops"`
	PendingReports  int `json:"pending_reports"`
	TotalReports    int `json:"total_reports"`
	BannedEntities  int `json:"banned_entities"`
	TotalUsers      int `json:"total_users"`
	TotalProducts   int `json:"total_products"`
	TotalCategories int `json:"total_categories"`
}


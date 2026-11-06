// Package dto handles request and response structs.
package dto

import "shopMe/internal/handler/model"

// MarginCheckRequest evaluates real-time bargaining safety on the counter.
type MarginCheckRequest struct {
	ProductID     string   `json:"product_id" validate:"required"`
	VariantID     *string  `json:"variant_id,omitempty"`
	ProposedPrice float64  `json:"proposed_price" validate:"required,gte=0"`
	Quantity      int      `json:"quantity" validate:"omitempty,gt=0"`
}

// MarginCheckResponse returns stealth visual indicators and profit safety for cashier.
type MarginCheckResponse struct {
	IsAllowed       bool    `json:"is_allowed"`
	FloorPrice      float64 `json:"floor_price"`
	CostPrice       float64 `json:"cost_price"`
	NormalPrice     float64 `json:"normal_price"`
	ProposedPrice   float64 `json:"proposed_price"`
	UnitProfit      float64 `json:"unit_profit"`
	TotalProfit     float64 `json:"total_profit"`
	ProfitMarginPct float64 `json:"profit_margin_pct"`
	IndicatorColor  string  `json:"indicator_color"` // "GREEN", "ORANGE", "RED"
	WarningMessage  string  `json:"warning_message,omitempty"`
	StealthBadge    string  `json:"stealth_badge"`   // "🟢 SAFE PROFIT", "🟠 TIGHT MARGIN", "🔴 LOSS ALERT"
}

// CreateShopHotkeyRequest creates a fast-grid tile for rapid checkout.
type CreateShopHotkeyRequest struct {
	ProductID   string  `json:"product_id" validate:"required"`
	VariantID   *string `json:"variant_id,omitempty"`
	DisplayName string  `json:"display_name" validate:"required,min=1,max=100"`
	ColorHex    string  `json:"color_hex" validate:"omitempty,max=10"`
	SortOrder   int     `json:"sort_order"`
	TimeSlot    string  `json:"time_slot" validate:"omitempty,oneof=ALL_DAY MORNING EVENING"`
}

// UpdateShopHotkeyRequest updates an existing fast-grid tile.
type UpdateShopHotkeyRequest struct {
	DisplayName *string `json:"display_name,omitempty" validate:"omitempty,min=1,max=100"`
	ColorHex    *string `json:"color_hex,omitempty" validate:"omitempty,max=10"`
	SortOrder   *int    `json:"sort_order,omitempty"`
	TimeSlot    *string `json:"time_slot,omitempty" validate:"omitempty,oneof=ALL_DAY MORNING EVENING"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ReorderHotkeysRequest rearranges hotkeys on the cashier grid.
type ReorderHotkeysRequest struct {
	HotkeyIDs []string `json:"hotkey_ids" validate:"required,min=1"`
}

// ShopHotkeysListResponse returns active fast-grid tiles for cashier screen.
type ShopHotkeysListResponse struct {
	Hotkeys []*model.ShopHotkey `json:"hotkeys"`
	Count   int                 `json:"count"`
}


// Package dto handles request and response structs.
package dto

import "shopMe/internal/handler/model"

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


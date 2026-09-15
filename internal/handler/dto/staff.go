// Package dto handle request and response structs.
package dto

type CreateStaffRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Phone    string `json:"phone" validate:"required,min=10,max=20"`
	PIN      string `json:"pin" validate:"required,len=4,numeric"` // 4-digit PIN for quick counter login
	Role     string `json:"role" validate:"required,oneof=cashier manager"`
}

type UpdateStaffRequest struct {
	FullName *string `json:"full_name,omitempty" validate:"omitempty,min=2,max=100"`
	Phone    *string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	PIN      *string `json:"pin,omitempty" validate:"omitempty,len=4,numeric"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=cashier manager"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type StaffLoginRequest struct {
	ShopID string `json:"shop_id" validate:"required"`
	Phone  string `json:"phone" validate:"required"`
	PIN    string `json:"pin" validate:"required,len=4,numeric"`
}

type StaffLoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	StaffID     string `json:"staff_id"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
	ShopID      string `json:"shop_id"`
}

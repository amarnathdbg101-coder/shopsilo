// Package model handle database structs.
package model

import "time"

const (
	StaffRoleCashier = "cashier"
	StaffRoleManager = "manager"
)

type ShopStaff struct {
	ID        string    `json:"id"`
	ShopID    string    `json:"shop_id"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	PinHash   string    `json:"-"` // Never expose
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

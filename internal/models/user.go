package models

import (
	"time"
)

type User struct {
	ID          int        `json:"id"`
	FirebaseUID string     `json:"firebase_uid"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Shop        *Shop      `json:"shop,omitempty"`
}

type Shop struct {
	ID           int       `json:"id"`
	UserID       int       `json:"owner_id"`
	ShopCustomID string    `json:"shop_custom_id"`
	Name         string    `json:"name"`
}

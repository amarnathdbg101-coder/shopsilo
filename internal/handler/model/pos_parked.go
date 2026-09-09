// Package model handles database structs.
package model

import (
	"time"
)

// POSParkedBill represents a temporarily held cart at the billing counter.
type POSParkedBill struct {
	ID            string      `json:"id"`
	ShopID        string      `json:"shop_id"`
	Label         string      `json:"label"`
	CustomerPhone string      `json:"customer_phone"`
	CartData      interface{} `json:"cart_data"`
	TotalAmount   float64     `json:"total_amount"`
	CreatedAt     time.Time   `json:"created_at"`
}

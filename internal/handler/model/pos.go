// Package model handle database struct.
package model

import "time"

type POSBill struct {
	ID             string         `json:"id"`
	ShopID         string         `json:"shop_id"`
	BillNumber     string         `json:"bill_number"`
	CustomerPhone  string         `json:"customer_phone,omitempty"`
	CustomerUserID *string        `json:"customer_user_id,omitempty"`
	Subtotal       float64        `json:"subtotal"`
	DiscountAmount float64        `json:"discount_amount"`
	TotalAmount    float64        `json:"total_amount"`
	TotalCost      float64        `json:"total_cost"`
	NetProfit      float64        `json:"net_profit"`
	PaymentMethod  string         `json:"payment_method"` // 'cash', 'upi', 'card'
	CreatedAt      time.Time      `json:"created_at"`
	Items          []*POSBillItem `json:"items,omitempty"`
	Shop           *Shop          `json:"shop,omitempty"`
}

type POSBillItem struct {
	ID          string  `json:"id"`
	BillID      string  `json:"bill_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ProductSKU  string  `json:"product_sku,omitempty"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	UnitCost    float64 `json:"unit_cost"`
	TotalPrice  float64 `json:"total_price"`
}

type DailySalesSummary struct {
	Date         string  `json:"date"`
	TotalBills   int     `json:"total_bills"`
	TotalRevenue float64 `json:"total_revenue"`
	TotalCost    float64 `json:"total_cost"`
	TotalProfit  float64 `json:"total_profit"`
	CashTotal    float64 `json:"cash_total"`
	UPITotal     float64 `json:"upi_total"`
	CardTotal    float64 `json:"card_total"`
}

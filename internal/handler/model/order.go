package model

import (
	"time"
)

type (
	OrderStatus   string
	PaymentStatus string
)

const (
	OrderPending    OrderStatus = "pending"
	OrderConfirmed  OrderStatus = "confirmed"
	OrderProcessing OrderStatus = "processing"
	OrderShipped    OrderStatus = "shipped"
	OrderDelivered  OrderStatus = "delivered"
	OrderCancelled  OrderStatus = "cancelled"
	OrderRefunded   OrderStatus = "refunded"
)

const (
	PaymentPending  PaymentStatus = "pending"
	PaymentPaid     PaymentStatus = "paid"
	PaymentFailed   PaymentStatus = "failed"
	PaymentRefunded PaymentStatus = "refunded"
)

type Address struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Line1    string `json:"line1"`
	Line2    string `json:"line2,omitempty"`
	City     string `json:"city"`
	State    string `json:"state"`
	Pincode  string `json:"pincode"`
	Country  string `json:"country"`
}

type Order struct {
	ID              string        `json:"id"`
	OrderNumber     string        `json:"order_number"`
	UserID          string        `json:"user_id"`
	User            *User         `json:"user,omitempty"`
	Status          OrderStatus   `json:"status"`
	PaymentStatus   PaymentStatus `json:"payment_status"`
	Subtotal        float64       `json:"subtotal"`
	TaxAmount       float64       `json:"tax_amount"`
	ShippingAmount  float64       `json:"shipping_amount"`
	DiscountAmount  float64       `json:"discount_amount"`
	TotalAmount     float64       `json:"total_amount"`
	ShippingAddress Address       `json:"shipping_address"`
	BillingAddress  Address       `json:"billing_address,omitempty"`
	Notes           string        `json:"notes,omitempty"`
	Items           []*OrderItem  `json:"items"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type OrderItem struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

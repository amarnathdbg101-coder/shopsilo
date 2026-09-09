// Package dto handle request and response struct.
package dto

type AdjustStockRequest struct {
	ProductID         string `json:"product_id" validate:"required,uuid"`
	Adjustment        int    `json:"adjustment" validate:"required"` // e.g. +50 or -5
	LowStockThreshold *int   `json:"low_stock_threshold,omitempty" validate:"omitempty,gte=0,lte=1000"`
	Notes             string `json:"notes,omitempty" validate:"omitempty,max=200"`
}

type LowStockProduct struct {
	ProductID           string  `json:"product_id"`
	Name                string  `json:"name"`
	SKU                 string  `json:"sku"`
	CurrentStock        int     `json:"current_stock"`
	ReservedStock       int     `json:"reserved_stock"`
	AvailableStock      int     `json:"available_stock"`
	LowStockThreshold   int     `json:"low_stock_threshold"`
	Price               float64 `json:"price"`
	SuggestedReorderQty int     `json:"suggested_reorder_qty"`
}

type LowStockResponse struct {
	TotalLowStockItems int                `json:"total_low_stock_items"`
	Items              []*LowStockProduct `json:"items"`
}

type SupplierReorderWhatsAppResponse struct {
	SupplierPhone string `json:"supplier_phone"`
	ItemsCount    int    `json:"items_count"`
	OrderMessage  string `json:"order_message"`
	WhatsAppURL   string `json:"whatsapp_url"`
}

// CreateStockAlertRequest is submitted by shoppers to get notified when an out-of-stock item is back.
type CreateStockAlertRequest struct {
	CustomerPhone string `json:"customer_phone" validate:"required,min=10,max=15"`
	CustomerName  string `json:"customer_name,omitempty" validate:"omitempty,max=100"`
}

// DemandWatchlistItem represents an out-of-stock or low-stock product with customer pre-orders/interest.
type DemandWatchlistItem struct {
	ProductID             string `json:"product_id"`
	ProductName           string `json:"product_name"`
	SKU                   string `json:"sku"`
	CurrentStock          int    `json:"current_stock"`
	WaitingCustomersCount int    `json:"waiting_customers_count"`
	WhatsAppBroadcastCopy string `json:"whatsapp_broadcast_copy"`
	WhatsAppBroadcastURL  string `json:"whatsapp_broadcast_url"`
}

// DemandWatchlistResponse summarizes all unmet customer demand for wholesale purchase prioritization.
type DemandWatchlistResponse struct {
	TotalDemandItems      int                    `json:"total_demand_items"`
	TotalWaitingCustomers int                    `json:"total_waiting_customers"`
	Items                 []*DemandWatchlistItem `json:"items"`
}

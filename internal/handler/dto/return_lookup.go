// Package dto defines request and response payloads.
package dto

import "time"

// ReturnLookupFilter specifies criteria to locate a past sale without a physical bill.
type ReturnLookupFilter struct {
	ShopID        string `json:"shop_id"`
	CustomerPhone string `json:"customer_phone"`
	Barcode       string `json:"barcode"`
	ProductID     string `json:"product_id"`
	BillNumber    string `json:"bill_number"`
	Days          int    `json:"days"` // How many past days to look back (default 30)
}

// ReturnLookupItem contains exact details of an item sold previously.
type ReturnLookupItem struct {
	BillID        string    `json:"bill_id"`
	BillNumber    string    `json:"bill_number"`
	BilledAt      time.Time `json:"billed_at"`
	CustomerName  string    `json:"customer_name"`
	CustomerPhone string    `json:"customer_phone"`
	ProductID     *string   `json:"product_id,omitempty"`
	VariantID     *string   `json:"variant_id,omitempty"`
	ProductName   string    `json:"product_name"`
	QuantitySold  int       `json:"quantity_sold"`
	UnitPriceSold float64   `json:"unit_price_sold"` // Critical: Refund must match sold price, not updated MRP
	UnitCost      float64   `json:"unit_cost"`
	TotalPrice    float64   `json:"total_price"`
	PaymentMethod string    `json:"payment_method"`
	BillStatus    string    `json:"bill_status"`
}

// ReturnLookupResponse bundles matching past transactions for customer verification.
type ReturnLookupResponse struct {
	Count   int                 `json:"count"`
	Results []*ReturnLookupItem `json:"results"`
}


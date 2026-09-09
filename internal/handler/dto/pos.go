// Package dto handles request and response struct.
package dto

import "shopMe/internal/handler/model"

type POSSaleItemRequest struct {
	ProductID   string   `json:"product_id" validate:"required,uuid"`
	Quantity    int      `json:"quantity" validate:"required,gt=0"`
	CustomPrice *float64 `json:"custom_price,omitempty" validate:"omitempty,gte=0"`
}

type POSSplitPayment struct {
	CashAmount   float64 `json:"cash_amount" validate:"gte=0"`
	OnlineAmount float64 `json:"online_amount" validate:"gte=0"`
	KhataAmount  float64 `json:"khata_amount" validate:"gte=0"`
}

type CreatePOSSaleRequest struct {
	CustomerPhone       string               `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	CustomerName        string               `json:"customer_name,omitempty" validate:"omitempty,max=100"`
	Items               []POSSaleItemRequest `json:"items" validate:"required,min=1,dive"`
	DiscountAmount      float64              `json:"discount_amount" validate:"gte=0"`
	PaymentMethod       string               `json:"payment_method" validate:"required,oneof=cash upi card credit split"`
	SplitPayments       *POSSplitPayment     `json:"split_payments,omitempty"`
	BargainDealCode     string               `json:"bargain_deal_code,omitempty"`
	RedeemLoyaltyPoints int                  `json:"redeem_loyalty_points,omitempty" validate:"omitempty,gte=0"`
}

type POSSaleResponse struct {
	Bill                  *model.POSBill `json:"bill"`
	ReceiptURL            string         `json:"receipt_url"`
	LoyaltyPointsCredited int            `json:"loyalty_points_credited"`
	LoyaltyPointsRedeemed int            `json:"loyalty_points_redeemed,omitempty"`
	LoyaltyDiscountAmount float64        `json:"loyalty_discount_amount,omitempty"`
	WhatsAppShareURL      string         `json:"whatsapp_share_url,omitempty"`
}

type BarcodeScanResponse struct {
	ProductID         string   `json:"product_id"`
	ShopID            string   `json:"shop_id"`
	Name              string   `json:"name"`
	SKU               string   `json:"sku"`
	Price             float64  `json:"price"`
	Images            []string `json:"images"`
	AvailableQuantity int      `json:"available_quantity"`
	IsActive          bool     `json:"is_active"`
}

type POSBillShareResponse struct {
	BillNumber       string  `json:"bill_number"`
	CustomerPhone    string  `json:"customer_phone"`
	MaskedPhone      string  `json:"masked_phone"`
	TotalAmount      float64 `json:"total_amount"`
	ReceiptURL       string  `json:"receipt_url"`
	WhatsAppShareURL string  `json:"whatsapp_share_url"`
}

type DailyCloseReport struct {
	Date                 string  `json:"date"`
	CashSales            float64 `json:"cash_sales"`
	UPISales             float64 `json:"upi_sales"`
	CardSales            float64 `json:"card_sales"`
	CreditGiven          float64 `json:"credit_given"`
	TotalGrossSales      float64 `json:"total_gross_sales"`
	TotalBillsCount      int     `json:"total_bills_count"`
	KhataCashCollected   float64 `json:"khata_cash_collected"`
	KhataUPICollected    float64 `json:"khata_upi_collected"`
	TotalKhataCollected  float64 `json:"total_khata_collected"`
	CashExpensesPaid     float64 `json:"cash_expenses_paid"`
	TotalExpensesPaid    float64 `json:"total_expenses_paid"`
	ExpectedCashInDrawer float64 `json:"expected_cash_in_drawer"`
}

// TaxSlabSummary breaks down sales and GST tax for a given tax slab rate (e.g. 5%, 12%, 18%).
type TaxSlabSummary struct {
	TaxRatePct    float64 `json:"tax_rate_pct"`
	TaxableAmount float64 `json:"taxable_amount"`
	CGSTAmount    float64 `json:"cgst_amount"`
	SGSTAmount    float64 `json:"sgst_amount"`
	TotalTax      float64 `json:"total_tax"`
	TotalGross    float64 `json:"total_gross"`
}

// MonthlyGSTReport provides a complete turnover and tax computation ready for CA / GST-3B filing.
type MonthlyGSTReport struct {
	ShopID           string           `json:"shop_id"`
	ShopName         string           `json:"shop_name"`
	Month            int              `json:"month"`
	Year             int              `json:"year"`
	MonthName        string           `json:"month_name"`
	TotalBills       int              `json:"total_bills"`
	TotalGrossSales  float64          `json:"total_gross_sales"`
	TotalTaxable     float64          `json:"total_taxable_amount"`
	TotalCGST        float64          `json:"total_cgst"`
	TotalSGST        float64          `json:"total_sgst"`
	TotalTax         float64          `json:"total_tax"`
	Slabs            []TaxSlabSummary `json:"slabs"`
	SummaryTextForCA string           `json:"summary_text_for_ca"`
	WhatsAppShareURL string           `json:"whatsapp_share_url"`
	GeneratedAt      string           `json:"generated_at"`
}

// ParkPOSBillRequest is used by the cashier to temporarily hold an active counter sale cart.
type ParkPOSBillRequest struct {
	Label          string               `json:"label,omitempty" validate:"omitempty,max=100"`
	CustomerPhone  string               `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	Items          []POSSaleItemRequest `json:"items" validate:"required,min=1,dive"`
	DiscountAmount float64              `json:"discount_amount" validate:"gte=0"`
	PaymentMethod  string               `json:"payment_method" validate:"omitempty,oneof=cash upi card credit"`
}

// ParkedBillSummaryItem is returned when listing all currently held carts on the counter.
type ParkedBillSummaryItem struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	CustomerPhone string  `json:"customer_phone"`
	MaskedPhone   string  `json:"masked_phone"`
	TotalItems    int     `json:"total_items"`
	TotalAmount   float64 `json:"total_amount"`
	ParkedAt      string  `json:"parked_at"`
}

// ParkedBillDetailResponse returns the full held cart so the POS frontend can resume billing.
type ParkedBillDetailResponse struct {
	ID             string               `json:"id"`
	ShopID         string               `json:"shop_id"`
	Label          string               `json:"label"`
	CustomerPhone  string               `json:"customer_phone"`
	Items          []POSSaleItemRequest `json:"items"`
	DiscountAmount float64              `json:"discount_amount"`
	PaymentMethod  string               `json:"payment_method"`
	TotalAmount    float64              `json:"total_amount"`
	ParkedAt       string               `json:"parked_at"`
}

// ParseParchiRequest carries raw multiline grocery list text copied from WhatsApp or written by customer.
type ParseParchiRequest struct {
	RawText string `json:"raw_text" validate:"required,min=2"`
}

// ParsedParchiItem is a single successfully recognized grocery item mapped to the shop inventory.
type ParsedParchiItem struct {
	ProductID         string  `json:"product_id"`
	ProductName       string  `json:"product_name"`
	SKU               string  `json:"sku"`
	RequestedQuantity int     `json:"requested_quantity"`
	ParsedUnit        string  `json:"parsed_unit"`
	UnitPrice         float64 `json:"unit_price"`
	TotalPrice        float64 `json:"total_price"`
	AvailableStock    int     `json:"available_stock"`
	InStock           bool    `json:"in_stock"`
}

// UnmatchedParchiLine records text lines that could not be confidently identified in the store.
type UnmatchedParchiLine struct {
	OriginalLine string `json:"original_line"`
	QueryTerm    string `json:"query_term"`
	Reason       string `json:"reason"`
}

// ParseParchiResponse returns parsed items and an immediately usable draft cart.
type ParseParchiResponse struct {
	TotalLinesParsed     int                    `json:"total_lines_parsed"`
	MatchedCount         int                    `json:"matched_count"`
	UnmatchedCount       int                    `json:"unmatched_count"`
	EstimatedTotalAmount float64                `json:"estimated_total_amount"`
	MatchedItems         []*ParsedParchiItem    `json:"matched_items"`
	UnmatchedLines       []*UnmatchedParchiLine `json:"unmatched_lines"`
	ReadyCart            *CreatePOSSaleRequest  `json:"ready_cart"`
}

// WeeklyTopProductItem tracks fast-moving inventory in weekly reporting.
type WeeklyTopProductItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitsSold   int     `json:"units_sold"`
	TotalSales  float64 `json:"total_sales"`
}

// WeeklyScorecardResponse gives store owners high-level business health and cash flow telemetry.
type WeeklyScorecardResponse struct {
	ShopID                 string                  `json:"shop_id"`
	ShopName               string                  `json:"shop_name"`
	CurrentWeekRange       string                  `json:"current_week_range"`
	PreviousWeekRange      string                  `json:"previous_week_range"`
	CurrentWeekRevenue     float64                 `json:"current_week_revenue"`
	PreviousWeekRevenue    float64                 `json:"previous_week_revenue"`
	GrowthPercentage       float64                 `json:"growth_percentage"`
	TotalBillsCount        int                     `json:"total_bills_count"`
	AverageOrderValue      float64                 `json:"average_order_value"`
	CashCollected          float64                 `json:"cash_collected"`
	OnlineCollected        float64                 `json:"online_collected"`
	KhataNewCreditIssued   float64                 `json:"khata_new_credit_issued"`
	KhataRecoveredCash     float64                 `json:"khata_recovered_cash"`
	NetKhataCashFlow       float64                 `json:"net_khata_cash_flow"`
	KhataHealthStatus      string                  `json:"khata_health_status"`
	TopSellingProducts     []*WeeklyTopProductItem `json:"top_selling_products"`
	OutOfStockSellersCount int                     `json:"out_of_stock_sellers_count"`
	WhatsAppSummaryCopy    string                  `json:"whatsapp_summary_copy"`
	WhatsAppShareURL       string                  `json:"whatsapp_share_url"`
	GeneratedAt            string                  `json:"generated_at"`
}

// CustomerRecentBasketItem represents an item previously bought by the customer.
type CustomerRecentBasketItem struct {
	ProductID      string  `json:"product_id"`
	ProductName    string  `json:"product_name"`
	SKU            string  `json:"sku"`
	Quantity       int     `json:"quantity"`
	LastUnitPrice  float64 `json:"last_unit_price"`
	CurrentPrice   float64 `json:"current_price"`
	AvailableStock int     `json:"available_stock"`
	InStock        bool    `json:"in_stock"`
}

// CustomerRecentBasketResponse returns the customer's previous grocery basket with 1-click reorder cart.
type CustomerRecentBasketResponse struct {
	CustomerPhone        string                      `json:"customer_phone"`
	LastBillNumber       string                      `json:"last_bill_number"`
	LastBillDate         string                      `json:"last_bill_date"`
	TotalItemsCount      int                         `json:"total_items_count"`
	EstimatedTotalAmount float64                     `json:"estimated_total_amount"`
	Items                []*CustomerRecentBasketItem `json:"items"`
	ReadyCart            *CreatePOSSaleRequest       `json:"ready_cart"`
}

// POSReturnItemRequest is a line item to return from an existing POS bill.
type POSReturnItemRequest struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

// ProcessPOSReturnRequest allows a shopkeeper to process a return against an existing bill.
type ProcessPOSReturnRequest struct {
	BillNumber string                 `json:"bill_number" validate:"required"`
	Items      []POSReturnItemRequest `json:"items" validate:"required,min=1,dive"`
	RefundMode string                 `json:"refund_mode" validate:"required,oneof=cash khata credit_note"`
	Reason     string                 `json:"reason,omitempty" validate:"omitempty,max=255"`
}

// ProcessPOSReturnResponse returns confirmation of restocked items and issued refund or credit note.
type ProcessPOSReturnResponse struct {
	ReturnNumber        string  `json:"return_number"`
	BillNumber          string  `json:"bill_number"`
	CustomerPhone       string  `json:"customer_phone"`
	RefundMode          string  `json:"refund_mode"`
	TotalRefundAmount   float64 `json:"total_refund_amount"`
	RestockedItemsCount int     `json:"restocked_items_count"`
	CreditNoteCode      string  `json:"credit_note_code,omitempty"`
	Message             string  `json:"message"`
	WhatsAppReceiptURL  string  `json:"whatsapp_receipt_url,omitempty"`
	ProcessedAt         string  `json:"processed_at"`
}



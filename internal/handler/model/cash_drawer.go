// Package model defines database entity models.
package model

import "time"

type CashDrawerSessionStatus string

const (
	DrawerStatusOpen        CashDrawerSessionStatus = "OPEN"
	DrawerStatusClosed      CashDrawerSessionStatus = "CLOSED"
	DrawerStatusDiscrepancy CashDrawerSessionStatus = "DISCREPANCY"
)

type CashDrawerEventType string

const (
	DrawerEventOpeningFloat CashDrawerEventType = "OPENING_FLOAT"
	DrawerEventCashSale     CashDrawerEventType = "CASH_SALE"
	DrawerEventSafeDrop     CashDrawerEventType = "SAFE_DROP"
	DrawerEventPettyExpense CashDrawerEventType = "PETTY_EXPENSE"
	DrawerEventManualOpen   CashDrawerEventType = "MANUAL_OPEN"
	DrawerEventBillVoid     CashDrawerEventType = "BILL_VOID"
	DrawerEventShiftClose   CashDrawerEventType = "SHIFT_CLOSE"
)

// CashDrawerSession represents a physical cash drawer shift for staff accountability.
type CashDrawerSession struct {
	ID                string                  `json:"id"`
	ShopID            string                  `json:"shop_id"`
	StaffUserID       string                  `json:"staff_user_id"`
	StaffName         string                  `json:"staff_name,omitempty"`
	ShiftName         string                  `json:"shift_name"`
	OpenedAt          time.Time               `json:"opened_at"`
	ClosedAt          *time.Time              `json:"closed_at,omitempty"`
	OpeningCash       float64                 `json:"opening_cash"`
	CashSalesTotal    float64                 `json:"cash_sales_total"`
	SafeDropTotal     float64                 `json:"safe_drop_total"`
	ExpensesTotal     float64                 `json:"expenses_total"`
	ExpectedCash      float64                 `json:"expected_cash"`
	ActualCash        *float64                `json:"actual_cash,omitempty"`
	DiscrepancyAmount float64                 `json:"discrepancy_amount"`
	SafeDropThreshold float64                 `json:"safe_drop_threshold"`
	Status            CashDrawerSessionStatus `json:"status"`
	ClosingNotes      string                  `json:"closing_notes"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

// CashDrawerEvent represents an individual cash movement event inside the drawer.
type CashDrawerEvent struct {
	ID          string              `json:"id"`
	SessionID   string              `json:"session_id"`
	ShopID      string              `json:"shop_id"`
	StaffUserID string              `json:"staff_user_id,omitempty"`
	StaffName   string              `json:"staff_name,omitempty"`
	EventType   CashDrawerEventType `json:"event_type"`
	Amount      float64             `json:"amount"`
	BillID      *string             `json:"bill_id,omitempty"`
	Category    string              `json:"category"`
	Reason      string              `json:"reason"`
	CreatedAt   time.Time           `json:"created_at"`
}


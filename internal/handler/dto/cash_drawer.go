// Package dto handles request and response structs.
package dto

import "shopMe/internal/handler/model"

// OpenDrawerSessionRequest starts a new cashier shift with an opening cash float.
type OpenDrawerSessionRequest struct {
	OpeningCash       float64  `json:"opening_cash" validate:"gte=0"`
	ShiftName         string   `json:"shift_name" validate:"omitempty,max=50"`
	SafeDropThreshold *float64 `json:"safe_drop_threshold,omitempty" validate:"omitempty,gt=0"`
}

// CloseDrawerSessionRequest reconciles physical cash at counter close.
type CloseDrawerSessionRequest struct {
	ActualCash   float64 `json:"actual_cash" validate:"gte=0"`
	ClosingNotes string  `json:"closing_notes" validate:"omitempty,max=500"`
}

// RecordSafeDropRequest moves surplus cash from register to secure safe tijori.
type RecordSafeDropRequest struct {
	Amount float64 `json:"amount" validate:"required,gt=0"`
	Notes  string  `json:"notes" validate:"omitempty,max=255"`
}

// RecordPettyExpenseRequest logs instant petty cash out (e.g. ₹50 tea, petrol, bags).
type RecordPettyExpenseRequest struct {
	Amount   float64 `json:"amount" validate:"required,gt=0"`
	Category string  `json:"category" validate:"omitempty,max=100"`
	Reason   string  `json:"reason" validate:"required,min=2,max=255"`
}

// CurrentDrawerStatusResponse returns live cash metrics for cashier & owner.
type CurrentDrawerStatusResponse struct {
	Session                   *model.CashDrawerSession `json:"session"`
	CurrentDrawerCash         float64                  `json:"current_drawer_cash"`
	SafeDropRecommended       bool                     `json:"safe_drop_recommended"`
	SafeDropRecommendedAmount float64                  `json:"safe_drop_recommended_amount"`
	RecentEvents              []*model.CashDrawerEvent `json:"recent_events"`
}

// DrawerSessionsHistoryResponse lists shift closures and discrepancy logs.
type DrawerSessionsHistoryResponse struct {
	Sessions []*model.CashDrawerSession `json:"sessions"`
	Count    int                        `json:"count"`
}


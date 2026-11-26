// Package model defines database entities and domain structs.
package model

import (
	"encoding/json"
	"time"
)

// ProductIntelligence stores ABC-XYZ tiers, velocity, dead capital, and semantic attributes.
type ProductIntelligence struct {
	ID                  string          `json:"id"`
	ProductID           string          `json:"product_id"`
	ShopID              string          `json:"shop_id"`
	ProductName         string          `json:"product_name,omitempty"`
	ProductPrice        float64         `json:"product_price,omitempty"`
	CostPrice           float64         `json:"cost_price,omitempty"`
	FloorPrice          float64         `json:"floor_price,omitempty"`
	CurrentStock        int             `json:"current_stock,omitempty"`
	CategoryName        string          `json:"category_name,omitempty"`
	Attributes          json.RawMessage `json:"attributes"`
	SearchKeywords      string          `json:"search_keywords"`
	ABCTier             string          `json:"abc_tier"` // A (80% rev), B (15%), C (5%)
	XYZTier             string          `json:"xyz_tier"` // X (stable), Y (seasonal), Z (erratic/dead)
	VelocityDaily       float64         `json:"velocity_daily"`
	DaysOfSupply        int             `json:"days_of_supply"`
	DaysSinceLastSale   int             `json:"days_since_last_sale"`
	IsDeadStock         bool            `json:"is_dead_stock"`
	CarryingCostMonthly float64         `json:"carrying_cost_monthly"` // Opportunity cost at 1.5%/month
	PriceElasticity     float64         `json:"price_elasticity"`
	IsPriceElastic      bool            `json:"is_price_elastic"`
	ClearanceStage      int             `json:"clearance_stage"` // 0: Normal, 1: Mild promo, 2: Deep clearance, 3: Dead stock liquidation
	RecommendedAction   string          `json:"recommended_action"`
	RecommendedMinPrice float64         `json:"recommended_min_price"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// CustomerPsychology stores RFM segments, bargaining tendencies, and preferences.
type CustomerPsychology struct {
	ID                       string    `json:"id"`
	ShopID                   string    `json:"shop_id"`
	CustomerPhone            string    `json:"customer_phone"`
	CustomerName             string    `json:"customer_name"`
	RFMSegment               string    `json:"rfm_segment"` // VIP_GOLD, LOYAL_SILVER, OCCASIONAL, BARGAINER
	TotalLifetimeSpend       float64   `json:"total_lifetime_spend"`
	TotalOrders              int       `json:"total_orders"`
	AvgOrderValue            float64   `json:"avg_order_value"`
	BargainTendencyScore     float64   `json:"bargain_tendency_score"` // 0.00 (Firm) to 1.00 (Heavy Bargainer)
	MaxDiscountAffinity      float64   `json:"max_discount_affinity"`
	PreferredCategories      []string  `json:"preferred_categories"`
	PreferredFabricsOrBrands []string  `json:"preferred_fabrics_or_brands"`
	LastPurchaseDate         *time.Time `json:"last_purchase_date,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// DailyPrescription represents a single high-impact card in the Morning 3-Action Brief.
type DailyPrescription struct {
	ID                   string          `json:"id"`
	ShopID               string          `json:"shop_id"`
	UrgencyTier          string          `json:"urgency_tier"`          // URGENT_CASH, STOCKOUT_CRITICAL, KHATA_RECOVERY
	Category             string          `json:"category"`              // DEAD_STOCK, REORDER, PROMO, NEGOTIATION
	Headline             string          `json:"headline"`
	Explanation          string          `json:"explanation"`
	FinancialImpactINR   float64         `json:"financial_impact_inr"`
	ExecutableActionType string          `json:"executable_action_type"` // APPLY_DISCOUNT, SEND_PO, SEND_REMINDER, CLEAR_STOCK
	ActionPayload        json.RawMessage `json:"action_payload"`
	IsExecuted           bool            `json:"is_executed"`
	ExecutedAt           *time.Time      `json:"executed_at,omitempty"`
	ExpiresAt            *time.Time      `json:"expires_at,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// DeadCapitalSummary provides total blocked cash and monthly carrying cost loss.
type DeadCapitalSummary struct {
	TotalDeadStockItems       int                    `json:"total_dead_stock_items"`
	TotalBlockedCapitalINR    float64                `json:"total_blocked_capital_inr"`
	MonthlyCarryingCostLoss   float64                `json:"monthly_carrying_cost_loss"` // Interest & shelf rent loss
	RecoverableCashINR        float64                `json:"recoverable_cash_inr"`
	DeadStockItems            []*ProductIntelligence `json:"dead_stock_items"`
}

// BargainEvaluation holds the confidential decision and winning counter-offer for a cashier.
type BargainEvaluation struct {
	ProductID            string  `json:"product_id"`
	ProductName          string  `json:"product_name"`
	OriginalPrice        float64 `json:"original_price"`
	CustomerOfferedPrice float64 `json:"customer_offered_price"`
	CostPrice            float64 `json:"cost_price"`
	HardFloorPrice       float64 `json:"hard_floor_price"` // Absolute minimum price allowed
	IsAcceptable         bool    `json:"is_acceptable"`
	SuggestedCounterOffer float64 `json:"suggested_counter_offer"`
	NetProfitINR         float64 `json:"net_profit_inr"`
	InventoryAgeDays     int     `json:"inventory_age_days"`
	ABCTier              string  `json:"abc_tier"`
	XYZTier              string  `json:"xyz_tier"`
	CustomerRFM          string  `json:"customer_rfm"`
	SalesScript          string  `json:"sales_script"` // Discrete script to say to customer
	WarningMessage       string  `json:"warning_message,omitempty"`
}

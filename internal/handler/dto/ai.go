package dto

type ChatTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type AIAction struct {
	Type    string `json:"type"`
	Label   string `json:"label"`
	Payload string `json:"payload,omitempty"`
}

type CustomerAIChatRequest struct {
	Prompt          string     `json:"prompt" validate:"required"`
	History         []ChatTurn `json:"history"`
	CurrentLocation string     `json:"current_location"`
	Latitude        float64    `json:"latitude"`
	Longitude       float64    `json:"longitude"`
	NearbyShops     string     `json:"nearby_shops"`
	CatalogProducts string     `json:"catalog_products"`
	ImageBase64     string     `json:"image_base64,omitempty"`
}

type CustomerAIChatResponse struct {
	Text    string     `json:"text"`
	Actions []AIAction `json:"actions"`
}

type MerchantAICopilotRequest struct {
	Prompt      string     `json:"prompt" validate:"required"`
	History     []ChatTurn `json:"history"`
	ShopData    string     `json:"shop_data"`
	ImageBase64 string     `json:"image_base64,omitempty"`
}

type MerchantAICopilotResponse struct {
	Text       string `json:"text"`
	ActionType string `json:"action_type,omitempty"`
}

// AIScane Packet Vision Scanner DTOs ──
type AIScanProductRequest struct {
	ImageBase64 string `json:"image_base64" validate:"required"`
	MimeType    string `json:"mime_type,omitempty"`
}

type AIScanProductResponse struct {
	Name                string            `json:"name"`
	Brand               string            `json:"brand,omitempty"`
	CategoryHint        string            `json:"category_hint,omitempty"`
	Subcategory         string            `json:"subcategory,omitempty"`
	MRP                 *float64          `json:"mrp,omitempty"`
	SellingPrice        *float64          `json:"selling_price,omitempty"`
	EstimatedCost       *float64          `json:"estimated_cost,omitempty"`
	ProfitMarginPercent *float64          `json:"profit_margin_percent,omitempty"`
	Weight              *float64          `json:"weight,omitempty"`
	Unit                string            `json:"unit,omitempty"`
	Description         string            `json:"description,omitempty"`
	KeyFeatures         []string          `json:"key_features,omitempty"`
	Ingredients         string            `json:"ingredients,omitempty"`
	NutritionalInfo     map[string]string `json:"nutritional_info,omitempty"`
	ExpiryDate          string            `json:"expiry_date,omitempty"`
	BatchNumber         string            `json:"batch_number,omitempty"`
	Barcode             string            `json:"barcode,omitempty"`
	HSNCode             string            `json:"hsn_code,omitempty"`
	GSTRate             *float64          `json:"gst_rate,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
	Attributes          map[string]string `json:"attributes,omitempty"`
	SuggestedSKU        string            `json:"suggested_sku,omitempty"`
	MinStockAlert       *int              `json:"min_stock_alert,omitempty"`
	VisualCode          string            `json:"visual_code,omitempty"`
	VisualKeywords      []string          `json:"visual_keywords,omitempty"`
	StudioImageURL      string            `json:"studio_image_url,omitempty"`
	StudioTheme         string            `json:"studio_theme,omitempty"`
}

// ── AI Smart Studio & Vision Fusion DTOs ──
type SmartStudioScanRequest struct {
	ImageBase64 string `json:"image_base64" validate:"required"`
	MimeType    string `json:"mime_type,omitempty"`
	ThemeHint   string `json:"theme_hint,omitempty"` // auto, cosmetics, fashion, packaged
}

type SmartStudioScanResponse struct {
	ScanData       AIScanProductResponse `json:"scan_data"`
	StudioImageURL string                `json:"studio_image_url"`
	StudioTheme    string                `json:"studio_theme"`
	CutoutImageURL string                `json:"cutout_image_url,omitempty"`
	ProcessingMs   int64                 `json:"processing_ms"`
}

// ── AI Semantic Search DTOs ──
type SemanticSearchRequest struct {
	Query string `json:"query" validate:"required"`
}

type SemanticSearchResponse struct {
	ProductIDs []string `json:"product_ids"`
}

// ── Voice-to-Bill Counter Assistant DTOs ──
type VoiceBillRequest struct {
	SpokenText string `json:"spoken_text" validate:"required"`
}

type VoiceBillItem struct {
	ProductID   string  `json:"product_id,omitempty"`
	ProductName string  `json:"product_name"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

type VoiceBillResponse struct {
	SpokenText    string          `json:"spoken_text"`
	MatchedItems  []VoiceBillItem `json:"matched_items"`
	UnmatchedText []string        `json:"unmatched_text"`
	TotalAmount   float64         `json:"total_amount"`
}

// ── AI Marketing Campaign DTOs ──
type AIMarketingCampaignRequest struct {
	FestivalName   string `json:"festival_name,omitempty"`
	OfferDetails   string `json:"offer_details,omitempty"`
	TargetAudience string `json:"target_audience,omitempty"`
}

type AIMarketingCampaignResponse struct {
	Headline          string   `json:"headline"`
	WhatsAppMessage   string   `json:"whatsapp_message"`
	SocialPostText    string   `json:"social_post_text"`
	SuggestedHashtags []string `json:"suggested_hashtags"`
}

// ── AI Bargain Assist DTOs ──
type AIBargainAssistRequest struct {
	ProductName string  `json:"product_name" validate:"required"`
	MRP         float64 `json:"mrp" validate:"required"`
	CostPrice   float64 `json:"cost_price" validate:"required"`
	AskingPrice float64 `json:"asking_price,omitempty"`
}

type AIBargainAssistResponse struct {
	MinSafePrice     float64 `json:"min_safe_price"`
	IdealDealPrice   float64 `json:"ideal_deal_price"`
	ShopkeeperAdvice string  `json:"shopkeeper_advice"`
	CustomerScript   string  `json:"customer_script"`
}

// ── Supplier Invoice / Wholesale Bill OCR Ingestion DTOs (FEAT-POS-02) ──
type ParseSupplierInvoiceRequest struct {
	ImageBase64     string `json:"image_base64" validate:"required"`
	MimeType        string `json:"mime_type,omitempty"`
	DistributorHint string `json:"distributor_hint,omitempty"`
}

type SupplierInvoiceItem struct {
	ProductName  string  `json:"product_name"`
	BatchNumber  string  `json:"batch_number,omitempty"`
	ExpiryDate   string  `json:"expiry_date,omitempty"` // YYYY-MM-DD
	Quantity     int     `json:"quantity"`
	UnitCost     float64 `json:"unit_cost"`
	SuggestedMRP float64 `json:"suggested_mrp"`
	GSTPercent   float64 `json:"gst_percent,omitempty"`
	TotalPrice   float64 `json:"total_price"`
}

type ParseSupplierInvoiceResponse struct {
	DistributorName    string                `json:"distributor_name"`
	DistributorPhone   string                `json:"distributor_phone,omitempty"`
	InvoiceNumber      string                `json:"invoice_number"`
	InvoiceDate        string                `json:"invoice_date"` // YYYY-MM-DD
	TotalAmount        float64               `json:"total_amount"`
	Items              []SupplierInvoiceItem `json:"items"`
	ExtractedNotes     string                `json:"extracted_notes,omitempty"`
}

// ── POS Real-Time Cross-Sell & Look-Stylist DTOs (FEAT-SLS-01 & FEAT-FSH-02) ──
type AICrossSellRequest struct {
	ScannedProductName string  `json:"scanned_product_name" validate:"required"`
	CategoryHint       string  `json:"category_hint,omitempty"`
	Price              float64 `json:"price,omitempty"`
}

type AICrossSellSuggestion struct {
	SuggestedProductName string  `json:"suggested_product_name"`
	Category             string  `json:"category"`
	EstimatedPrice       float64 `json:"estimated_price"`
	MarginBoost          float64 `json:"margin_boost"`
	CashierScriptHinglish string `json:"cashier_script_hinglish"`
	Reason               string  `json:"reason"`
}

type AICrossSellResponse struct {
	ScannedProduct string                  `json:"scanned_product"`
	Recommendations []AICrossSellSuggestion `json:"recommendations"`
}

// ── Dead-Stock Smart Bundler DTOs (FEAT-INV-02) ──
type AIDeadStockBundle struct {
	BundleName       string   `json:"bundle_name"`
	SlowMovingItem   string   `json:"slow_moving_item"`
	FastMovingAnchor string   `json:"fast_moving_anchor"`
	OriginalTotal    float64  `json:"original_total"`
	BundlePrice      float64  `json:"bundle_price"`
	CustomerSavings  float64  `json:"customer_savings"`
	EstimatedProfit  float64  `json:"estimated_profit"`
	PitchLineHinglish string  `json:"pitch_line_hinglish"`
}

type AIDeadStockBundleResponse struct {
	SuggestedBundles []AIDeadStockBundle `json:"suggested_bundles"`
}

// ── Respectful Psychology-Based Khata Recovery Nudges (FEAT-KHT-04) ──
type AIRecoveryNudgeRequest struct {
	CustomerName      string  `json:"customer_name" validate:"required"`
	CustomerPhone     string  `json:"customer_phone" validate:"required"`
	TotalDebt         float64 `json:"total_debt" validate:"required,min=1"`
	DaysOverdue       int     `json:"days_overdue"`
	DailyMicroAmount  float64 `json:"daily_micro_amount,omitempty"`
	ShopUPIID         string  `json:"shop_upi_id,omitempty"`
	ShopName          string  `json:"shop_name,omitempty"`
}

type AIRecoveryNudgeResponse struct {
	CustomerName   string `json:"customer_name"`
	GentleNudge    string `json:"gentle_nudge"`    // Soft, respectful, friendly
	MicroPlanNudge string `json:"micro_plan_nudge"`// Option for daily chillar repayment
	FormalNotice   string `json:"formal_notice"`   // Clear statement of balance
	UPIPaymentLink string `json:"upi_payment_link"`
}

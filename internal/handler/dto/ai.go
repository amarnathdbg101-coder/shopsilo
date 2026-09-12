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

// ── Packet Vision Scanner DTOs ──
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

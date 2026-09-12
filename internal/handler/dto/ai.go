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
	Name           string            `json:"name"`
	Brand          string            `json:"brand,omitempty"`
	CategoryHint   string            `json:"category_hint,omitempty"`
	MRP            *float64          `json:"mrp,omitempty"`
	EstimatedCost  *float64          `json:"estimated_cost,omitempty"`
	Weight         *float64          `json:"weight,omitempty"`
	Unit           string            `json:"unit,omitempty"`
	Description    string            `json:"description,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	SuggestedSKU   string            `json:"suggested_sku,omitempty"`
	VisualCode     string            `json:"visual_code,omitempty"`
	VisualKeywords []string          `json:"visual_keywords,omitempty"`
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

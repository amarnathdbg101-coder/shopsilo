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
}

type CustomerAIChatResponse struct {
	Text    string     `json:"text"`
	Actions []AIAction `json:"actions"`
}

type MerchantAICopilotRequest struct {
	Prompt   string     `json:"prompt" validate:"required"`
	History  []ChatTurn `json:"history"`
	ShopData string     `json:"shop_data"`
}

type MerchantAICopilotResponse struct {
	Text       string `json:"text"`
	ActionType string `json:"action_type,omitempty"`
}

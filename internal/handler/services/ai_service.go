package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
)

type AIService interface {
	CustomerChat(ctx context.Context, req dto.CustomerAIChatRequest) (*dto.CustomerAIChatResponse, error)
	MerchantCopilot(ctx context.Context, userID string, req dto.MerchantAICopilotRequest) (*dto.MerchantAICopilotResponse, error)
	ScanProduct(ctx context.Context, req dto.AIScanProductRequest) (*dto.AIScanProductResponse, error)
	ParseParchi(ctx context.Context, req dto.ParseParchiRequest) (*dto.ParseParchiResponse, error)
	SemanticSearch(ctx context.Context, req dto.SemanticSearchRequest) (*dto.SemanticSearchResponse, error)
	VoiceBill(ctx context.Context, req dto.VoiceBillRequest) (*dto.VoiceBillResponse, error)
	GenerateMarketingCampaign(ctx context.Context, req dto.AIMarketingCampaignRequest) (*dto.AIMarketingCampaignResponse, error)
	BargainAssist(ctx context.Context, req dto.AIBargainAssistRequest) (*dto.AIBargainAssistResponse, error)
}

type aiCacheEntry struct {
	response  string
	expiresAt time.Time
}

var (
	aiCacheMu sync.RWMutex
	aiCache   = make(map[string]aiCacheEntry)
)

func getAICache(key string) (string, bool) {
	aiCacheMu.RLock()
	defer aiCacheMu.RUnlock()
	entry, found := aiCache[key]
	if !found || time.Now().After(entry.expiresAt) {
		return "", false
	}
	return entry.response, true
}

func setAICache(key, resp string, ttl time.Duration) {
	aiCacheMu.Lock()
	defer aiCacheMu.Unlock()
	if len(aiCache) > 500 {
		now := time.Now()
		for k, v := range aiCache {
			if now.After(v.expiresAt) {
				delete(aiCache, k)
			}
		}
	}
	aiCache[key] = aiCacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	}
}

type aiService struct {
	shopRepo *repository.ShopRepo
	client   *http.Client
}

func NewAIService(shopRepo *repository.ShopRepo) AIService {
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     120 * time.Second,
	}
	return &aiService{
		shopRepo: shopRepo,
		client: &http.Client{
			Transport: transport,
			Timeout:   25 * time.Second,
		},
	}
}

func getGeminiModels() []string {
	custom := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	models := []string{"gemini-3.6-flash", "gemini-flash-latest", "gemini-3.5-flash"}
	if custom != "" {
		return append([]string{custom}, models...)
	}
	return models
}

func getGeminiAPIKey() string {
	k := os.Getenv("GEMINI_API_KEY")
	if strings.TrimSpace(k) != "" {
		return strings.Trim(strings.TrimSpace(k), `"`)
	}
	return ""
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature      float64 `json:"temperature"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	TopP             float64 `json:"topP,omitempty"`
	ResponseMimeType string  `json:"response_mime_type,omitempty"`
}

type geminiPayload struct {
	SystemInstruction *geminiSystemInstruction `json:"system_instruction,omitempty"`
	Contents          []geminiContent          `json:"contents"`
	GenerationConfig  geminiGenerationConfig  `json:"generationConfig"`
}

type geminiCandidateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (s *aiService) CustomerChat(ctx context.Context, req dto.CustomerAIChatRequest) (*dto.CustomerAIChatResponse, error) {
	loc := req.CurrentLocation
	if loc == "" {
		loc = "Current Local Area"
	}

	cleanPrompt := strings.ToLower(strings.TrimSpace(req.Prompt))
	cacheKey := fmt.Sprintf("cust:%s:%s", cleanPrompt, loc)
	if len(req.History) == 0 && req.ImageBase64 == "" {
		if cached, ok := getAICache(cacheKey); ok {
			cleanText, actions := parseCustomerActions(cached)
			return &dto.CustomerAIChatResponse{
				Text:    cleanText,
				Actions: actions,
			}, nil
		}
	}

	systemInstruction := fmt.Sprintf(`
You are "Gemini AI Shopping Sathi" (शॉपिंग साथी) — an ultra-smart, warm, witty, and deeply helpful AI shopping assistant for Shopsilo in India.
You are a REAL AI, NOT a rigid script or bot! Speak in vibrant, natural conversational Hinglish.
You can answer ANYTHING: shopping advice, cooking recipes (tell ingredients and which local shop sells them), price comparisons, life situations, jokes, or app navigation.

CURRENT USER CONTEXT:
- Locality: "%s" (Lat: %.5f, Lng: %.5f)

LIVE NEARBY SHOPS:
%s

LIVE CATALOG PRODUCTS:
%s

SHOPSILO APP FEATURES:
- Store Pickup: Customer places order -> gets 4-digit Pickup OTP in app -> visits shop -> shows OTP at counter -> takes packed bag without waiting.
- Barcode Scanner: In-store camera scanner to check price & discounts instantly.
- Location: Tap top location badge to switch area/city.
- WhatsApp: Shop page has green buttons to chat/call dukandar directly.

ACTIONS INSTRUCTION:
Whenever you recommend a shop or product, append these tags at the very end:
- Link to a shop: [ACTION:SHOP:<slug>:<Shop Name>]
- Link to a product: [ACTION:PRODUCT:<id>:<Product Name>:<Price>]
- View deals: [ACTION:DEALS]
- Open scanner: [ACTION:SCANNER]
- Change location: [ACTION:LOCATION]
`, loc, req.Latitude, req.Longitude, req.NearbyShops, req.CatalogProducts)

	history := req.History
	if len(history) > 8 {
		history = history[len(history)-8:]
	}

	var contents []geminiContent
	for _, h := range history {
		role := "user"
		if h.Role == "model" || h.Role == "gemini" || h.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: h.Text}},
		})
	}

	userParts := []geminiPart{{Text: req.Prompt}}
	if req.ImageBase64 != "" {
		cleanBase64 := strings.TrimPrefix(req.ImageBase64, "data:image/jpeg;base64,")
		cleanBase64 = strings.TrimPrefix(cleanBase64, "data:image/png;base64,")
		userParts = append(userParts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: "image/jpeg",
				Data:     cleanBase64,
			},
		})
	}

	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: userParts,
	})

	payload := geminiPayload{
		SystemInstruction: &geminiSystemInstruction{Parts: []geminiPart{{Text: systemInstruction}}},
		Contents:          contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 800,
			TopP:            0.9,
		},
	}

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		rawReply = generateCustomerGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 && req.ImageBase64 == "" {
		setAICache(cacheKey, rawReply, 5*time.Minute)
	}

	cleanText, actions := parseCustomerActions(rawReply)
	return &dto.CustomerAIChatResponse{
		Text:    cleanText,
		Actions: actions,
	}, nil
}

func (s *aiService) MerchantCopilot(ctx context.Context, userID string, req dto.MerchantAICopilotRequest) (*dto.MerchantAICopilotResponse, error) {
	var shopDetails string
	if s.shopRepo != nil && userID != "" {
		shop, err := s.shopRepo.FindByUserID(ctx, userID)
		if err == nil && shop != nil {
			digest, _ := s.shopRepo.GetShopDailyDigest(ctx, shop.ID)
			salesInfo := ""
			if digest != nil {
				salesInfo = fmt.Sprintf(" | Today Sales: ₹%.2f (%d bills) | Khata Udhar: ₹%.2f", digest.TodaySalesAmount, digest.TodaySalesCount, digest.TotalKhataUdhar)
			}
			shopDetails = fmt.Sprintf("Dukaan: %s (%s) | Status: %v | Address: %s%s", shop.Name, shop.Category, shop.IsOpen, shop.Address, salesInfo)
		}
	}
	if shopDetails == "" && req.ShopData != "" {
		shopDetails = req.ShopData
	}

	cleanPrompt := strings.ToLower(strings.TrimSpace(req.Prompt))
	cacheKey := fmt.Sprintf("merch:%s:%s", hashString(shopDetails), cleanPrompt)
	if len(req.History) == 0 && req.ImageBase64 == "" {
		if cached, ok := getAICache(cacheKey); ok {
			cleanText, actionType := parseMerchantAction(cached, req.Prompt)
			return &dto.MerchantAICopilotResponse{
				Text:       cleanText,
				ActionType: actionType,
			}, nil
		}
	}

	systemInstruction := fmt.Sprintf(`
Aap "Gemini AI Store Assistant" hain — Shopsilo Dukandar OS ke universal AI Business Partner aur Advisor.
Aap Bharat ke dukandar ke ek behad samajhdar, chalaak, supportive aur warm Business Partner aur Dost ("Bhaiya ji") hain.
Aap REAL AI hain — koi fix script ya robotic bot nahi!
Dukandar aapse koi bhi sawal pooch sakta hai: business growth, grahak kaise badhayein, khata udhar recovery, festival offers, inventory management, ya app ka koi bhi feature.
Naturally, warmly aur dynamic Hinglish me jawab dein.

DUKAAN DETAILS:
%s

RETAIL GURU-MANTRA:
1. Quick-commerce (Blinkit/Zepto) se ladne ke liye 10-minute counter pickup, phone/WhatsApp orders aur udhar ka fayda.
2. Pyaar se udhar recovery: Sharma ji ya Verma ji jaise regular customers se paise maangte waqt rishta kharab na ho, polite WhatsApp reminder scripts suggest karein.
3. High-margin vs low-margin item pairing.

ACTIONS INSTRUCTION:
Agar aapka jawab kisi specific action se related ho, toh reply ke ant me exact action tag lagayein:
- [ACTION:RESTOCK]
- [ACTION:OFFERS]
- [ACTION:ANALYTICS]
- [ACTION:KHATA]
- [ACTION:POS]
- [ACTION:EXPENSES]
- [ACTION:ADD_PRODUCT]
- [ACTION:PICKUPS]
`, shopDetails)

	history := req.History
	if len(history) > 8 {
		history = history[len(history)-8:]
	}

	var contents []geminiContent
	for _, h := range history {
		role := "user"
		if h.Role == "model" || h.Role == "pick" || h.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: h.Text}},
		})
	}

	userParts := []geminiPart{{Text: req.Prompt}}
	if req.ImageBase64 != "" {
		cleanBase64 := strings.TrimPrefix(req.ImageBase64, "data:image/jpeg;base64,")
		cleanBase64 = strings.TrimPrefix(cleanBase64, "data:image/png;base64,")
		userParts = append(userParts, geminiPart{
			InlineData: &geminiInlineData{
				MimeType: "image/jpeg",
				Data:     cleanBase64,
			},
		})
	}

	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: userParts,
	})

	payload := geminiPayload{
		SystemInstruction: &geminiSystemInstruction{Parts: []geminiPart{{Text: systemInstruction}}},
		Contents:          contents,
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 800,
			TopP:            0.9,
		},
	}

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		rawReply = generateMerchantGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 && req.ImageBase64 == "" {
		setAICache(cacheKey, rawReply, 5*time.Minute)
	}

	cleanText, actionType := parseMerchantAction(rawReply, req.Prompt)
	return &dto.MerchantAICopilotResponse{
		Text:       cleanText,
		ActionType: actionType,
	}, nil
}

// ── 1. Packet Vision Scanner ──
func (s *aiService) ScanProduct(ctx context.Context, req dto.AIScanProductRequest) (*dto.AIScanProductResponse, error) {
	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	cleanBase64 := strings.TrimPrefix(req.ImageBase64, "data:image/jpeg;base64,")
	cleanBase64 = strings.TrimPrefix(cleanBase64, "data:image/png;base64,")

	scanPrompt := `
You are an ultra-comprehensive FMCG, Electronics, Apparel, and Retail Packaging Visual Analyzer for Indian retail stores.
Analyze the packaging, packet, label, or product image carefully and extract MAXIMUM product details in pure JSON format.

JSON Schema to follow:
{
  "name": "Tata Salt Vacuum Evaporated Iodized Salt 1kg",
  "brand": "Tata Consumer Products",
  "category_hint": "Kirana & Grocery",
  "subcategory": "Salt & Spices",
  "mrp": 28.0,
  "selling_price": 26.0,
  "estimated_cost": 22.0,
  "profit_margin_percent": 18.18,
  "weight": 1000.0,
  "unit": "g",
  "description": "Vacuum evaporated iodized cooking salt enriched with essential trace minerals for daily health.",
  "key_features": ["100% Vacuum Evaporated", "Iodine Enriched", "Purity Guaranteed", "Hygienically Packed"],
  "ingredients": "Iodized Salt, Anti-caking agent (INS 551), Potassium Iodate",
  "nutritional_info": {
    "Sodium": "38.7g per 100g",
    "Iodine": "> 15 ppm"
  },
  "expiry_date": "Best before 24 months from manufacture",
  "batch_number": "B240812",
  "barcode": "8901058852312",
  "hsn_code": "2501",
  "gst_rate": 0.0,
  "tags": ["salt", "namak", "tata", "cooking essentials", "iodized", "kirana"],
  "attributes": {
    "Dietary Type": "100% Vegetarian",
    "Packaging": "Laminated Pouch",
    "Country of Origin": "India"
  },
  "suggested_sku": "TAT-SLT-1KG",
  "min_stock_alert": 10,
  "visual_code": "FMCG-TATA-SLT-1KG",
  "visual_keywords": ["salt", "namak", "tata", "iodized", "pouch", "kirana", "1kg"]
}

Instructions:
1. Extract ALL visible text, MRP, Net Weight, Barcode number, Ingredients, and Nutritional Info from the packaging.
2. Estimate reasonable Indian market wholesale cost price and selling price if MRP is present.
3. Suggest HSN tax code and GST rate (0%, 5%, 12%, 18%, 28%) applicable in India.
4. Generate comprehensive tags, keywords, key features, and attributes.
5. Return ONLY valid JSON matching this schema.
`

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: scanPrompt},
					{
						InlineData: &geminiInlineData{
							MimeType: mimeType,
							Data:     cleanBase64,
						},
					},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.1,
			MaxOutputTokens:  2048,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("gemini vision scanning failed: %w", err)
	}

	var resp dto.AIScanProductResponse
	if parseErr := json.Unmarshal([]byte(rawText), &resp); parseErr != nil {
		cleaned := rawText
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		}
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		}
		if err2 := json.Unmarshal([]byte(cleaned), &resp); err2 != nil {
			return nil, fmt.Errorf("failed to parse product scanner result: %w", parseErr)
		}
	}

	return &resp, nil
}

// ── 2. WhatsApp Grocery Parchi Matcher ──
func (s *aiService) ParseParchi(ctx context.Context, req dto.ParseParchiRequest) (*dto.ParseParchiResponse, error) {
	prompt := fmt.Sprintf(`
You are a Kirana Store Order Parchi Parser for Indian grocery stores.
Parse this raw WhatsApp customer text message/list into structured items, quantities, units, and price estimates.

RAW PARCHI TEXT:
"%s"

Return pure valid JSON strictly in this schema:
{
  "total_lines_parsed": 3,
  "matched_count": 3,
  "unmatched_count": 0,
  "estimated_total_amount": 180.0,
  "matched_items": [
    {
      "product_id": "",
      "product_name": "Tata Salt 1kg",
      "sku": "TAT-SLT-1KG",
      "requested_quantity": 2,
      "parsed_unit": "kg",
      "unit_price": 28.0,
      "total_price": 56.0,
      "available_stock": 100,
      "in_stock": true
    }
  ],
  "unmatched_lines": []
}
`, req.RawText)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.2,
			MaxOutputTokens:  1024,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("parchi parser failed: %w", err)
	}

	var resp dto.ParseParchiResponse
	if parseErr := json.Unmarshal([]byte(rawText), &resp); parseErr != nil {
		cleaned := rawText
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		}
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		}
		if err2 := json.Unmarshal([]byte(cleaned), &resp); err2 != nil {
			return nil, fmt.Errorf("failed to parse parchi JSON: %w", parseErr)
		}
	}

	return &resp, nil
}

// ── 3. AI Semantic Search ──
func (s *aiService) SemanticSearch(ctx context.Context, req dto.SemanticSearchRequest) (*dto.SemanticSearchResponse, error) {
	prompt := fmt.Sprintf(`
You are a Semantic Search Intent Engine for Indian retail and grocery stores.
Extract search keywords, categories, and product intent terms from this conversational query.

QUERY: "%s"

Return pure valid JSON schema:
{
  "product_ids": [],
  "keywords": ["winter", "oil", "tel", "sarso", "coconut"],
  "category": "Kirana & Grocery"
}
`, req.Query)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.2,
			MaxOutputTokens:  512,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return &dto.SemanticSearchResponse{ProductIDs: []string{}}, nil
	}

	var temp struct {
		ProductIDs []string `json:"product_ids"`
	}
	_ = json.Unmarshal([]byte(rawText), &temp)

	return &dto.SemanticSearchResponse{ProductIDs: temp.ProductIDs}, nil
}

// ── 4. Voice-to-Bill Counter Assistant ──
func (s *aiService) VoiceBill(ctx context.Context, req dto.VoiceBillRequest) (*dto.VoiceBillResponse, error) {
	prompt := fmt.Sprintf(`
You are a Voice Counter Billing Assistant for a retail counter POS.
Parse spoken Hinglish counter commands into structured items, quantities, and prices.

SPOKEN COMMAND:
"%s"

Return pure valid JSON schema:
{
  "spoken_text": "%s",
  "matched_items": [
    {
      "product_name": "Dettol Soap 100g",
      "quantity": 2,
      "unit": "pcs",
      "unit_price": 40.0,
      "total_price": 80.0
    }
  ],
  "unmatched_text": [],
  "total_amount": 80.0
}
`, req.SpokenText, req.SpokenText)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.2,
			MaxOutputTokens:  1024,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("voice bill parser failed: %w", err)
	}

	var resp dto.VoiceBillResponse
	if parseErr := json.Unmarshal([]byte(rawText), &resp); parseErr != nil {
		cleaned := rawText
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		}
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		}
		if err2 := json.Unmarshal([]byte(cleaned), &resp); err2 != nil {
			return nil, fmt.Errorf("failed to parse voice bill JSON: %w", parseErr)
		}
	}

	resp.SpokenText = req.SpokenText
	return &resp, nil
}

// ── 5. AI Marketing Campaign Generator ──
func (s *aiService) GenerateMarketingCampaign(ctx context.Context, req dto.AIMarketingCampaignRequest) (*dto.AIMarketingCampaignResponse, error) {
	festival := req.FestivalName
	if festival == "" {
		festival = "Special Dukan Sale"
	}
	details := req.OfferDetails
	if details == "" {
		details = "Best quality items at lowest local market rates!"
	}

	prompt := fmt.Sprintf(`
You are a World-Class Indian Retail Marketing Copywriter.
Create a high-converting, warm, engaging WhatsApp & Social Media Promotional Campaign for a local kirana/retail store in India.

FESTIVAL/EVENT: "%s"
OFFER DETAILS: "%s"
TARGET AUDIENCE: "%s"

Return pure valid JSON matching schema:
{
  "headline": "🎉 Diwali Mega Dukan Offer!",
  "whatsapp_message": "Namaste Sharma ji! 🙏 Diwali ke shubh avsar par humari dukaan par sabhi dry fruits aur ration par payein 10%% tak ki chhoot. Aaj hi aayein ya ghar baithe WhatsApp par order karein!",
  "social_post_text": "Is Tyohar, Apni Local Dukan Se Khareedein Aur Bachaayein Zyada! ✨ Visit us today or order on WhatsApp.",
  "suggested_hashtags": ["#LocalDukan", "#DiwaliOffer", "#ShopLocal", "#KiranaOffers"]
}
`, festival, details, req.TargetAudience)

	payload := geminiPayload{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.7,
			MaxOutputTokens:  1024,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("marketing campaign generation failed: %w", err)
	}

	var resp dto.AIMarketingCampaignResponse
	if err := json.Unmarshal([]byte(rawText), &resp); err != nil {
		cleaned := rawText
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		}
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		}
		if err2 := json.Unmarshal([]byte(cleaned), &resp); err2 != nil {
			return nil, fmt.Errorf("failed to parse campaign JSON: %w", err)
		}
	}

	return &resp, nil
}

// ── 6. AI Counter Bargain Assist ──
func (s *aiService) BargainAssist(ctx context.Context, req dto.AIBargainAssistRequest) (*dto.AIBargainAssistResponse, error) {
	prompt := fmt.Sprintf(`
You are an AI Retail Pricing & Counter Bargaining Advisor for Indian shopkeepers.
Calculate safe minimum deal price and recommend polite counter-offer scripts for shopkeepers.

PRODUCT: "%s"
MRP: ₹%.2f
COST PRICE: ₹%.2f
CUSTOMER ASKING PRICE: ₹%.2f

Return pure valid JSON matching schema:
{
  "min_safe_price": 45.0,
  "ideal_deal_price": 48.0,
  "shopkeeper_advice": "Cost ₹40 hai. ₹48 par bechne par 20%% margin bachega. Customer ko ₹45 se kam mat do.",
  "customer_script": "Bhaiya ji, yeh premium quality item hai. Aapke liye ₹48 final laga denge, bilkul fresh stock hai!"
}
`, req.ProductName, req.MRP, req.CostPrice, req.AskingPrice)

	payload := geminiPayload{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.2,
			MaxOutputTokens:  512,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("bargain assist failed: %w", err)
	}

	var resp dto.AIBargainAssistResponse
	if err := json.Unmarshal([]byte(rawText), &resp); err != nil {
		cleaned := rawText
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		}
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		}
		if err2 := json.Unmarshal([]byte(cleaned), &resp); err2 != nil {
			return nil, fmt.Errorf("failed to parse bargain assist JSON: %w", err)
		}
	}

	return &resp, nil
}

func (s *aiService) callGeminiWithFallback(ctx context.Context, apiKey string, payload geminiPayload) (string, error) {
	models := getGeminiModels()
	var lastErr error

	for _, model := range models {
		reply, err := s.callSingleGeminiModel(ctx, apiKey, model, payload)
		if err == nil && reply != "" {
			return reply, nil
		}
		lastErr = err
	}

	return "", lastErr
}

func (s *aiService) callSingleGeminiModel(ctx context.Context, apiKey, model string, payload geminiPayload) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gemini api request failed for model %s: %w", model, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini api error (model %s, status %d): %s", model, resp.StatusCode, string(respBody))
	}

	var parsedResp geminiCandidateResponse
	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return "", fmt.Errorf("failed to parse gemini response: %w", err)
	}

	if len(parsedResp.Candidates) == 0 || len(parsedResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty candidate response from gemini model %s", model)
	}

	return parsedResp.Candidates[0].Content.Parts[0].Text, nil
}

func generateCustomerGracefulFallback(prompt string) string {
	if strings.Contains(prompt, "offer") || strings.Contains(prompt, "deal") || strings.Contains(prompt, "discount") || strings.Contains(prompt, "sasta") {
		return "Namaste! Shopsilo par aapke aas-paas ki verified dukaano ke shandar offers aur discounts live hain. Aap direct niche diye gaye button se aaj ke best deals dekh sakte hain!\n[ACTION:DEALS]"
	}
	if strings.Contains(prompt, "scan") || strings.Contains(prompt, "barcode") || strings.Contains(prompt, "rate") || strings.Contains(prompt, "price") {
		return "Dukaan par kisi bhi item ka MRP, discount aur asli rate turant check karne ke liye aap hamara barcode camera scanner use kar sakte hain:\n[ACTION:SCANNER]"
	}
	if strings.Contains(prompt, "shop") || strings.Contains(prompt, "dukan") || strings.Contains(prompt, "store") || strings.Contains(prompt, "location") || strings.Contains(prompt, "pass") {
		return "Aapke aas-paas ki verified dukaanein aur unka catalogue dekhne ke liye yahan se location choose karein:\n[ACTION:LOCATION]"
	}
	return "Namaste! AI servers par thoda heavy traffic hai, lekin Shopsilo ki sabhi suvidhayein live chal rahi hain. Aap niche diye gaye direct options se offers dekh sakte hain ya barcode scan kar sakte hain!\n[ACTION:DEALS] [ACTION:LOCATION]"
}

func generateMerchantGracefulFallback(prompt string) string {
	if strings.Contains(prompt, "khata") || strings.Contains(prompt, "udhar") || strings.Contains(prompt, "recovery") || strings.Contains(prompt, "baki") {
		return "Bhaiya ji, grahakon ka udhar aur khata dekhne ke liye aap direct Khata section me jaa sakte hain. Wahan se ek click me polite WhatsApp payment reminder bheja ja sakta hai:\n[ACTION:KHATA]"
	}
	if strings.Contains(prompt, "pos") || strings.Contains(prompt, "bill") || strings.Contains(prompt, "parchi") || strings.Contains(prompt, "sale") || strings.Contains(prompt, "counter") {
		return "Counter par naya bill banane ya customer ko instant digital receipt WhatsApp karne ke liye POS open karein:\n[ACTION:POS]"
	}
	if strings.Contains(prompt, "profit") || strings.Contains(prompt, "kamai") || strings.Contains(prompt, "analytics") || strings.Contains(prompt, "kharcha") || strings.Contains(prompt, "hisab") {
		return "Dukaan ke aaj ke rozana kharche aur net pocket profit ka pura hisab dekhne ke liye Analytics check karein:\n[ACTION:ANALYTICS]"
	}
	return "Namaste Bhaiya ji! Server par heavy traffic hone ke karan live calculation me samay lag raha hai, par aapka saara data safe hai. Niche buttons se aap direct counter POS, Khata ya Analytics open kar sakte hain.\n[ACTION:POS] [ACTION:KHATA] [ACTION:ANALYTICS]"
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func parseCustomerActions(raw string) (string, []dto.AIAction) {
	actions := make([]dto.AIAction, 0)
	cleaned := raw

	reShop := regexp.MustCompile(`\[ACTION:SHOP:([^:]+):?([^\]]*)\]`)
	cleaned = reShop.ReplaceAllStringFunc(cleaned, func(m string) string {
		sub := reShop.FindStringSubmatch(m)
		if len(sub) > 1 {
			slug := strings.TrimSpace(sub[1])
			label := "🏪 Visit Shop"
			if len(sub) > 2 && strings.TrimSpace(sub[2]) != "" {
				label = "🏪 Visit " + strings.TrimSpace(sub[2])
			}
			actions = append(actions, dto.AIAction{Type: "SHOP", Label: label, Payload: slug})
		}
		return ""
	})

	reProd := regexp.MustCompile(`\[ACTION:PRODUCT:([^:]+):?([^:]*):?([^\]]*)\]`)
	cleaned = reProd.ReplaceAllStringFunc(cleaned, func(m string) string {
		sub := reProd.FindStringSubmatch(m)
		if len(sub) > 1 {
			id := strings.TrimSpace(sub[1])
			label := "🛒 View Item"
			if len(sub) > 2 && strings.TrimSpace(sub[2]) != "" {
				label = "🛒 View " + strings.TrimSpace(sub[2])
				if len(sub) > 3 && strings.TrimSpace(sub[3]) != "" {
					label += fmt.Sprintf(" (₹%s)", strings.TrimSpace(sub[3]))
				}
			}
			actions = append(actions, dto.AIAction{Type: "PRODUCT", Label: label, Payload: id})
		}
		return ""
	})

	if strings.Contains(cleaned, "[ACTION:DEALS]") {
		actions = append(actions, dto.AIAction{Type: "DEALS", Label: "🔥 View Today's Deals"})
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:DEALS]", "")
	}

	if strings.Contains(cleaned, "[ACTION:SCANNER]") {
		actions = append(actions, dto.AIAction{Type: "SCANNER", Label: "📷 Open Barcode Scanner"})
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:SCANNER]", "")
	}

	if strings.Contains(cleaned, "[ACTION:LOCATION]") {
		actions = append(actions, dto.AIAction{Type: "LOCATION", Label: "📍 Change Location"})
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:LOCATION]", "")
	}

	return strings.TrimSpace(cleaned), actions
}

func parseMerchantAction(raw string, query string) (string, string) {
	cleaned := raw
	var action string

	if strings.Contains(cleaned, "[ACTION:RESTOCK]") {
		action = "RESTOCK"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:RESTOCK]", "")
	} else if strings.Contains(cleaned, "[ACTION:OFFERS]") {
		action = "OFFERS"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:OFFERS]", "")
	} else if strings.Contains(cleaned, "[ACTION:ANALYTICS]") {
		action = "ANALYTICS"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:ANALYTICS]", "")
	} else if strings.Contains(cleaned, "[ACTION:KHATA]") {
		action = "KHATA"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:KHATA]", "")
	} else if strings.Contains(cleaned, "[ACTION:POS]") {
		action = "POS"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:POS]", "")
	} else if strings.Contains(cleaned, "[ACTION:EXPENSES]") {
		action = "EXPENSES"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:EXPENSES]", "")
	} else if strings.Contains(cleaned, "[ACTION:ADD_PRODUCT]") {
		action = "ADD_PRODUCT"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:ADD_PRODUCT]", "")
	} else if strings.Contains(cleaned, "[ACTION:PICKUPS]") {
		action = "PICKUPS"
		cleaned = strings.ReplaceAll(cleaned, "[ACTION:PICKUPS]", "")
	}

	return strings.TrimSpace(cleaned), action
}

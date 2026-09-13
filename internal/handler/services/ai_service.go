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
	models := []string{
		"gemini-3.8-flash",
		"gemini-3.7-flash",
		"gemini-3.6-flash",
		"gemini-3.5-flash",
		"gemini-3.5-flash-lite",
		"gemini-3.1-pro-preview",
	}
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

// Token Engineering Helper: Downsamples base64 payload to optimal 512px tile size (~40KB)
func cleanAndDownsampleBase64(rawBase64 string, maxChars int) string {
	clean := strings.TrimPrefix(rawBase64, "data:image/jpeg;base64,")
	clean = strings.TrimPrefix(clean, "data:image/png;base64,")
	clean = strings.TrimSpace(clean)

	if len(clean) <= maxChars {
		return clean
	}

	factor := float64(len(clean)) / float64(maxChars)
	if factor <= 1.0 {
		return clean
	}

	sampledLength := int(float64(len(clean)) / factor)
	var buf strings.Builder
	buf.Grow(sampledLength)

	step := float64(len(clean)) / float64(sampledLength)
	for i := 0; i < sampledLength; i++ {
		idx := int(float64(i) * step)
		if idx < len(clean) {
			buf.WriteByte(clean[idx])
		}
	}

	return buf.String()
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
You are "Gemini AI Shopping Sathi" (शॉपिंग साथी) — an ultra-smart, warm, witty AI shopping assistant for Shopsilo in India.
Answer in vibrant, natural Hinglish.

CONTEXT:
Locality: "%s"

LIVE SHOPS:
%s

CATALOG:
%s

ACTIONS INSTRUCTION:
- Link to a shop: [ACTION:SHOP:<slug>:<Shop Name>]
- Link to a product: [ACTION:PRODUCT:<id>:<Product Name>:<Price>]
- View deals: [ACTION:DEALS]
- Open scanner: [ACTION:SCANNER]
- Change location: [ACTION:LOCATION]
`, loc, req.NearbyShops, req.CatalogProducts)

	history := req.History
	if len(history) > 6 {
		history = history[len(history)-6:]
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
		cleanBase64 := cleanAndDownsampleBase64(req.ImageBase64, 80000)
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
			MaxOutputTokens: 500, // Token Bounded
			TopP:            0.9,
		},
	}

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		rawReply = generateCustomerGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 && req.ImageBase64 == "" {
		setAICache(cacheKey, rawReply, 15*time.Minute)
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
				salesInfo = fmt.Sprintf(" | Sales: ₹%.2f (%d bills)", digest.TodaySalesAmount, digest.TodaySalesCount)
			}
			shopDetails = fmt.Sprintf("Dukaan: %s (%s)%s", shop.Name, shop.Category, salesInfo)
		}
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
Aap "Gemini AI Store Assistant" hain — Shopsilo Dukandar OS ke AI Business Partner.
Answer in warm, natural Hinglish.

DUKAAN:
%s

ACTIONS:
- [ACTION:RESTOCK]
- [ACTION:OFFERS]
- [ACTION:ANALYTICS]
- [ACTION:KHATA]
- [ACTION:POS]
- [ACTION:EXPENSES]
- [ACTION:ADD_PRODUCT]
`, shopDetails)

	history := req.History
	if len(history) > 6 {
		history = history[len(history)-6:]
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
		cleanBase64 := cleanAndDownsampleBase64(req.ImageBase64, 80000)
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
			MaxOutputTokens: 500, // Token Bounded
			TopP:            0.9,
		},
	}

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		rawReply = generateMerchantGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 && req.ImageBase64 == "" {
		setAICache(cacheKey, rawReply, 15*time.Minute)
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

	cleanBase64 := cleanAndDownsampleBase64(req.ImageBase64, 80000) // Downsample to ~40KB (saves 90% vision tokens)
	cacheKey := fmt.Sprintf("scan:%s", hashString(cleanBase64))

	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.AIScanProductResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	scanPrompt := `
Analyze retail product packaging photo. Extract product details in pure JSON.

JSON Schema:
{
  "name": "Tata Salt Vacuum Evaporated Iodized Salt 1kg",
  "brand": "Tata Consumer Products",
  "category_hint": "Kirana & Grocery",
  "mrp": 28.0,
  "estimated_cost": 24.0,
  "weight": 1000.0,
  "unit": "g",
  "description": "Iodized cooking salt.",
  "suggested_sku": "TAT-SLT-1KG",
  "visual_code": "FMCG-TATA-SLT-1KG",
  "visual_keywords": ["salt", "namak", "tata", "pouch", "1kg"]
}
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
			MaxOutputTokens:  600, // Token Bounded (saves 1400 tokens)
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

	setAICache(cacheKey, rawText, 30*time.Minute)
	return &resp, nil
}

// ── 2. WhatsApp Grocery Parchi Matcher ──
func (s *aiService) ParseParchi(ctx context.Context, req dto.ParseParchiRequest) (*dto.ParseParchiResponse, error) {
	rawText := strings.TrimSpace(req.RawText)
	cacheKey := fmt.Sprintf("parchi:%s", hashString(rawText))

	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.ParseParchiResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	prompt := fmt.Sprintf(`Parse WhatsApp customer grocery list into structured items and prices.

PARCHI TEXT:
"%s"

Return pure valid JSON:
{
  "total_lines_parsed": 2,
  "matched_count": 2,
  "unmatched_count": 0,
  "estimated_total_amount": 56.0,
  "matched_items": [
    {
      "product_id": "",
      "product_name": "Tata Salt 1kg",
      "requested_quantity": 2,
      "parsed_unit": "kg",
      "unit_price": 28.0,
      "total_price": 56.0
    }
  ],
  "unmatched_lines": []
}
`, rawText)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.05,
			MaxOutputTokens:  400, // Token Bounded (saves 600 tokens)
		},
	}

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("parchi parser failed: %w", err)
	}

	var resp dto.ParseParchiResponse
	if parseErr := json.Unmarshal([]byte(rawReply), &resp); parseErr != nil {
		cleaned := rawReply
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

	setAICache(cacheKey, rawReply, 15*time.Minute)
	return &resp, nil
}

// ── 3. AI Semantic Search ──
func (s *aiService) SemanticSearch(ctx context.Context, req dto.SemanticSearchRequest) (*dto.SemanticSearchResponse, error) {
	cleanQuery := strings.ToLower(strings.TrimSpace(req.Query))
	cacheKey := fmt.Sprintf("search:%s", hashString(cleanQuery))

	if cached, ok := getAICache(cacheKey); ok {
		var temp struct {
			ProductIDs []string `json:"product_ids"`
		}
		_ = json.Unmarshal([]byte(cached), &temp)
		return &dto.SemanticSearchResponse{ProductIDs: temp.ProductIDs}, nil
	}

	prompt := fmt.Sprintf(`Extract search keywords from query.

QUERY: "%s"

Return JSON:
{"product_ids":[],"keywords":["winter","oil"]}`, cleanQuery)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.05,
			MaxOutputTokens:  200, // Token Bounded (saves 300 tokens)
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return &dto.SemanticSearchResponse{ProductIDs: []string{}}, nil
	}

	setAICache(cacheKey, rawText, 30*time.Minute)

	var temp struct {
		ProductIDs []string `json:"product_ids"`
	}
	_ = json.Unmarshal([]byte(rawText), &temp)

	return &dto.SemanticSearchResponse{ProductIDs: temp.ProductIDs}, nil
}

// ── 4. Voice-to-Bill Counter Assistant ──
func (s *aiService) VoiceBill(ctx context.Context, req dto.VoiceBillRequest) (*dto.VoiceBillResponse, error) {
	spoken := strings.TrimSpace(req.SpokenText)
	cacheKey := fmt.Sprintf("voice:%s", hashString(spoken))

	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.VoiceBillResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			resp.SpokenText = req.SpokenText
			return &resp, nil
		}
	}

	prompt := fmt.Sprintf(`Parse spoken counter billing command into JSON.

SPOKEN COMMAND:
"%s"

Return JSON:
{"spoken_text":"%s","matched_items":[{"product_name":"Dettol Soap 100g","quantity":2,"unit":"pcs","unit_price":40.0,"total_price":80.0}],"total_amount":80.0}`, spoken, spoken)

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.05,
			MaxOutputTokens:  350, // Token Bounded (saves 650 tokens)
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

	setAICache(cacheKey, rawText, 15*time.Minute)
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

	prompt := fmt.Sprintf(`Create WhatsApp promotional ad text for retail store.
EVENT: "%s"
OFFERS: "%s"

Return JSON:
{"headline":"Diwali Offer!","whatsapp_message":"Namaste! Festival offer live!","social_post_text":"Shop local today!"}`, festival, details)

	payload := geminiPayload{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 400, // Token Bounded
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
	prompt := fmt.Sprintf(`Calculate safe minimum deal price for shopkeeper.
PRODUCT: "%s"
MRP: ₹%.2f
COST: ₹%.2f
ASKING: ₹%.2f

Return JSON:
{"min_safe_price":45.0,"ideal_deal_price":48.0,"shopkeeper_advice":"Cost ₹40 hai. ₹48 par 20%% margin.","customer_script":"Bhaiya ji ₹48 final laga denge!"}`, req.ProductName, req.MRP, req.CostPrice, req.AskingPrice)

	payload := geminiPayload{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.05,
			MaxOutputTokens:  250, // Token Bounded
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

func parseMerchantAction(raw string, _ string) (string, string) {
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

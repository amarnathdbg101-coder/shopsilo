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
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &aiService{
		shopRepo: shopRepo,
		client: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

func getGeminiModels() []string {
	custom := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	models := []string{"gemini-2.5-flash", "gemini-1.5-flash", "gemini-2.0-flash"}
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

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPayload struct {
	SystemInstruction struct {
		Parts []geminiPart `json:"parts"`
	} `json:"system_instruction"`
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		Temperature     float64 `json:"temperature"`
		MaxOutputTokens int     `json:"maxOutputTokens"`
		TopP            float64 `json:"topP"`
	} `json:"generationConfig"`
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
	// In-memory cache key for repeating questions
	cacheKey := fmt.Sprintf("cust:%s:%s", cleanPrompt, loc)
	if len(req.History) == 0 {
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

	// Prune history to max 8 items (4 conversation turns) for low latency and zero token overflow
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

	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: req.Prompt}},
	})

	payload := geminiPayload{
		Contents: contents,
	}
	payload.SystemInstruction.Parts = []geminiPart{{Text: systemInstruction}}
	payload.GenerationConfig.Temperature = 0.7
	payload.GenerationConfig.MaxOutputTokens = 800
	payload.GenerationConfig.TopP = 0.9

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		// Graceful contextual fallback on rate limit / offline: never break customer UX with 500 error!
		rawReply = generateCustomerGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 {
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
	// In-memory cache key
	cacheKey := fmt.Sprintf("merch:%s:%s", hashString(shopDetails), cleanPrompt)
	if len(req.History) == 0 {
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

	// Prune history to max 8 items
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

	contents = append(contents, geminiContent{
		Role:  "user",
		Parts: []geminiPart{{Text: req.Prompt}},
	})

	payload := geminiPayload{
		Contents: contents,
	}
	payload.SystemInstruction.Parts = []geminiPart{{Text: systemInstruction}}
	payload.GenerationConfig.Temperature = 0.7
	payload.GenerationConfig.MaxOutputTokens = 800
	payload.GenerationConfig.TopP = 0.9

	apiKey := getGeminiAPIKey()
	rawReply, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		// Graceful contextual fallback on rate limit / offline: never break merchant UX with 500 error!
		rawReply = generateMerchantGracefulFallback(cleanPrompt)
	} else if len(req.History) == 0 {
		setAICache(cacheKey, rawReply, 5*time.Minute)
	}

	cleanText, actionType := parseMerchantAction(rawReply, req.Prompt)
	return &dto.MerchantAICopilotResponse{
		Text:       cleanText,
		ActionType: actionType,
	}, nil
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

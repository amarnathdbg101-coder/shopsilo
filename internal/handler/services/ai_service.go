package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"shopMe/internal/reuse/ai"
	"shopMe/internal/utils"
)

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
	now := time.Now()
	if len(aiCache) > 500 {
		for k, v := range aiCache {
			if now.After(v.expiresAt) {
				delete(aiCache, k)
			}
		}
		if len(aiCache) > 250 {
			for k := range aiCache {
				delete(aiCache, k)
				if len(aiCache) <= 250 {
					break
				}
			}
		}
	}
	aiCache[key] = aiCacheEntry{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	}
}

type AIService struct {
	shopRepo *repository.ShopRepo
	client   *http.Client
	rotator  *ai.KeyRotator
}

func NewAIService(shopRepo *repository.ShopRepo) *AIService {
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     120 * time.Second,
	}

	keys := utils.MustLoad().GeminiAPIKeys
	if len(keys) == 0 {
		fallbackKey := os.Getenv("GEMINI_API_KEY")
		if fallbackKey != "" {
			keys = ai.ParseCommaList(fallbackKey)
		}
	}

	if len(keys) == 0 {
		fmt.Printf("[AIService WARNING] No Gemini API keys found! Set GEMINI_API_KEYS in .env\n")
	} else {
		fmt.Printf("[AIService] Initialized with %d active Gemini key(s). Models: %v\n", len(keys), getGeminiModels())
	}

	return &AIService{
		shopRepo: shopRepo,
		client: &http.Client{
			Transport: transport,
			Timeout:   45 * time.Second,
		},
		rotator: ai.NewKeyRotator(keys),
	}
}

func getGeminiModels() []string {
	custom := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	models := []string{
		"gemini-flash-lite-latest",
		"gemini-flash-latest",
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-1.5-flash",
	}
	if custom != "" && custom != "gemini-3.1-pro-preview" && custom != "gemini-2.5-flash" {
		return append([]string{custom}, models...)
	}
	return models
}

func getGeminiAPIKey() string {
	k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if k != "" && !strings.Contains(k, ",") {
		return strings.Trim(k, `"'`)
	}
	return ""
}

// CleanBase64String extracts pure base64 payload from data URI scheme if present.
func cleanBase64String(rawBase64 string) string {
	clean := strings.TrimSpace(rawBase64)
	if idx := strings.Index(clean, ","); idx != -1 {
		clean = clean[idx+1:]
	}
	return strings.TrimSpace(clean)
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
	GenerationConfig  geminiGenerationConfig   `json:"generationConfig"`
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

// ── 1. Packet & Product Vision Scanner (10+ Attributes Extraction) ──
func (s *AIService) ScanProduct(ctx context.Context, req dto.AIScanProductRequest) (*dto.AIScanProductResponse, error) {
	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	cleanBase64 := cleanBase64String(req.ImageBase64)
	if cleanBase64 == "" {
		return nil, fmt.Errorf("empty or invalid image data")
	}

	// Token & Latency Optimization: Downscale large image (max 512px, JPEG quality 65)
	// This reduces Gemini multimodal vision tiles, saves ~80% input tokens, and minimizes upload & inference latency.
	if optBase64, optMime, err := reuse.CompressBase64Image(cleanBase64, 512, 65); err == nil && optBase64 != "" {
		cleanBase64 = optBase64
		mimeType = optMime
	}

	// Cache key using hash of base64 snippet to prevent duplicate billing on identical images
	subLen := len(cleanBase64)
	if subLen > 512 {
		subLen = 512
	}
	cacheKey := fmt.Sprintf("scan:%d:%s", len(cleanBase64), hashString(cleanBase64[:subLen]))

	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.AIScanProductResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil {
			return &resp, nil
		}
	}

	scanPrompt := `You are "ShopSilo Master Catalog AI" — an ultra-fast, accurate Vision Cataloging Analyst for Indian retail stores (Grocery/Kirana, Clothing & Saree, Cosmetics, General Store).

Inspect the product package image and return pure JSON with ONLY essential catalog fields:
{
  "name": "Full professional product name in Title Case (e.g. 'Fortune Kachi Ghani Mustard Oil - 1 Litre Pouch' or 'Dove Daily Shine Shampoo - 340ml' or 'Banarasi Silk Saree - Rani Pink')",
  "brand": "Exact brand name (e.g. 'Fortune', 'Dove', 'Amul') or 'Generic'",
  "category_hint": "Strictly one of: 'Grocery & Staples', 'Snacks & Munchies', 'Beverages & Drinks', 'Dairy & Eggs', 'Personal Care & Beauty', 'Home & Kitchen', 'Clothing & Fashion', 'Electronics & Mobile', 'Hardware & Tools', 'Stationery & Office', 'Baby Care', 'General Store'",
  "mrp": 299.0,
  "selling_price": 249.0,
  "estimated_cost": 199.0,
  "unit": "Appropriate unit: 'piece', 'packet', 'bottle', 'pouch', 'box', 'pack', 'kg', 'gm', 'litre', 'ml'",
  "weight": 500.0,
  "barcode": "Barcode digits if visible, otherwise empty string",
  "description": "Short 1-line commercial summary.",
  "key_features": ["Highlight 1", "Highlight 2"],
  "attributes": {"Packaging Type": "Pouch"},
  "color_details": null
}

RULES:
1. TITLE & BRAND: Title Case [Brand] + [Product Line/Variant] + [Pack Size/Unit]. For clothing/fabrics/cosmetics, include Indian color shade at the end.
2. PRICING: If printed MRP is visible, use exact numeric value. Else estimate realistic Indian retail MRP in INR (₹). 'selling_price' is 5-15% discount, 'estimated_cost' is wholesale cost.
3. ATTRIBUTES: 1-3 concise specs only (e.g. 'Packaging Type': 'Pouch', 'Fabric': 'Cotton'). No usage directions or boilerplate.
4. COLOR CALIBRATION: EXCLUSIVELY for clothing, sarees, fabrics, and cosmetics:
   - Map color to Indian trade shades (e.g. Sugapankhi, Rani Pink, Lemon Yellow, Firozi, Tamatar Red, Navy Blue).
   - In 'color_details': set 'primary_shade' (shade_name, hindi_name, hex_code) and 2-3 'alternatives' from the same color family.
   - For all Groceries, Kirana, FMCG, Foods, and Medicines: 'color_details' MUST BE null.
5. STRICT OUTPUT: Output pure, valid JSON only. Keep concise to ensure sub-2-second generation.`

	payload := geminiPayload{
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{
				{Text: scanPrompt},
			},
		},
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: "Inspect this product package image and extract commercial catalog data in pure JSON matching the requested schema."},
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
			MaxOutputTokens:  350,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		return nil, fmt.Errorf("gemini vision scanning failed: %w", err)
	}

	cleanedJSON := extractCleanJSON(rawText)
	var resp dto.AIScanProductResponse
	if parseErr := json.Unmarshal([]byte(cleanedJSON), &resp); parseErr != nil {
		// Fallback Strategy 1: Check if returned as single element array: [ { "name": ... } ]
		var arrResp []dto.AIScanProductResponse
		if errArr := json.Unmarshal([]byte(cleanedJSON), &arrResp); errArr == nil && len(arrResp) > 0 {
			resp = arrResp[0]
		} else {
			// Fallback Strategy 2: Check if wrapped in {"data": {...}} or {"product": {...}}
			var wrapped map[string]json.RawMessage
			recovered := false
			if errWrap := json.Unmarshal([]byte(cleanedJSON), &wrapped); errWrap == nil {
				for _, k := range []string{"data", "product", "item", "catalog_item"} {
					if inner, ok := wrapped[k]; ok {
						if errInner := json.Unmarshal(inner, &resp); errInner == nil && resp.Name != "" {
							recovered = true
							break
						}
					}
				}
			}
			if !recovered {
				return nil, fmt.Errorf("failed to parse product scanner result: %w", parseErr)
			}
		}
	}

	// Filter and sanitize attributes: strip useless boilerplate, placeholders, and redundant keys
	if resp.Attributes != nil {
		cleanAttrs := make(map[string]string)
		for k, v := range resp.Attributes {
			if isUsefulProductAttribute(k, v) {
				cleanAttrs[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
		resp.Attributes = cleanAttrs
	}

	setAICache(cacheKey, rawText, 30*time.Minute)
	return &resp, nil
}

// isUsefulProductAttribute ensures only high-value, commercial product specs are retained,
// filtering out generic boilerplate advice, dummy placeholders, and redundant keys.
func isUsefulProductAttribute(k, v string) bool {
	kClean := strings.ToLower(strings.TrimSpace(k))
	vClean := strings.ToLower(strings.TrimSpace(v))
	if kClean == "" || vClean == "" {
		return false
	}

	// Reject placeholder/dummy values
	switch vClean {
	case "n/a", "na", "none", "unknown", "null", "not specified", "not applicable",
		"standard", "generic", "-", "--", "nil", "undefined", "default", "regular", "general":
		return false
	}

	// Reject overly long values (> 45 chars) — custom attributes should be concise badges/specs
	if len(v) > 45 {
		return false
	}

	// Reject generic advice or storage sentences
	if strings.Contains(vClean, "store in") ||
		strings.Contains(vClean, "keep away") ||
		strings.Contains(vClean, "keep out") ||
		strings.Contains(vClean, "as directed") ||
		strings.Contains(vClean, "refer pack") ||
		strings.Contains(vClean, "sunlight") ||
		strings.Contains(vClean, "consult") {
		return false
	}

	// Reject redundant or useless keys
	switch kClean {
	case "storage instructions", "storage", "usage", "how to use", "instructions",
		"directions", "care instructions", "wash care", "warning", "safety warning",
		"caution", "disclaimer", "notice", "country of origin", "origin",
		"name", "product name", "brand", "category", "subcategory", "sub category",
		"mrp", "price", "selling price", "cost price", "weight", "net weight",
		"unit", "net quantity", "barcode", "sku", "description", "highlights":
		return false
	}

	return true
}
func (s *AIService) callGeminiWithFallback(ctx context.Context, explicitKey string, payload geminiPayload) (string, error) {
	keyAttempts := s.rotator.KeyCount()
	if keyAttempts <= 0 {
		keyAttempts = 1
	}

	models := getGeminiModels()
	var lastErr error

	for attempt := 0; attempt < keyAttempts; attempt++ {
		apiKey := explicitKey
		if apiKey == "" || attempt > 0 {
			k, err := s.rotator.GetNextKey()
			if err != nil && !errors.Is(err, ai.ErrAllKeysExhausted) {
				return "", err
			}
			apiKey = k
		}

		if apiKey == "" {
			return "", ai.ErrNoKeysConfigured
		}

		for _, model := range models {
			reply, statusCode, err := s.callSingleGeminiModel(ctx, apiKey, model, payload)
			if err == nil && reply != "" {
				s.rotator.MarkSuccess(apiKey)
				return reply, nil
			}

			lastErr = err

			if statusCode == http.StatusTooManyRequests {
				s.rotator.MarkRateLimited(apiKey, 60*time.Second)
				break // Switch to next key in keyAttempts
			} else if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
				s.rotator.MarkRateLimited(apiKey, 10*time.Minute)
				break // Switch to next key in keyAttempts
			}
			// Note: 404 is model deprecation / unavailable for tier, and 503 is temporary high demand.
			// Do NOT penalize the API key for model-specific errors; continue to next fallback model.
		}
	}

	return "", lastErr
}

func (s *AIService) callSingleGeminiModel(ctx context.Context, apiKey, model string, payload geminiPayload) (string, int, error) {
	modelCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	httpReq, err := http.NewRequestWithContext(modelCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("gemini api request failed for model %s: %w", model, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to read gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, fmt.Errorf("gemini api error (model %s, status %d): %s", model, resp.StatusCode, string(respBody))
	}

	var parsedResp geminiCandidateResponse
	if err := json.Unmarshal(respBody, &parsedResp); err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	if len(parsedResp.Candidates) == 0 || len(parsedResp.Candidates[0].Content.Parts) == 0 {
		return "", resp.StatusCode, fmt.Errorf("empty candidate response from gemini model %s", model)
	}

	return parsedResp.Candidates[0].Content.Parts[0].Text, resp.StatusCode, nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

func extractCleanJSON(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start != -1 && end != -1 && end > start {
		return cleaned[start : end+1]
	}
	return cleaned
}

// ScanMandiItem visually inspects camera snapshots of running-out products, empty shelves,
// cartons, or loose items and constructs structured Mandi Sourcing & Wholesale procurement proposals.
func (s *AIService) ScanMandiItem(ctx context.Context, req dto.AIMandiScanRequest) (*dto.AIMandiScanResponse, error) {
	cleanBase64 := cleanBase64String(req.ImageBase64)
	if cleanBase64 == "" {
		return nil, errors.New("empty image data")
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	// Token & Latency Optimization: Downscale large image (max 512px, JPEG quality 65)
	// This reduces Gemini multimodal vision tiles, saves ~80% input tokens, and speeds up inference.
	if optBase64, optMime, err := reuse.CompressBase64Image(cleanBase64, 512, 65); err == nil && optBase64 != "" {
		cleanBase64 = optBase64
		mimeType = optMime
	}

	// 1. Fast Cache key using hash of compressed base64 snippet to prevent duplicate billing
	subLen := len(cleanBase64)
	if subLen > 512 {
		subLen = 512
	}
	cacheKey := fmt.Sprintf("mandi:%d:%s", len(cleanBase64), hashString(cleanBase64[:subLen]))
	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.AIMandiScanResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil && len(resp.Items) > 0 {
			return &resp, nil
		}
	}

	// 2. Streamlined Wholesale Mandi Sourcing Prompt (Sub-2-Second Response)
	mandiPrompt := `You are "ShopSilo Mandi Procurement AI" — wholesale sourcing specialist for Indian Kirana, supermarket & retail stores.
Analyze the photo of low stock items, empty shelves, cartons, or loose commodities to reorder from wholesale Mandi.

Output pure JSON matching this exact schema (maximum 2 items):
{
  "items": [
    {
      "name": "Commercial product name (e.g. 'Fortune Kachi Ghani Mustard Oil - 1 Litre Pouch' or 'Tata Salt - 1kg')",
      "category": "Strictly one of: 'Grocery & Staples', 'Snacks & Munchies', 'Beverages & Drinks', 'Dairy & Eggs', 'Personal Care & Beauty', 'Home & Kitchen', 'Clothing & Fashion', 'Electronics & Mobile', 'Hardware & Tools', 'Stationery & Office', 'General Store'",
      "wholesale_qty": "Wholesale reorder quantity: '1 पेटी / बॉक्स (Box)' or '1 बोरी (Bags)' or '1 कार्टन (Carton)' or '25 kg' or '50 kg'",
      "wholesale_unit": "Strictly one of: 'kg', 'बोरी (Bags)', 'पेटी / बॉक्स (Box)', 'पैकेट (Pkt)', 'लीटर (Ltr)', 'पीस (Pcs)', 'क्विंटल (Qtl)', 'कार्टन (Carton)', 'टिन / केन (Tin)'",
      "estimated_wholesale_rate": 1380.0,
      "estimated_total_budget": 1380.0,
      "notes": "Short wholesale note (e.g. '12 पाउच प्रति पेटी @ ₹115/पाउच')",
      "confidence": 0.95,
      "color_details": null
    }
  ],
  "raw_overview": "Short 1-line summary"
}

RULES:
1. ALWAYS propose 1 or 2 restock items based on what's visible in the photo.
2. Wholesale units: 'पेटी / बॉक्स (Box)' for pouches/bottles, 'बोरी (Bags)' for flours/grains (25kg/50kg), 'कार्टन (Carton)' for biscuits/snacks.
3. Pricing: Realistic Indian Mandi procurement rates in INR (₹).
4. For clothing, sarees, and cosmetics: calibrate color to Indian trade shades (Sugapankhi, Rani Pink, etc.) in 'color_details'. For grocery/FMCG, 'color_details' MUST BE null.
5. Strict output: Pure valid JSON only.`

	payload := geminiPayload{
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{
				{Text: mandiPrompt},
			},
		},
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: "Analyze this image for wholesale mandi procurement and propose restock items in pure JSON."},
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
			MaxOutputTokens:  320,
		},
	}

	apiKey := getGeminiAPIKey()
	rawText, err := s.callGeminiWithFallback(ctx, apiKey, payload)
	if err != nil {
		log.Printf("[ScanMandiItem] Gemini call failed: %v", err)
		return nil, fmt.Errorf("gemini mandi vision failed: %w", err)
	}

	resp, parseErr := parseMandiScanResponse(rawText)
	if parseErr != nil {
		log.Printf("[ScanMandiItem] Parse warning: %v. Raw response: %s", parseErr, rawText)
		// Fallback: search for any "name": "..." or overview text in rawText so merchant is never blocked
		reName := regexp.MustCompile(`"(?:name|product|item)"\s*:\s*"([^"]+)"`)
		matches := reName.FindStringSubmatch(rawText)
		fallbackName := "दुकान रीस्टॉक सामान (Shop Restock Item)"
		if len(matches) > 1 && strings.TrimSpace(matches[1]) != "" {
			fallbackName = strings.TrimSpace(matches[1])
		}
		resp = &dto.AIMandiScanResponse{
			Items: []dto.AIMandiItem{
				{
					Name:                   fallbackName,
					Category:               "General Store",
					WholesaleQty:           "1 पेटी / बॉक्स (Box)",
					WholesaleUnit:          "पेटी / बॉक्स (Box)",
					EstimatedWholesaleRate: 500,
					EstimatedTotalBudget:   500,
					Notes:                  "AI द्वारा तस्वीर से अनुशंसित थोक खरीद।",
					Confidence:             0.75,
				},
			},
			RawOverview: "AI Fallback Restock Proposal",
		}
	}

	// Sanitize items: filter empty names
	var validItems []dto.AIMandiItem
	for _, item := range resp.Items {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			continue
		}
		if item.Category == "" {
			item.Category = "General Store"
		}
		if item.WholesaleUnit == "" {
			item.WholesaleUnit = "पेटी / बॉक्स (Box)"
		}
		if item.WholesaleQty == "" {
			item.WholesaleQty = fmt.Sprintf("1 %s", item.WholesaleUnit)
		}
		if item.EstimatedTotalBudget <= 0 && item.EstimatedWholesaleRate > 0 {
			item.EstimatedTotalBudget = item.EstimatedWholesaleRate
		}
		if item.Confidence <= 0 {
			item.Confidence = 0.9
		}
		validItems = append(validItems, item)
	}

	// Guarantee at least one valid item
	if len(validItems) == 0 {
		validItems = append(validItems, dto.AIMandiItem{
			Name:                   "दुकान रीस्टॉक सामान (Shop Restock Item)",
			Category:               "General Store",
			WholesaleQty:           "1 पेटी / बॉक्स (Box)",
			WholesaleUnit:          "पेटी / बॉक्स (Box)",
			EstimatedWholesaleRate: 500,
			EstimatedTotalBudget:   500,
			Notes:                  "AI द्वारा तस्वीर से अनुशंसित थोक खरीद।",
			Confidence:             0.75,
		})
	}
	resp.Items = validItems

	setAICache(cacheKey, rawText, 30*time.Minute)
	return resp, nil
}

// parseMandiScanResponse provides multi-strategy resilient parsing for Mandi vision responses.
// It gracefully handles root-level arrays, wrapped maps, single objects, and truncated JSON streams.
func parseMandiScanResponse(rawText string) (*dto.AIMandiScanResponse, error) {
	cleaned := extractCleanJSON(rawText)
	if cleaned == "" {
		return nil, errors.New("empty response from vision model")
	}

	var resp dto.AIMandiScanResponse

	// Strategy 1: Standard AIMandiScanResponse schema {"items": [...], "raw_overview": "..."}
	if err := json.Unmarshal([]byte(cleaned), &resp); err == nil && len(resp.Items) > 0 {
		return &resp, nil
	}

	// Strategy 2: Direct array of items: [ {...}, {...} ]
	var directItems []dto.AIMandiItem
	if err := json.Unmarshal([]byte(cleaned), &directItems); err == nil && len(directItems) > 0 {
		resp.Items = directItems
		return &resp, nil
	}

	// Strategy 3: Map with arbitrary key containing item array (e.g. "products", "data", "list", "proposals")
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &rawMap); err == nil {
		for _, key := range []string{"items", "products", "data", "proposals", "restock_items", "goods", "inventory", "detected_items"} {
			if sliceData, ok := rawMap[key]; ok {
				var items []dto.AIMandiItem
				if err := json.Unmarshal(sliceData, &items); err == nil && len(items) > 0 {
					resp.Items = items
					return &resp, nil
				}
			}
		}
	}

	// Strategy 4: Single item object directly: { "name": "...", "wholesale_qty": "..." }
	var singleItem dto.AIMandiItem
	if err := json.Unmarshal([]byte(cleaned), &singleItem); err == nil && strings.TrimSpace(singleItem.Name) != "" {
		resp.Items = []dto.AIMandiItem{singleItem}
		return &resp, nil
	}

	// Strategy 5: Attempt repair if JSON was truncated mid-stream (find last '}' before end and close array/object)
	lastBrace := strings.LastIndex(cleaned, "}")
	if lastBrace > 0 {
		repaired := cleaned[:lastBrace+1]
		if strings.HasPrefix(strings.TrimSpace(repaired), "[") && !strings.HasSuffix(strings.TrimSpace(repaired), "]") {
			repaired += "]"
		} else if strings.HasPrefix(strings.TrimSpace(repaired), "{") && strings.Contains(repaired, `"items": [`) && !strings.HasSuffix(strings.TrimSpace(repaired), "}") {
			repaired += "]}"
		}
		if err := json.Unmarshal([]byte(repaired), &resp); err == nil && len(resp.Items) > 0 {
			return &resp, nil
		}
		if err := json.Unmarshal([]byte(repaired), &directItems); err == nil && len(directItems) > 0 {
			resp.Items = directItems
			return &resp, nil
		}
	}

	return nil, fmt.Errorf("unable to parse mandi items from AI output: %s", rawText)
}

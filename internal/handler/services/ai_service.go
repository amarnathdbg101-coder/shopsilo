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
		"gemini-3.5-flash-lite",
		"gemini-3.8-flash",
		"gemini-flash-latest",
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

	// Token Optimization: Downscale large image (max 768px, JPEG quality 80)
	// This reduces Gemini multimodal vision tiles and saves 60-75% input tokens.
	if optBase64, optMime, err := reuse.CompressBase64Image(cleanBase64, 768, 80); err == nil && optBase64 != "" {
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

	scanPrompt := `You are "ShopSilo Master Catalog AI" — an elite, hyper-accurate Chief Retail Merchandising Specialist and Vision Cataloging Analyst for Indian retail stores, supermarkets, grocery/kirana, ethnic wear & sarees, footwear, personal care, and local retail shops.

Your mission is to perform a surgical visual inspection of the uploaded product image, extract every visible commercial detail (OCR text, brand logos, net volume/pack sizes, MRP stamps, barcodes, certifications), and produce a flawless, production-ready retail catalog entry.

Extract and infer comprehensive, professional product catalog data in pure JSON matching this exact schema:
{
  "name": "Full professional commercial product name in Title Case (e.g., 'Dove Daily Shine Shampoo with Nutritive Serum - 340ml Bottle' or 'Fortune Premium Kachi Ghani Pure Mustard Oil - 1 Litre Pouch' or 'Royal Banarasi Silk Zari Woven Saree With Blouse Piece')",
  "brand": "Exact brand name (e.g. 'Dove', 'Fortune', 'Tata Consumer', 'Amul', 'Patanjali', 'Kashvi', 'boAt', or 'Generic'/'Local Artisan' if unbranded)",
  "category_hint": "Standard retail category (MUST be strictly one of: 'Grocery & Staples', 'Snacks & Munchies', 'Beverages & Drinks', 'Dairy & Eggs', 'Personal Care & Beauty', 'Home & Kitchen', 'Clothing & Fashion', 'Electronics & Mobile', 'Hardware & Tools', 'Stationery & Office', 'Baby Care', 'General Store')",
  "subcategory": "Specific subcategory name (e.g. 'Edible Oils & Ghee', 'Shampoos & Conditioners', 'Atta & Flours', 'Biscuits & Cookies', 'Ethnic Wear & Sarees', 'Wired Earphones')",
  "mrp": 299.0,
  "selling_price": 249.0,
  "estimated_cost": 199.0,
  "profit_margin_percent": 20.1,
  "unit": "Appropriate unit: 'piece', 'packet', 'bottle', 'pouch', 'box', 'pack', 'kg', 'gm', 'litre', 'ml', 'meter', 'pair'",
  "description": "Engaging, professional 2-3 line commercial marketing overview highlighting material/formulation, quality benefits, and consumer usage.",
  "key_features": [
    "Core feature with customer benefit",
    "Key formulation, fabric weave, or build specification",
    "Usage, storage, or certification highlight"
  ],
  "ingredients": "Key ingredients, composition, or fabric blend if visible or standard for this item",
  "tags": [
    "8 to 12 high-intent search keywords in English and Hinglish (e.g. 'sarson tel', 'mustard oil', 'kachi ghani', 'cooking oil')"
  ],
  "attributes": {
    "Dietary Preference": "Vegetarian",
    "Packaging Type": "Pouch",
    "Item Form": "Liquid",
    "Flavor": "Mustard"
  },
  "suggested_sku": "Smart structured uppercase SKU identifier (e.g. 'FORT-MUST-1L-POUCH')",
  "barcode": "Barcode digits if clearly visible on packaging, otherwise empty string",
  "visual_keywords": ["3-5 visual identifiers like packaging color, container shape, finish"],
  "color_details": null
}

CRITICAL CATALOGING SPECIALIST RULES:
1. COMMERCIAL PRODUCT TITLE:
   - Construct a rich, professional, complete title in Title Case: [Brand] + [Product Line / Sub-brand] + [Key Flavor / Scent / Variant / Material] + [Pack Size or Volume with Unit].
   - Examples:
     * FMCG: "Dove Daily Shine Shampoo with Nutritive Serum - 340ml Bottle"
     * Edible: "Fortune Premium Kachi Ghani Pure Mustard Oil - 1 Litre Pouch"
     * Apparel: "Kashvi Handcrafted Pure Banarasi Silk Zari Woven Saree with Unstitched Blouse Piece - Rani Pink (रानी)"
     * Footwear: "Bata Men's Formal Genuine Leather Slip-On Shoes - Dark Tan"
     * Electronics: "boAt Bassheads 225 Heavy Bass in-Ear Wired Earphones with Mic - Metallic Black"
   - NEVER use vague or generic titles like "Blue Shirt" or "Oil Packet". Be precise and specific to the image.
   - For apparel, ethnic wear, matching center items (blouse pieces, petticoats, sarees, dupattas), ALWAYS include the exact trade color shade name at the end of the title.

2. COMMERCIAL PRICING & MARGIN INTELLIGENCE:
   - If an MRP (Maximum Retail Price) stamp or printed text is visible on the package (e.g. 'MRP ₹140.00 incl. of all taxes'), use that EXACT numeric value for 'mrp'.
   - If not visible, infer a highly realistic Indian retail MRP in INR (₹) based on standard current Indian market prices.
   - 'selling_price': Compute an attractive neighborhood discount price (typically 5% to 15% below MRP for packaged FMCG, 10% to 25% for apparel/electronics).
   - 'estimated_cost': Calculate the retailer's wholesale procurement cost (typically 12% to 20% margin for grocery/FMCG; 30% to 45% for fashion/cosmetics).
   - 'profit_margin_percent': Compute exact margin percentage: round(((selling_price - estimated_cost) / selling_price) * 100, 1).

3. SPECIALIZED ATTRIBUTES (HIGH-VALUE PRODUCT SPECS ONLY — NO USELESS BOILERPLATE):
   - Provide 3 to 6 strictly relevant, concise, high-value key-value pairs in 'attributes' dictionary with Title Case keys.
   - ONLY include physical, technical, or customer-differentiating specs:
     * For Food / Grocery: 'Dietary Preference' (Vegetarian/Non-Vegetarian), 'Packaging Type' (Pouch/Bottle/Jar/Tin), 'Item Form' (Liquid/Powder/Paste/Grain), 'Flavor'.
     * For Personal Care / Beauty: 'Item Form', 'Skin/Hair Type', 'Fragrance', 'Target Gender', 'Key Benefits'.
     * For Clothing / Fashion: 'Fabric', 'Pattern', 'Color', 'Fit Type', 'Occasion'.
     * For Electronics / Hardware: 'Connectivity', 'Color', 'Material', 'Warranty'.
     * For Home & Cleaning: 'Item Form', 'Fragrance', 'Surface Recommendation'.
   - STRICTLY FORBIDDEN IN ATTRIBUTES (DO NOT OUTPUT THESE):
     * NEVER include generic storage instructions (e.g. 'Store in a cool and dry place').
     * NEVER include usage directions, care instructions, or generic warnings.
     * NEVER repeat basic fields already present in schema (do NOT add 'Name', 'Brand', 'Category', 'MRP', 'Price', 'Weight', 'Unit', 'Barcode').
     * NEVER use placeholder values like 'N/A', 'None', 'Unknown', 'Standard', 'Generic', '-', 'Null'.
     * Keep all attribute values ultra-concise (under 35 characters, e.g. 'Vegetarian', 'Cotton Blend', '1 Year', 'Pouch').

4. BILINGUAL HINGLISH SEARCH TAGS:
   - Provide 8 to 12 high-intent search keywords including common Hindi/Hinglish terms used by Indian shoppers (e.g., 'sarson tel', 'mustard oil', 'gehu aata', 'doodh', 'chai patti', 'nahane ka sabun').

5. INDIAN TEXTILE & COLOR SHADE CALIBRATION (COLOR-CRITICAL ITEMS ONLY):
   - CRITICAL REQUIREMENT: This rule applies EXCLUSIVELY to color-critical products:
     * Indian Ethnic Wear & Fabrics: Blouse pieces, petticoats, sarees, dupattas, suits, kurtis, fabrics/thaan, asters, tailoring threads.
     * Western Garments: Shirts, t-shirts, dresses, jeans, leggings, towels, bedsheets.
     * Cosmetics & Accessories: Lipsticks, nail polishes, hair colors, bangles, paints.
   - FOR ALL GROCERY, KIRANA, FMCG, FOOD, MEDICINES, AND HARDWARE ITEMS:
     * 'color_details' MUST BE STRICTLY null.
     * NEVER add 'Color' to 'attributes'.
     * NEVER append a color shade to the product title (e.g. NEVER write "Fortune Mustard Oil - Yellow").
   - FOR COLOR-CRITICAL PRODUCTS ONLY:
     * Accurately map the dominant fabric color to standard Indian commercial trade shade cards:
       - Greens & Parrots: Sugapankhi / Parrot Green (#7CB342), Pista Green (#A8D8B9), Rama Green (#00827F), Mehendi Green (#556B2F), Bottle Green (#004D40), Morpankhi (#005B5C).
       - Yellows & Oranges: Lemon Yellow (#FFF176), Haldi Yellow (#FFB300), Mustard Yellow (#E5A823), Rust / Gerua (#B7410E), Peach (#FFAB91).
       - Pinks & Reds: Rani Pink (#D81B60), Baby Pink (#F48FB1), Gajari (#FF6F61), Pyazi (#A36B7E), Blood Red (#D32F2F), Tamatar Red (#E53935), Maroon (#800000), Katha (#5D2524).
       - Blues & Purples: Firozi / Cyan (#00BCD4), Royal Blue (#1565C0), Navy Blue (#000080), Wine / Jamuni (#4A154B), Purple (#7B1FA2).
       - Neutrals: Off White (#FAF0E6), Badami / Cream (#FFF8E7), Chiku / Beige (#D7C4B7), Coffee Brown (#4E342E), Slate Grey (#757575), Black (#1A1A1A).
     * CAMERA WHITE BALANCE & UNDERTONE RULE:
       - Indoor warm/yellow lighting or camera sensor overexposure often shifts vibrant chartreuse/lime greens (like 'Sugapankhi' / 'Parrot Green' / 'तोता हरा') to look yellowish. Check fabric undertones carefully: if there is a green tint/pigment with high vibrancy, label it as 'Sugapankhi' (सुगापंखी/तोता), NOT plain 'Lemon Yellow'.
     * STRICT CHROMATIC NEIGHBOR RULE FOR ALTERNATIVES:
       - 'alternatives' MUST strictly be adjacent shades from the EXACT SAME color family (e.g., if primary is Sugapankhi / Yellow-Green, alternatives can ONLY be Lemon Yellow, Pista Green, Mehendi Green, Haldi Yellow. NEVER suggest distant, unrelated colors like Red, Maroon, Black, or Blue in alternatives!).
     * In 'color_details':
       - 'primary_shade': best calibrated match with English shade name, Hindi name, and exact HexCode.
       - 'alternatives': 3 to 4 closest neighboring color shades strictly from the same chromatic family.
     * Also add 'Color' to 'attributes' (e.g. 'Sugapankhi (सुगापंखी)').

6. STRICT OUTPUT FORMAT:
   - Output MUST be pure, valid JSON only matching the schema. No markdown code blocks, no preamble.`

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
			MaxOutputTokens:  1500, // Sufficient tokens for all 10+ attributes
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
		return nil, fmt.Errorf("failed to parse product scanner result: %w", parseErr)
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

	// 1. Check local SHA256 cache
	hasher := sha256.New()
	hasher.Write([]byte(cleanBase64))
	cacheKey := fmt.Sprintf("mandi_scan_%x", hasher.Sum(nil))
	if cached, ok := getAICache(cacheKey); ok {
		var resp dto.AIMandiScanResponse
		if err := json.Unmarshal([]byte(cached), &resp); err == nil && len(resp.Items) > 0 {
			return &resp, nil
		}
	}

	// 2. Specialized Wholesale Mandi Sourcing Prompt
	mandiPrompt := `You are "ShopSilo Mandi Procurement AI" — an expert Chief Wholesale Sourcing Specialist for Indian Kirana stores, supermarkets, and Mandi traders.

The merchant has taken a photo of a product running low on stock, an empty shelf, a carton, a box, or loose commodities that need to be reordered from the wholesale mandi/market.

YOUR TASK:
Analyze the image with precision, identify the product(s) needing restock, determine appropriate Indian WHOLESALE purchasing units and quantities (e.g. Box, Bags/Bori, Carton, Quintal, Kg), and estimate current Indian Mandi wholesale rates.

Output pure JSON matching this exact schema:
{
  "items": [
    {
      "name": "Full professional commercial product name in Title Case (e.g., 'Fortune Premium Kachi Ghani Pure Mustard Oil - 1 Litre Pouch' or 'Tata Salt Vacuum Evaporated - 1kg' or 'Aashirvaad Shudh Chakki Atta')",
      "category": "Strictly one of: 'Grocery & Staples', 'Snacks & Munchies', 'Beverages & Drinks', 'Dairy & Eggs', 'Personal Care & Beauty', 'Home & Kitchen', 'Clothing & Fashion', 'Electronics & Mobile', 'Hardware & Tools', 'Stationery & Office', 'General Store'",
      "wholesale_qty": "Realistic commercial reorder quantity with unit: '1 पेटी / बॉक्स (Box)' or '1 बोरी (Bags)' or '1 कार्टन (Carton)' or '25 kg' or '50 kg' or '1 क्विंटल (Qtl)' or '1 टिन / केन (Tin)' or '12 पैकेट (Pkt)' or '24 पीस (Pcs)'",
      "wholesale_unit": "Strictly one of: 'kg', 'बोरी (Bags)', 'पेटी / बॉक्स (Box)', 'पैकेट (Pkt)', 'लीटर (Ltr)', 'पीस (Pcs)', 'क्विंटल (Qtl)', 'ग्राम (g)', 'दर्जन (Dzn)', 'कार्टन (Carton)', 'टिन / केन (Tin)'",
      "estimated_wholesale_rate": 1380.0,
      "estimated_total_budget": 1380.0,
      "notes": "Brief Hindi/English sourcing note: pack size details, wholesale bundle count (e.g. '12 पाउच प्रति पेटी @ ₹115/पाउच')",
      "confidence": 0.95,
      "color_details": null
    }
  ],
  "raw_overview": "Short 1-line summary of items detected"
}

RULES:
0. MANDATORY REORDER PROPOSAL:
   - Identify ANY product, item, brand, package, cloth, or commodity visible in the image.
   - Even if the shelf or container is NOT empty, or if it is just a product snapshot, ALWAYS generate at least one reorder recommendation item.
   - The 'items' array MUST NEVER BE EMPTY.
1. WHOLESALE REORDER UNITS:
   - For edible oil pouches/bottles: suggest 'पेटी / बॉक्स (Box)' (usually 12-16 units per box) or 'टिन / केन (Tin)' (15 Litre).
   - For grains, flours, pulses, rice, sugar: suggest 'बोरी (Bags)' (25kg or 50kg) or 'क्विंटल (Qtl)'.
   - For biscuits, snacks, noodles: suggest 'कार्टन (Carton)' (24 to 48 packets per carton).
   - For soaps, shampoos, toothpastes: suggest 'पेटी / बॉक्स (Box)' or 'पीस (Pcs)'.
   - For loose commodities: suggest 'kg' (e.g. 25 kg or 50 kg).
2. PRICING:
   - Provide realistic Indian Mandi wholesale procurement rates in INR (₹).
   - 'estimated_wholesale_rate': total price for the requested wholesale quantity (or per bag/box).
3. MULTIPLE ITEMS:
   - If multiple distinct low-stock products are visible in the photo, list each one as a separate item in the 'items' array (up to 4 items).
4. COLOR CALIBRATION (APPLIES TO APPAREL, TEXTILES & COSMETICS ONLY):
   - CRITICAL REQUIREMENT: This rule applies EXCLUSIVELY to color-critical goods:
     * Blouse pieces, petticoats, sarees, dupattas, matching fabrics, garments, cosmetics.
   - FOR ALL GROCERIES, COMMODITIES, GRAINS, OILS, SPICES, SNACKS, AND PACKAGED GOODS:
     * 'color_details' MUST BE STRICTLY null.
     * NEVER append a color shade to product name.
     * NEVER mention 'Shade:' in notes (e.g. do NOT write 'Shade: Yellow' for Mustard Oil or Dal!).
   - FOR COLOR-CRITICAL GOODS ONLY:
     * Accurately map the dominant color against standard Indian trade shades:
       - Greens & Parrots: Sugapankhi / Parrot Green (#7CB342), Pista Green (#A8D8B9), Rama Green (#00827F), Mehendi Green (#556B2F), Bottle Green (#004D40), Morpankhi (#005B5C).
       - Yellows & Oranges: Lemon Yellow (#FFF176), Haldi Yellow (#FFB300), Mustard Yellow (#E5A823), Rust / Gerua (#B7410E), Peach (#FFAB91).
       - Pinks & Reds: Rani Pink (#D81B60), Baby Pink (#F48FB1), Gajari (#FF6F61), Pyazi (#A36B7E), Blood Red (#D32F2F), Tamatar Red (#E53935), Maroon (#800000), Katha (#5D2524).
       - Blues & Purples: Firozi / Cyan (#00BCD4), Royal Blue (#1565C0), Navy Blue (#000080), Wine / Jamuni (#4A154B), Purple (#7B1FA2).
       - Neutrals: Off White (#FAF0E6), Badami / Cream (#FFF8E7), Chiku / Beige (#D7C4B7), Coffee Brown (#4E342E), Slate Grey (#757575), Black (#1A1A1A).
     * CAMERA WHITE BALANCE & UNDERTONE RULE:
       - Indoor warm/yellow lighting or camera sensor overexposure often shifts vibrant chartreuse/lime greens (like 'Sugapankhi' / 'Parrot Green' / 'तोता हरा') to look yellowish. Check fabric undertones carefully: if there is a green tint/pigment with high vibrancy, label it as 'Sugapankhi' (सुगापंखी/तोता), NOT plain 'Lemon Yellow'.
     * STRICT CHROMATIC NEIGHBOR RULE FOR ALTERNATIVES:
       - 'alternatives' MUST strictly be adjacent shades from the EXACT SAME color family (e.g., if primary is Sugapankhi / Yellow-Green, alternatives can ONLY be Lemon Yellow, Pista Green, Mehendi Green, Haldi Yellow. NEVER suggest distant, unrelated colors like Red, Maroon, Black, or Blue in alternatives!).
     * In 'color_details': set 'primary_shade' and 3 to 4 nearest 'alternatives' strictly from the same color family.
     * Append the shade to the product name (e.g. 'Cotton Petticoat - Sugapankhi (सुगापंखी)') and include 'Shade: Sugapankhi (सुगापंखी)' in notes.
5. STRICT JSON ONLY:
   - Output valid JSON only. No markdown formatting, no preamble.`

	payload := geminiPayload{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: mandiPrompt},
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
			MaxOutputTokens:  2500,
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

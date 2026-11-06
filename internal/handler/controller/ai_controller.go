package controller

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type AIController interface {
	CustomerChat(w http.ResponseWriter, r *http.Request)
	MerchantCopilot(w http.ResponseWriter, r *http.Request)
	ScanProduct(w http.ResponseWriter, r *http.Request)
	SmartStudioScan(w http.ResponseWriter, r *http.Request)
	ParseParchi(w http.ResponseWriter, r *http.Request)
	SemanticSearch(w http.ResponseWriter, r *http.Request)
	VoiceBill(w http.ResponseWriter, r *http.Request)
	GenerateMarketingCampaign(w http.ResponseWriter, r *http.Request)
	BargainAssist(w http.ResponseWriter, r *http.Request)
	ParseSupplierInvoice(w http.ResponseWriter, r *http.Request)
	GetCrossSellRecommendations(w http.ResponseWriter, r *http.Request)
	GetDeadStockBundles(w http.ResponseWriter, r *http.Request)
	GenerateSmartRecoveryNudge(w http.ResponseWriter, r *http.Request)
}

type aiController struct {
	aiService services.AIService
}

func NewAIController(aiService services.AIService) AIController {
	return &aiController{aiService: aiService}
}

func (c *aiController) CustomerChat(w http.ResponseWriter, r *http.Request) {
	var req dto.CustomerAIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Prompt == "" {
		reuse.Error(w, http.StatusBadRequest, "prompt is required")
		return
	}

	resp, err := c.aiService.CustomerChat(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "AI response generated", resp)
}

func (c *aiController) MerchantCopilot(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.ContextKeyUser).(*middleware.Claims)
	var userID string
	if ok && claims != nil {
		userID = claims.UserID
	}

	var req dto.MerchantAICopilotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Prompt == "" {
		reuse.Error(w, http.StatusBadRequest, "prompt is required")
		return
	}

	resp, err := c.aiService.MerchantCopilot(r.Context(), userID, req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Merchant Copilot response generated", resp)
}

func (c *aiController) ScanProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.AIScanProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image_base64 is required")
		return
	}

	resp, err := c.aiService.ScanProduct(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product packet scanned successfully", resp)
}

func (c *aiController) SmartStudioScan(w http.ResponseWriter, r *http.Request) {
	var req dto.SmartStudioScanRequest

	// Check if multipart form was sent (direct camera photo upload)
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// Parse max 10MB
		if err := r.ParseMultipartForm(10 * 1024 * 1024); err == nil {
			file, header, fileErr := r.FormFile("image")
			if fileErr == nil {
				defer file.Close()
				imgBytes, _ := io.ReadAll(file)
				req.ImageBase64 = base64.StdEncoding.EncodeToString(imgBytes)
				req.MimeType = header.Header.Get("Content-Type")
				req.ThemeHint = r.FormValue("theme_hint")
			}
		}
	} else {
		// JSON payload with Base64 image
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			reuse.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image is required (as multipart form 'image' or json 'image_base64')")
		return
	}

	resp, err := c.aiService.SmartStudioScan(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Product smart studio scan completed successfully", resp)
}

func (c *aiController) ParseParchi(w http.ResponseWriter, r *http.Request) {
	var req dto.ParseParchiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RawText == "" {
		reuse.Error(w, http.StatusBadRequest, "raw_text is required")
		return
	}

	resp, err := c.aiService.ParseParchi(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Parchi parsed successfully", resp)
}

func (c *aiController) SemanticSearch(w http.ResponseWriter, r *http.Request) {
	var req dto.SemanticSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		reuse.Error(w, http.StatusBadRequest, "query is required")
		return
	}

	resp, err := c.aiService.SemanticSearch(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Semantic search intent resolved", resp)
}

func (c *aiController) VoiceBill(w http.ResponseWriter, r *http.Request) {
	var req dto.VoiceBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SpokenText == "" {
		reuse.Error(w, http.StatusBadRequest, "spoken_text is required")
		return
	}

	resp, err := c.aiService.VoiceBill(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Voice bill parsed successfully", resp)
}

func (c *aiController) GenerateMarketingCampaign(w http.ResponseWriter, r *http.Request) {
	var req dto.AIMarketingCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := c.aiService.GenerateMarketingCampaign(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Marketing campaign generated successfully", resp)
}

func (c *aiController) BargainAssist(w http.ResponseWriter, r *http.Request) {
	var req dto.AIBargainAssistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProductName == "" || req.MRP <= 0 || req.CostPrice <= 0 {
		reuse.Error(w, http.StatusBadRequest, "product_name, mrp, and cost_price are required")
		return
	}

	resp, err := c.aiService.BargainAssist(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Bargain assist advice generated successfully", resp)
}

func (c *aiController) ParseSupplierInvoice(w http.ResponseWriter, r *http.Request) {
	var req dto.ParseSupplierInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ImageBase64 == "" {
		reuse.Error(w, http.StatusBadRequest, "image_base64 is required")
		return
	}

	resp, err := c.aiService.ParseSupplierInvoice(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Supplier invoice parsed successfully via OCR", resp)
}

func (c *aiController) GetCrossSellRecommendations(w http.ResponseWriter, r *http.Request) {
	var req dto.AICrossSellRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ScannedProductName == "" {
		reuse.Error(w, http.StatusBadRequest, "scanned_product_name is required")
		return
	}

	resp, err := c.aiService.GetCrossSellRecommendations(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Cross-sell recommendations generated", resp)
}

func (c *aiController) GetDeadStockBundles(w http.ResponseWriter, r *http.Request) {
	shopID := r.URL.Query().Get("shop_id")
	resp, err := c.aiService.GetDeadStockBundleSuggestions(r.Context(), shopID)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Dead stock liquidation bundles suggested", resp)
}

func (c *aiController) GenerateSmartRecoveryNudge(w http.ResponseWriter, r *http.Request) {
	var req dto.AIRecoveryNudgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CustomerName == "" || req.TotalDebt <= 0 {
		reuse.Error(w, http.StatusBadRequest, "customer_name and total_debt (>0) are required")
		return
	}

	resp, err := c.aiService.GenerateSmartRecoveryNudge(r.Context(), req)
	if err != nil {
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "Smart recovery nudges generated", resp)
}

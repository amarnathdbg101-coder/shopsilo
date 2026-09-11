package controller

import (
	"encoding/json"
	"net/http"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
)

type AIController interface {
	CustomerChat(w http.ResponseWriter, r *http.Request)
	MerchantCopilot(w http.ResponseWriter, r *http.Request)
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

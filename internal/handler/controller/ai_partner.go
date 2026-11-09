package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// AIPartnerController handles incoming HTTP endpoints for the Autonomous AI Business Partner.
type AIPartnerController struct {
	partnerService *services.AIPartnerService
	shopService    *services.ShopService
	logger         *zap.Logger
}

// NewAIPartnerController creates an initialized AIPartnerController.
func NewAIPartnerController(
	partnerService *services.AIPartnerService,
	shopService *services.ShopService,
	logger *zap.Logger,
) *AIPartnerController {
	return &AIPartnerController{
		partnerService: partnerService,
		shopService:    shopService,
		logger:         logger,
	}
}

// Chat handles dialogue turns with the AI Business Partner (Protected - Shopkeeper).
func (c *AIPartnerController) Chat(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	var req dto.AIPartnerChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		reuse.Error(w, http.StatusBadRequest, "message cannot be empty")
		return
	}

	resp, err := c.partnerService.Chat(r.Context(), shop.ID, claims.UserID, req.Persona, req.Message, shop)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to process ai partner chat", zap.Error(err), zap.String("shop_id", shop.ID))
		}
		reuse.Error(w, http.StatusInternalServerError, "failed to process dialogue turn")
		return
	}

	reuse.Success(w, "AI Partner response generated successfully", resp)
}

// GetBriefings returns active, unacknowledged morning kickoff, evening settle, or anomaly alerts.
func (c *AIPartnerController) GetBriefings(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	briefings, err := c.partnerService.GetActiveBriefings(r.Context(), shop.ID)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to fetch active briefings", zap.Error(err), zap.String("shop_id", shop.ID))
		}
		reuse.Error(w, http.StatusInternalServerError, "failed to fetch proactive briefings")
		return
	}

	reuse.Success(w, "Active briefings retrieved successfully", briefings)
}

// DismissBriefing acknowledges and dismisses a briefing card.
func (c *AIPartnerController) DismissBriefing(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	briefingID := chi.URLParam(r, "id")
	if briefingID == "" {
		reuse.Error(w, http.StatusBadRequest, "briefing id is required")
		return
	}

	if err := c.partnerService.DismissBriefing(r.Context(), shop.ID, briefingID); err != nil {
		if c.logger != nil {
			c.logger.Error("failed to dismiss briefing", zap.Error(err), zap.String("briefing_id", briefingID))
		}
		reuse.Error(w, http.StatusInternalServerError, "failed to dismiss briefing")
		return
	}

	reuse.Success(w, "Briefing dismissed successfully", nil)
}

// ExecuteAction handles approval or cancellation of high-impact AI tool actions.
func (c *AIPartnerController) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	var req dto.AIConfirmActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.ActionID) == "" {
		reuse.Error(w, http.StatusBadRequest, "action_id is required")
		return
	}

	res, err := c.partnerService.ExecuteAction(r.Context(), shop.ID, claims.UserID, req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to execute ai partner action", zap.Error(err), zap.String("action_id", req.ActionID))
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "AI action executed successfully", res)
}

// RollbackAction reverses a previously executed AI action if rollback metadata is available.
func (c *AIPartnerController) RollbackAction(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	actionID := chi.URLParam(r, "id")
	if strings.TrimSpace(actionID) == "" {
		reuse.Error(w, http.StatusBadRequest, "action id is required")
		return
	}

	res, err := c.partnerService.RollbackAction(r.Context(), shop.ID, claims.UserID, actionID)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to rollback ai partner action", zap.Error(err), zap.String("action_id", actionID))
		}
		reuse.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	reuse.Success(w, "AI action rolled back successfully", res)
}

// ListActionLogs retrieves recent tool executions and audit logs for the shop.
func (c *AIPartnerController) ListActionLogs(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if c.shopService == nil || c.partnerService == nil {
		reuse.Error(w, http.StatusServiceUnavailable, "ai partner service temporarily unavailable")
		return
	}

	shop, err := c.shopService.GetMyShop(r.Context(), claims.UserID)
	if err != nil || shop == nil {
		reuse.Error(w, http.StatusNotFound, "merchant shop not found")
		return
	}

	logs, err := c.partnerService.ListActionLogs(r.Context(), shop.ID, 30)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to list ai action logs", zap.Error(err), zap.String("shop_id", shop.ID))
		}
		reuse.Error(w, http.StatusInternalServerError, "failed to list action logs")
		return
	}

	reuse.Success(w, "Action audit logs retrieved successfully", logs)
}

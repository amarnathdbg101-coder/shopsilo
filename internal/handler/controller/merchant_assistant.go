package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"

	"github.com/go-chi/chi/v5"
)

type MerchantAssistantUseCase interface {
	Chat(ctx context.Context, userID string, req dto.MerchantAssistantChatRequest) (*dto.MerchantAssistantChatResponse, error)
	ListConversations(ctx context.Context, userID string) ([]dto.MerchantAssistantConversation, error)
	ListMessages(ctx context.Context, userID, conversationID string) ([]dto.MerchantAssistantMessage, error)
}

type MerchantAssistantController struct {
	service MerchantAssistantUseCase
}

func NewMerchantAssistantController(service MerchantAssistantUseCase) *MerchantAssistantController {
	return &MerchantAssistantController{service: service}
}

func (c *MerchantAssistantController) Chat(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req dto.MerchantAssistantChatRequest
	if err := decoder.Decode(&req); err != nil {
		reuse.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		reuse.Error(w, http.StatusBadRequest, "request must contain one JSON object")
		return
	}

	response, err := c.service.Chat(r.Context(), claims.UserID, req)
	if err != nil {
		writeMerchantAssistantError(w, err)
		return
	}
	reuse.Success(w, "Merchant AI response generated", response)
}

func (c *MerchantAssistantController) ListConversations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	conversations, err := c.service.ListConversations(r.Context(), claims.UserID)
	if err != nil {
		writeMerchantAssistantError(w, err)
		return
	}
	reuse.Success(w, "Merchant AI conversations retrieved", dto.MerchantAssistantConversationsResponse{
		Conversations: conversations,
	})
}

func (c *MerchantAssistantController) ListMessages(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.UserID == "" {
		reuse.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	messages, err := c.service.ListMessages(r.Context(), claims.UserID, chi.URLParam(r, "conversationID"))
	if err != nil {
		writeMerchantAssistantError(w, err)
		return
	}
	reuse.Success(w, "Merchant AI messages retrieved", dto.MerchantAssistantMessagesResponse{
		Messages: messages,
	})
}

func writeMerchantAssistantError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrMerchantAssistantInvalidRequest):
		reuse.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrMerchantAssistantConversation), errors.Is(err, services.ErrShopNotFound):
		reuse.Error(w, http.StatusNotFound, "shop or conversation not found")
	case errors.Is(err, services.ErrMerchantAssistantModel):
		reuse.Error(w, http.StatusServiceUnavailable, "AI assistant is temporarily unavailable")
	case errors.Is(err, services.ErrMerchantAssistantToolLimit):
		reuse.Error(w, http.StatusBadGateway, "AI assistant could not complete this request safely")
	default:
		reuse.Error(w, http.StatusInternalServerError, "AI assistant could not process the request")
	}
}

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
)

type merchantAssistantTestUseCase struct {
	called bool
}

func (s *merchantAssistantTestUseCase) Chat(context.Context, string, dto.MerchantAssistantChatRequest) (*dto.MerchantAssistantChatResponse, error) {
	s.called = true
	return &dto.MerchantAssistantChatResponse{Text: "ok"}, nil
}

func (s *merchantAssistantTestUseCase) ListConversations(context.Context, string) ([]dto.MerchantAssistantConversation, error) {
	return nil, nil
}

func (s *merchantAssistantTestUseCase) ListMessages(context.Context, string, string) ([]dto.MerchantAssistantMessage, error) {
	return nil, nil
}

func TestMerchantAssistantChatRequiresAuthenticatedUser(t *testing.T) {
	service := &merchantAssistantTestUseCase{}
	controller := NewMerchantAssistantController(service)
	request := httptest.NewRequest(http.MethodPost, "/ai/merchant-assistant/chat", strings.NewReader(`{"prompt":"sales"}`))
	response := httptest.NewRecorder()

	controller.Chat(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	if service.called {
		t.Fatal("unauthenticated request reached the service")
	}
}

func TestMerchantAssistantChatRejectsClientSuppliedShopScope(t *testing.T) {
	service := &merchantAssistantTestUseCase{}
	controller := NewMerchantAssistantController(service)
	request := httptest.NewRequest(http.MethodPost, "/ai/merchant-assistant/chat", strings.NewReader(`{"prompt":"sales","shop_id":"other-shop"}`))
	request = request.WithContext(context.WithValue(request.Context(), middleware.ContextKeyUser, &middleware.Claims{UserID: "owner-1"}))
	response := httptest.NewRecorder()

	controller.Chat(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
	if service.called {
		t.Fatal("request with client-supplied shop scope reached the service")
	}
}

package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
)

func TestAIPartnerController_AuthAndValidation(t *testing.T) {
	ctrl := controller.NewAIPartnerController(nil, nil, nil)

	t.Run("Chat Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.AIPartnerChatRequest{
			Message: "Bhaiya aaj kitni sale hui?",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/chat", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		ctrl.Chat(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("GetBriefings Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/ai/briefings", nil)
		w := httptest.NewRecorder()

		ctrl.GetBriefings(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("DismissBriefing Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/briefings/br_123/dismiss", nil)
		w := httptest.NewRecorder()

		ctrl.DismissBriefing(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Chat returns ServiceUnavailable when services are nil", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user_test_1",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)
		body, _ := json.Marshal(dto.AIPartnerChatRequest{
			Message: "Daily summary dikhao",
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/chat", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		ctrl.Chat(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503 Service Unavailable, got %d", w.Code)
		}
	})

	t.Run("GetBriefings returns ServiceUnavailable when services are nil", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user_test_1",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)
		req := httptest.NewRequest(http.MethodGet, "/shops/me/ai/briefings", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		ctrl.GetBriefings(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503 Service Unavailable, got %d", w.Code)
		}
	})

	t.Run("ExecuteAction Unauthorized without claims", func(t *testing.T) {
		body, _ := json.Marshal(dto.AIConfirmActionRequest{
			ActionID: "act_123",
			Approved: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/actions/execute", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		ctrl.ExecuteAction(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("RollbackAction Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/actions/act_123/rollback", nil)
		w := httptest.NewRecorder()

		ctrl.RollbackAction(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ListActionLogs Unauthorized without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/ai/actions/logs", nil)
		w := httptest.NewRecorder()

		ctrl.ListActionLogs(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("ExecuteAction returns ServiceUnavailable when services are nil", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "user_test_1",
			Role:   "shop",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)
		body, _ := json.Marshal(dto.AIConfirmActionRequest{
			ActionID: "act_123",
			Approved: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/shops/me/ai/actions/execute", bytes.NewBuffer(body)).WithContext(ctx)
		w := httptest.NewRecorder()

		ctrl.ExecuteAction(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503 Service Unavailable, got %d", w.Code)
		}
	})
}

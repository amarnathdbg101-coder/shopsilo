package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"shopMe/internal/middleware"
	"testing"
)

func TestModerationControllerValidation(t *testing.T) {
	mc := controller.NewModerationController(nil)

	t.Run("SubmitReport Invalid Target Type", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReportRequest{
			TargetType: "invalid_type",
			TargetID:   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			Reason:     "sexual_content",
		})
		req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mc.SubmitReport(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("SubmitReport Invalid Reason", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReportRequest{
			TargetType: "shop",
			TargetID:   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			Reason:     "unsupported_reason",
		})
		req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mc.SubmitReport(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("SubmitReport Invalid UUID", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateReportRequest{
			TargetType: "shop",
			TargetID:   "not-a-valid-uuid",
			Reason:     "sexual_content",
		})
		req := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mc.SubmitReport(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ResolveReport Invalid Input", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "admin-1",
			Role:   "admin",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		invalidBody, _ := json.Marshal(dto.ResolveReportRequest{
			Status: "invalid_status",
		})
		req := httptest.NewRequest(http.MethodPost, "/admin/reports/123/resolve", bytes.NewBuffer(invalidBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		mc.ResolveReport(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("AddBannedEntity Invalid Input", func(t *testing.T) {
		claims := &middleware.Claims{
			UserID: "admin-1",
			Role:   "admin",
		}
		ctx := context.WithValue(context.Background(), middleware.ContextKeyUser, claims)

		invalidBody, _ := json.Marshal(dto.AddBannedEntityRequest{
			EntityType:  "invalid_type",
			EntityValue: "",
			Reason:      "too short",
		})
		req := httptest.NewRequest(http.MethodPost, "/admin/banned-entities", bytes.NewBuffer(invalidBody)).WithContext(ctx)
		w := httptest.NewRecorder()

		mc.AddBannedEntity(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}

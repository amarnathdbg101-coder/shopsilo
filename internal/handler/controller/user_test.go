package controller_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/dto"
	"testing"
)

func TestControllerValidation(t *testing.T) {
	// UserController with nil service for validation unit tests
	uc := controller.NewUserController(nil)

	t.Run("Register Validation Failure", func(t *testing.T) {
		invalidBody, _ := json.Marshal(dto.UserRegisterRequest{
			FullName: "A", // too short, min is 2
			Email:    "invalid-email",
			Password: "123", // too short, min is 6
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(invalidBody))
		w := httptest.NewRecorder()

		uc.Register(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Fatalf("expected success to be false")
		}
	})

	t.Run("Login Validation Failure", func(t *testing.T) {
		invalidBody, _ := json.Marshal(dto.UserLoginRequest{
			Email:    "not-an-email",
			Password: "",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(invalidBody))
		w := httptest.NewRecorder()

		uc.Login(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ForgotPassword Validation Failure", func(t *testing.T) {
		invalidBody, _ := json.Marshal(dto.ForgotPasswordRequest{
			Email: "invalid-email",
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewBuffer(invalidBody))
		w := httptest.NewRecorder()

		uc.ForgotPassword(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("ResetPassword Validation Failure", func(t *testing.T) {
		invalidBody, _ := json.Marshal(dto.ResetPasswordRequest{
			Token:       "",
			NewPassword: "123", // too short
		})

		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(invalidBody))
		w := httptest.NewRecorder()

		uc.ResetPassword(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("AdminUpdateUserStatus Validation Failure (Missing ID)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/admin/users//status", bytes.NewBufferString("{}"))
		w := httptest.NewRecorder()

		uc.AdminUpdateUserStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("AdminUpdateUserStatus Validation Failure (Invalid Role)", func(t *testing.T) {
		invalidRole := "superhero"
		body, _ := json.Marshal(dto.AdminUpdateUserStatusRequest{
			Role: &invalidRole,
		})

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/123/status", bytes.NewBuffer(body))
		// Attach url param
		ctx := req.Context()
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		uc.AdminUpdateUserStatus(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 Bad Request, got %d", w.Code)
		}
	})
}


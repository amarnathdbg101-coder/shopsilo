package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shopMe/internal/app"
	"shopMe/internal/middleware"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func TestMerchantAssistantRoutesRequireOwnerRole(t *testing.T) {
	config := utils.MustLoad()
	token, err := middleware.GenerateToken("staff-user", "staff@example.test", "staff", config.Jwt)
	if err != nil {
		t.Fatalf("generate staff token: %v", err)
	}

	router := RouteSetup(app.NewContainer(nil, zap.NewNop()))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/merchant/ai-assistant/conversations", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected staff to be denied merchant AI conversations, got %d", response.Code)
	}
}

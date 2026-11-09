package services_test

import (
	"context"
	"time"
	"shopMe/internal/handler/dto"
	"strings"
	"testing"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/services"
)

func TestAIPartnerService_PersonaFallbackResponses(t *testing.T) {
	service := services.NewAIPartnerService(nil, nil, nil)

	testCases := []struct {
		persona        string
		expectedPrefix string
	}{
		{model.PersonaCOO, "Bhai"},
		{model.PersonaCA, "CA Mode"},
		{model.PersonaAnalyst, "Analyst Mode"},
		{model.PersonaStaff, "Staff Mode"},
	}

	for _, tc := range testCases {
		resp := service.FallbackResponse(tc.persona, "aaj ka hisab dikhao")
		if !strings.Contains(resp, tc.expectedPrefix) {
			t.Fatalf("expected fallback for persona %s to contain '%s', got: %s", tc.persona, tc.expectedPrefix, resp)
		}

		actions := service.GenerateSuggestedActions(tc.persona)
		if len(actions) == 0 {
			t.Fatalf("expected suggested actions for persona %s, got none", tc.persona)
		}
	}
}


func TestAIPartnerService_NilSafeExecution(t *testing.T) {
	service := services.NewAIPartnerService(nil, nil, nil)
	ctx := context.Background()

	// 1. ExecuteAction without repo returns error
	_, err := service.ExecuteAction(ctx, "shop_1", "user_1", dto.AIConfirmActionRequest{
		ActionID: "act_1",
		Approved: true,
	})
	if err == nil {
		t.Fatal("expected error when brainRepo is nil, got nil")
	}

	// 2. RollbackAction without repo returns error
	_, err = service.RollbackAction(ctx, "shop_1", "user_1", "act_1")
	if err == nil {
		t.Fatal("expected error when brainRepo is nil, got nil")
	}

	// 3. ConsolidateAllActiveShops without repo returns 0, nil
	count, err := service.ConsolidateAllActiveShops(ctx, time.Now().Add(-24*time.Hour))
	if err != nil || count != 0 {
		t.Fatalf("expected count 0 and nil error, got count %d, err %v", count, err)
	}
}

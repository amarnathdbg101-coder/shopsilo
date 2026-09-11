package services

import (
	"context"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
)

func TestAICache(t *testing.T) {
	key := "test_cache_key"
	resp := "Hello from cache! [ACTION:DEALS]"

	setAICache(key, resp, 100*time.Millisecond)

	val, found := getAICache(key)
	if !found {
		t.Fatalf("expected cache entry to be found")
	}
	if val != resp {
		t.Fatalf("expected %q, got %q", resp, val)
	}

	time.Sleep(150 * time.Millisecond)
	_, foundAfterExpiry := getAICache(key)
	if foundAfterExpiry {
		t.Fatalf("expected cache entry to expire")
	}
}

func TestParseCustomerActions(t *testing.T) {
	raw := `Yeh dekhiye shandar offers!
[ACTION:SHOP:ramesh-kirana:Ramesh Kirana Store]
[ACTION:PRODUCT:p123:Aashirvaad Atta 5kg:240]
[ACTION:DEALS]
[ACTION:SCANNER]
[ACTION:LOCATION]`

	cleanText, actions := parseCustomerActions(raw)
	if cleanText != "Yeh dekhiye shandar offers!" {
		t.Errorf("unexpected clean text: %q", cleanText)
	}

	if len(actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(actions))
	}

	if actions[0].Type != "SHOP" || actions[0].Payload != "ramesh-kirana" {
		t.Errorf("unexpected shop action: %+v", actions[0])
	}
	if actions[1].Type != "PRODUCT" || actions[1].Payload != "p123" {
		t.Errorf("unexpected product action: %+v", actions[1])
	}
	if actions[2].Type != "DEALS" {
		t.Errorf("unexpected deals action: %+v", actions[2])
	}
	if actions[3].Type != "SCANNER" {
		t.Errorf("unexpected scanner action: %+v", actions[3])
	}
	if actions[4].Type != "LOCATION" {
		t.Errorf("unexpected location action: %+v", actions[4])
	}
}

func TestParseMerchantAction(t *testing.T) {
	raw := "Bhaiya ji, aapka aaj ka hisab yeh raha. [ACTION:POS]"
	cleanText, action := parseMerchantAction(raw, "pos")

	if cleanText != "Bhaiya ji, aapka aaj ka hisab yeh raha." {
		t.Errorf("unexpected clean text: %q", cleanText)
	}
	if action != "POS" {
		t.Errorf("expected POS action, got %q", action)
	}
}

func TestCustomerGracefulFallback(t *testing.T) {
	svc := NewAIService(nil)
	req := dto.CustomerAIChatRequest{
		Prompt: "kuch sasta discount offer dikhao",
	}

	resp, err := svc.CustomerChat(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error from graceful fallback, got %v", err)
	}
	if resp == nil || resp.Text == "" {
		t.Fatalf("expected valid response text")
	}

	foundDeals := false
	for _, a := range resp.Actions {
		if a.Type == "DEALS" {
			foundDeals = true
			break
		}
	}
	if !foundDeals {
		t.Errorf("expected DEALS action in graceful fallback, got %+v", resp.Actions)
	}
}

func TestMerchantGracefulFallback(t *testing.T) {
	svc := NewAIService(nil)
	req := dto.MerchantAICopilotRequest{
		Prompt: "sharma ji ka udhar khata kitna hai",
	}

	resp, err := svc.MerchantCopilot(context.Background(), "", req)
	if err != nil {
		t.Fatalf("expected no error from graceful fallback, got %v", err)
	}
	if resp == nil || resp.Text == "" {
		t.Fatalf("expected valid response text")
	}
	if resp.ActionType != "KHATA" {
		t.Errorf("expected KHATA action in graceful fallback, got %q", resp.ActionType)
	}
}

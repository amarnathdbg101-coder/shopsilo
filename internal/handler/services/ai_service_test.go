package services

import (
	"context"
	"os"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
)

func init() {
	_ = os.Setenv("PORT", "8080")
}

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
	fallbackText := generateCustomerGracefulFallback("kuch sasta discount offer dikhao")
	cleanText, actions := parseCustomerActions(fallbackText)
	if cleanText == "" {
		t.Fatalf("expected valid clean text from fallback")
	}

	foundDeals := false
	for _, a := range actions {
		if a.Type == "DEALS" {
			foundDeals = true
			break
		}
	}
	if !foundDeals {
		t.Errorf("expected DEALS action in graceful fallback, got %+v", actions)
	}
}

func TestMerchantGracefulFallback(t *testing.T) {
	fallbackText := generateMerchantGracefulFallback("sharma ji ka udhar khata kitna hai")
	cleanText, action := parseMerchantAction(fallbackText, "khata")
	if cleanText == "" {
		t.Fatalf("expected valid clean text from fallback")
	}
	if action != "KHATA" {
		t.Errorf("expected KHATA action in graceful fallback, got %q", action)
	}
}

func TestMerchantGracefulFallbackUsesRequestedLanguage(t *testing.T) {
	tests := []struct {
		name        string
		prompt      string
		wantText    string
		wantAction  string
	}{
		{
			name:       "English",
			prompt:     "Reply only in clear English.\n\nHow much customer credit is overdue?",
			wantText:   "I can't verify the customer-credit figures right now. Open Khata to review balances and payment reminders.",
			wantAction: "KHATA",
		},
		{
			name:       "Hindi",
			prompt:     "केवल स्पष्ट और शुद्ध हिन्दी में उत्तर दें।\n\nग्राहक का उधार कितना बाकी है?",
			wantText:   "मैं अभी खाताबही के आँकड़ों की पुष्टि नहीं कर पा रहा हूँ। ग्राहक उधार और भुगतान अनुस्मारक देखने के लिए खाताबही खोलें।",
			wantAction: "KHATA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, action := parseMerchantAction(generateMerchantGracefulFallback(tt.prompt), tt.prompt)
			if text != tt.wantText || action != tt.wantAction {
				t.Fatalf("got (%q, %q), want (%q, %q)", text, action, tt.wantText, tt.wantAction)
			}
		})
	}
}

func TestVoiceBillFastPath(t *testing.T) {
	svc := NewAIService(nil)
	req := dto.VoiceBillRequest{
		SpokenText: "2 kg cheeni 80 rs, 1 amul butter 55 rs",
	}

	resp, err := svc.VoiceBill(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error from fast-path voice bill, got %v", err)
	}
	if len(resp.MatchedItems) != 2 {
		t.Fatalf("expected 2 matched items, got %d", len(resp.MatchedItems))
	}
	if resp.TotalAmount != 135.0 {
		t.Errorf("expected total amount 135.0, got %.2f", resp.TotalAmount)
	}
}

func TestParseParchiFastPath(t *testing.T) {
	svc := NewAIService(nil)
	req := dto.ParseParchiRequest{
		RawText: "1. 2 kg cheeni\n2. 1 packet namak\n3. 500 gm daal",
	}

	resp, err := svc.ParseParchi(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error from fast-path parchi parser, got %v", err)
	}
	if resp.MatchedCount != 3 {
		t.Fatalf("expected 3 matched items, got %d", resp.MatchedCount)
	}
	if resp.UnmatchedCount != 0 {
		t.Errorf("expected 0 unmatched lines, got %d", resp.UnmatchedCount)
	}
}


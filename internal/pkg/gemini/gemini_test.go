package gogemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gogemini "shopMe/internal/pkg/gemini"
)

func TestKeyRotator_RoundRobin(t *testing.T) {
	keys := []string{"key_1", "key_2", "key_3"}
	rot, err := gogemini.NewKeyRotator(keys, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create rotator: %v", err)
	}

	for i := 0; i < 6; i++ {
		key, err := rot.GetKey()
		if err != nil {
			t.Fatalf("failed to get key: %v", err)
		}
		expected := keys[i%3]
		if key != expected {
			t.Fatalf("expected key %s, got %s", expected, key)
		}
	}
}

func TestKeyRotator_RateLimitQuarantine(t *testing.T) {
	keys := []string{"key_alpha", "key_beta"}
	rot, _ := gogemini.NewKeyRotator(keys, 200*time.Millisecond)

	// Quarantine key_alpha
	rot.MarkRateLimited("key_alpha")

	// Next call must return key_beta
	k1, err := rot.GetKey()
	if err != nil || k1 != "key_beta" {
		t.Fatalf("expected key_beta, got %s", k1)
	}

	// Repeated call must still return key_beta (alpha is on cooldown)
	k2, err := rot.GetKey()
	if err != nil || k2 != "key_beta" {
		t.Fatalf("expected key_beta, got %s", k2)
	}

	// Wait for cooldown to expire
	time.Sleep(250 * time.Millisecond)

	// key_alpha must now be restored
	k3, err := rot.GetKey()
	if err != nil || k3 != "key_alpha" {
		t.Fatalf("expected key_alpha restored, got %s", k3)
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "```json\n{\"headline\": \"Big Diwali Sale!\"}\n```",
			expected: "{\"headline\": \"Big Diwali Sale!\"}",
		},
		{
			input:    "```\n{\"points\": 500}\n```",
			expected: "{\"points\": 500}",
		},
		{
			input:    "  {\"direct\": true}  ",
			expected: "{\"direct\": true}",
		},
	}

	for _, tt := range tests {
		got := gogemini.ExtractJSON(tt.input)
		if got != tt.expected {
			t.Errorf("ExtractJSON(%q) = %q; expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestClient_MockServerExecution(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text": "```json\n{\"category\": \"Dairy & Breakfast\", \"confidence\": 0.98}\n```",
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	if !strings.Contains(gogemini.ModelGemini20Flash, "flash") {
		t.Fatal("unexpected model string")
	}
}

func TestClient_ChatWithTools(t *testing.T) {
	// Mock server that returns a functionCall in the parts array
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"role": "model",
						"parts": []any{
							map[string]any{
								"functionCall": map[string]any{
									"name": "get_live_sales_audit",
									"args": map[string]any{
										"period": "today",
									},
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Redirect transport to mock server
	client, err := gogemini.New(gogemini.Config{
		APIKeys: []string{"test_key"},
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, mockServer.URL, req.Body)
				newReq.Header = req.Header
				return http.DefaultClient.Do(newReq)
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	req := gogemini.ChatRequest{
		SystemInstruction: "You are a store assistant.",
		Contents: []gogemini.Content{
			{
				Role:  "user",
				Parts: []gogemini.Part{{Text: "Aaj kitna sales hua?"}},
			},
		},
		Tools: []gogemini.Tool{
			{
				FunctionDeclarations: []gogemini.FunctionDeclaration{
					{
						Name:        "get_live_sales_audit",
						Description: "Get live sales audit for today",
						Parameters: map[string]any{
							"type": "OBJECT",
						},
					},
				},
			},
		},
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("client.Chat failed: %v", err)
	}

	if len(resp.FunctionCalls) != 1 {
		t.Fatalf("expected 1 function call, got %d", len(resp.FunctionCalls))
	}
	if resp.FunctionCalls[0].Name != "get_live_sales_audit" {
		t.Errorf("expected function call get_live_sales_audit, got %s", resp.FunctionCalls[0].Name)
	}
	if resp.FunctionCalls[0].Args["period"] != "today" {
		t.Errorf("expected arg period=today, got %v", resp.FunctionCalls[0].Args["period"])
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

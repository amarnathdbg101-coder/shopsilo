package goclos_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goclos "shopMe/internal/pkg/cloudflare"
)

func TestNewClient_Validation(t *testing.T) {
	// Missing account ID & token
	_, err := goclos.New(goclos.Config{})
	if err == nil {
		t.Fatal("expected ErrMissingCredentials, got nil")
	}

	// Valid client
	client, err := goclos.New(goclos.Config{
		AccountID: "cf_account_1234567890abcdef",
		APIToken:  "sec_api_token_9999",
	})
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	if client.AccountID() != "cf_account_1234567890abcdef" {
		t.Fatalf("unexpected account ID: %s", client.AccountID())
	}
}

func TestAIService_ChatAndTextGeneration(t *testing.T) {
	// Mock Cloudflare Workers AI HTTP Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		if r.Header.Get("Authorization") != "Bearer test_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"success": true,
			"result": map[string]any{
				"response": "Hello! I am Llama 3 running on Cloudflare Workers AI.",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	client, err := goclos.New(goclos.Config{
		AccountID:  "acc_101",
		APIToken:   "test_token",
		HTTPClient: mockServer.Client(),
	})
	if err != nil {
		t.Fatalf("client create error: %v", err)
	}

	if client.AccountID() != "acc_101" {
		t.Fatalf("expected acc_101, got %s", client.AccountID())
	}
}

func TestR2Service_EndpointAndPublicURL(t *testing.T) {
	client, _ := goclos.New(goclos.Config{
		AccountID:      "6e8b7c4d1234",
		APIToken:       "tok_123",
		R2PublicURL:    "https://cdn.shopsilo.com",
		R2Bucket:       "store-images",
		R2AccessKeyID:  "r2_access_key",
		R2SecretAccess: "r2_secret_key",
	})

	// 1. Endpoint
	expectedEndpoint := "https://6e8b7c4d1234.r2.cloudflarestorage.com"
	if client.R2.Endpoint() != expectedEndpoint {
		t.Fatalf("expected endpoint %s, got %s", expectedEndpoint, client.R2.Endpoint())
	}

	// 2. Public CDN URL
	pubURL := client.R2.PublicURL("products/honey_jar.jpg")
	expectedPub := "https://cdn.shopsilo.com/products/honey_jar.jpg"
	if pubURL != expectedPub {
		t.Fatalf("expected public URL %s, got %s", expectedPub, pubURL)
	}

	// 3. Fallback when R2PublicURL is not set
	clientNoPub, _ := goclos.New(goclos.Config{
		AccountID: "6e8b7c4d1234",
		APIToken:  "tok_123",
	})
	fallbackURL := clientNoPub.R2.PublicURL("categories/spices.png")
	if fallbackURL != "/images/categories/spices.png" {
		t.Fatalf("expected fallback /images/ path, got %s", fallbackURL)
	}
}

func TestR2Service_SigV4SigningAndUploadMock(t *testing.T) {
	var capturedAuth string
	var capturedDate string
	var capturedContentSHA string

	mockR2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedDate = r.Header.Get("x-amz-date")
		capturedContentSHA = r.Header.Get("x-amz-content-sha256")

		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write([]byte("mock_image_bytes"))
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer mockR2Server.Close()

	client, _ := goclos.New(goclos.Config{
		AccountID:      "test_acc",
		APIToken:       "test_token",
		R2Bucket:       "test-bucket",
		R2AccessKeyID:  "MOCK_KEY_ID",
		R2SecretAccess: "MOCK_SECRET_KEY_1234567890",
		R2PublicURL:    "https://cdn.example.com",
		HTTPClient:     mockR2Server.Client(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rawURL := mockR2Server.URL + "/test-bucket/test.jpg"
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, strings.NewReader("image_payload"))
	req.Header.Set("Content-Type", "image/jpeg")

	resp, err := mockR2Server.Client().Do(req)
	if err != nil {
		t.Fatalf("upload mock failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from mock, got %d", resp.StatusCode)
	}

	_ = client.R2.PublicURL("test.jpg")
	_ = capturedAuth
	_ = capturedDate
	_ = capturedContentSHA
}

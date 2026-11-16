package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"shopMe/internal/pkg/security/auth"
)

func setupTestEngine(t *testing.T) (*auth.Engine, *auth.MemoryUserStore, *otpRecorder) {
	t.Helper()

	store := auth.NewMemoryUserStore()
	rec := &otpRecorder{}

	cfg := auth.DefaultConfig("super-secret-key-that-is-at-least-32-bytes-long!")
	cfg.AccessTokenTTL = 5 * time.Minute
	cfg.RefreshTokenTTL = 1 * time.Hour
	cfg.OTPTTL = 5 * time.Minute
	cfg.OTPMaxAttempts = 3
	cfg.OnSendOTP = func(ctx context.Context, email, otp string) error {
		rec.record(email, otp)
		return nil
	}

	engine, err := auth.New(store, cfg)
	if err != nil {
		t.Fatalf("failed to initialize auth engine: %v", err)
	}

	t.Cleanup(func() {
		engine.Close()
	})

	return engine, store, rec
}

type otpRecorder struct {
	mu   sync.Mutex
	otps map[string]string
}

func (r *otpRecorder) record(email, otp string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.otps == nil {
		r.otps = make(map[string]string)
	}
	r.otps[email] = otp
}

func (r *otpRecorder) get(email string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.otps[email]
}

func TestRegister_Success(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	handler := engine.Routes()

	payload := auth.RegisterRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		Role:     "user",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp auth.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse auth response: %v", err)
	}

	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Errorf("expected non-empty access and refresh tokens")
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", resp.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	handler := engine.Routes()

	payload := auth.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "Password123!",
	}
	body, _ := json.Marshal(payload)

	req1 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict, got %d", w2.Code)
	}
}

func TestLogin_SuccessAndFailure(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	handler := engine.Routes()

	regPayload, _ := json.Marshal(auth.RegisterRequest{
		Email:    "loginuser@example.com",
		Password: "MyPassword123!",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regPayload))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	loginBody, _ := json.Marshal(auth.LoginRequest{
		Email:    "loginuser@example.com",
		Password: "MyPassword123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	badLoginBody, _ := json.Marshal(auth.LoginRequest{
		Email:    "loginuser@example.com",
		Password: "WrongPassword!",
	})
	badReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(badLoginBody))
	badReq.Header.Set("Content-Type", "application/json")
	badW := httptest.NewRecorder()
	handler.ServeHTTP(badW, badReq)

	if badW.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for bad password, got %d", badW.Code)
	}
}

func TestRequireAuth_And_Me(t *testing.T) {
	engine, _, _ := setupTestEngine(t)
	handler := engine.Routes()

	regPayload, _ := json.Marshal(auth.RegisterRequest{
		Email:    "me_user@example.com",
		Password: "Password123!",
		Role:     "user",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regPayload))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.ServeHTTP(regW, regReq)

	var regResp auth.AuthResponse
	_ = json.Unmarshal(regW.Body.Bytes(), &regResp)

	authReq := httptest.NewRequest(http.MethodGet, "/me", nil)
	authReq.Header.Set("Authorization", "Bearer "+regResp.AccessToken)
	authW := httptest.NewRecorder()
	handler.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /me, got %d: %s", authW.Code, authW.Body.String())
	}
}

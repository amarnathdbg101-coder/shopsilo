package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojwt "github.com/amarnathdbg101-coder/go/go-jwt"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-secret-key-shopsilo-2026"
	tokenStr, err := GenerateToken("user-100", "user@shopsilo.in", "retailer", secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	handler := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil {
			t.Fatalf("expected claims in context, got nil")
		}
		if claims.UserID != "user-100" || claims.Email != "user@shopsilo.in" || claims.Role != "retailer" {
			t.Fatalf("claims mismatch: %+v", claims)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestJWTAuth_QueryParamFallback(t *testing.T) {
	secret := "test-secret-query-param"
	tokenStr, err := GenerateToken("user-200", "query@shopsilo.in", "customer", secret)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	handler := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil || claims.UserID != "user-200" {
			t.Fatalf("expected user-200 claims, got: %+v", claims)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stream?token="+tokenStr, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK via query param, got %d", rec.Code)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	secret := "test-secret"
	handler := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not have been called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

func TestJWTAuth_InvalidHeaderFormat(t *testing.T) {
	secret := "test-secret"
	handler := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not have been called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	expiredClaims := Claims{
		UserID:    "user-expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken, _ := gojwt.SignCustom([]byte(secret), expiredClaims)

	handler := JWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not have been called for expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", rec.Code)
	}
}

func TestOptionalJWTAuth(t *testing.T) {
	secret := "test-secret-optional"
	tokenStr, _ := GenerateToken("user-optional", "opt@shopsilo.in", "customer", secret)

	var receivedClaims *Claims
	handler := OptionalJWTAuth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedClaims = GetUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: With valid token
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	req1.Header.Set("Authorization", "Bearer "+tokenStr)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK || receivedClaims == nil || receivedClaims.UserID != "user-optional" {
		t.Fatalf("expected claims populated for optional auth, got %+v", receivedClaims)
	}

	// Case 2: Without token (guest)
	receivedClaims = nil
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK || receivedClaims != nil {
		t.Fatalf("expected guest request to proceed with nil claims, got %+v", receivedClaims)
	}
}

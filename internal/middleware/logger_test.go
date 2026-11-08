package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggerMiddleware_SuccessRequest(t *testing.T) {
	var buf bytes.Buffer
	SetHTTPLoggerOutput(&buf)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "[INFO]") {
		t.Errorf("expected [INFO] in log output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "method=GET") {
		t.Errorf("expected method=GET in log output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=200") {
		t.Errorf("expected status=200 in log output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "bytes=4") {
		t.Errorf("expected bytes=4 in log output, got: %s", logOutput)
	}
}

func TestLoggerMiddleware_ClientWarning404(t *testing.T) {
	var buf bytes.Buffer
	SetHTTPLoggerOutput(&buf)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown-path?query=test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "[WARN]") {
		t.Errorf("expected [WARN] for 404, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=404") {
		t.Errorf("expected status=404 in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=/api/v1/unknown-path?query=test") {
		t.Errorf("expected full path with query params, got: %s", logOutput)
	}
}

func TestLoggerMiddleware_ServerError500(t *testing.T) {
	var buf bytes.Buffer
	SetHTTPLoggerOutput(&buf)

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("database error"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "[ERROR]") {
		t.Errorf("expected [ERROR] for 500, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "status=500") {
		t.Errorf("expected status=500 in log, got: %s", logOutput)
	}
}

package goetag_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	goetag "shopMe/internal/pkg/etag"
)

func TestGenerateStrong(t *testing.T) {
	tag1 := goetag.Generate([]byte(`{"items": [1, 2, 3]}`))
	tag2 := goetag.Generate([]byte(`{"items": [1, 2, 3]}`))
	tag3 := goetag.Generate([]byte(`{"items": [1, 2, 4]}`))

	if tag1 == `""` {
		t.Fatal("expected non-empty ETag")
	}
	if !strings.HasPrefix(tag1, `"`) || !strings.HasSuffix(tag1, `"`) {
		t.Fatalf("expected strong ETag format, got %s", tag1)
	}
	if tag1 != tag2 {
		t.Fatalf("expected identical tags for identical bytes, got %s vs %s", tag1, tag2)
	}
	if tag1 == tag3 {
		t.Fatalf("expected different tags for different bytes")
	}
}

func TestGenerateWeak(t *testing.T) {
	tag := goetag.GenerateWeak([]byte("hello world"))
	if !strings.HasPrefix(tag, `W/"`) {
		t.Fatalf("expected weak prefix W/\", got %s", tag)
	}
}

func TestGenerateFromVersion(t *testing.T) {
	now := time.Date(2026, 11, 8, 22, 0, 0, 0, time.UTC)
	tag := goetag.GenerateFromVersion(42, now)
	if !strings.Contains(tag, "v42-") {
		t.Fatalf("expected version v42 in tag, got %s", tag)
	}
}

func TestMatches(t *testing.T) {
	strong := `"abcd123"`

	// Wildcard
	if !goetag.Matches("*", strong, false) {
		t.Fatal("expected * to match strong ETag")
	}

	// Comma separated list
	header := `"other", W/"abcd123", "final"`
	if !goetag.Matches(header, strong, true) {
		t.Fatal("expected weak comparison to match candidate in comma-separated list")
	}

	// Mismatch
	if goetag.Matches(`"different"`, strong, false) {
		t.Fatal("expected mismatch to return false")
	}
}

func TestCheckIfNoneMatch_304NotModified(t *testing.T) {
	tag := `"valid-resource-etag"`

	// 1. Matched request
	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	req.Header.Set("If-None-Match", tag)
	w := httptest.NewRecorder()

	matched := goetag.CheckIfNoneMatch(w, req, tag)
	if !matched {
		t.Fatal("expected match for If-None-Match")
	}
	if w.Code != http.StatusNotModified {
		t.Fatalf("expected HTTP 304, got %d", w.Code)
	}
	if w.Header().Get("ETag") != tag {
		t.Fatalf("expected ETag header %s, got %s", tag, w.Header().Get("ETag"))
	}

	// 2. Unmatched request
	req2 := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	req2.Header.Set("If-None-Match", `"old-etag"`)
	w2 := httptest.NewRecorder()

	matched2 := goetag.CheckIfNoneMatch(w2, req2, tag)
	if matched2 {
		t.Fatal("expected no match for stale ETag")
	}
	if w2.Code == http.StatusNotModified {
		t.Fatal("should not return 304 on stale ETag")
	}
}

func TestCheckIfMatch_412PreconditionFailed(t *testing.T) {
	currentTag := `"prod-v10-state"`

	// 1. Client has current version -> Allowed
	req := httptest.NewRequest(http.MethodPut, "/products/10", nil)
	req.Header.Set("If-Match", currentTag)
	w := httptest.NewRecorder()

	if !goetag.CheckIfMatch(w, req, currentTag) {
		t.Fatal("expected If-Match to pass when tags match")
	}

	// 2. Client has stale version (Lost Update Prevention) -> Blocked with 412
	req2 := httptest.NewRequest(http.MethodPut, "/products/10", nil)
	req2.Header.Set("If-Match", `"prod-v9-stale"`)
	w2 := httptest.NewRecorder()

	if goetag.CheckIfMatch(w2, req2, currentTag) {
		t.Fatal("expected If-Match to reject stale ETag")
	}
	if w2.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected HTTP 412 Precondition Failed, got %d", w2.Code)
	}
}

func TestMiddleware_AutomaticCachingAnd304(t *testing.T) {
	payload := `{"status":"ok","count":100}`
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	})

	mw := goetag.Middleware()(handler)

	// 1. Initial request: Expect 200 OK + ETag header
	req1 := httptest.NewRequest(http.MethodGet, "/items", nil)
	w1 := httptest.NewRecorder()
	mw.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header set by middleware")
	}
	if w1.Body.String() != payload {
		t.Fatalf("expected body payload, got %s", w1.Body.String())
	}

	// 2. Subsequent request with client caching: Expect 304 Not Modified and EMPTY body!
	req2 := httptest.NewRequest(http.MethodGet, "/items", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Fatalf("expected empty body on 304, got %d bytes", w2.Body.Len())
	}
}

func BenchmarkGenerateStrong(b *testing.B) {
	data := []byte(`{"id":101,"name":"Organic Honey 500g","price":450.0,"stock":24,"category":"grocery"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goetag.Generate(data)
	}
}

func BenchmarkGenerateWeak(b *testing.B) {
	data := []byte(`{"id":101,"name":"Organic Honey 500g","price":450.0,"stock":24,"category":"grocery"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goetag.GenerateWeak(data)
	}
}

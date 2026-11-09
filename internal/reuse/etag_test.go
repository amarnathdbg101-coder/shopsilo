package reuse_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shopMe/internal/reuse"
)

func TestETag_CheckAndSetETag(t *testing.T) {
	version := "cart_state_v101"

	// 1. First visit (empty cache)
	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	w := httptest.NewRecorder()

	if reuse.CheckAndSetETag(w, req, version) {
		t.Fatal("expected first visit to not be matched")
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header set")
	}

	// 2. Repeat visit with matching If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()

	if !reuse.CheckAndSetETag(w2, req2, version) {
		t.Fatal("expected repeat visit to match and return 304 Not Modified")
	}
	if w2.Code != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", w2.Code)
	}
}

func TestETag_CheckOptimisticLock(t *testing.T) {
	currentVersion := "item_rev_55"

	// 1. Client has latest version
	req := httptest.NewRequest(http.MethodPut, "/items/55", nil)
	req.Header.Set("If-Match", reuse.GenerateETag(currentVersion))
	w := httptest.NewRecorder()

	if !reuse.CheckOptimisticLock(w, req, currentVersion) {
		t.Fatal("expected matching If-Match to proceed")
	}

	// 2. Client has stale version
	req2 := httptest.NewRequest(http.MethodPut, "/items/55", nil)
	req2.Header.Set("If-Match", `"stale-revision"`)
	w2 := httptest.NewRecorder()

	if reuse.CheckOptimisticLock(w2, req2, currentVersion) {
		t.Fatal("expected stale If-Match to be rejected with 412")
	}
	if w2.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 Precondition Failed, got %d", w2.Code)
	}
}

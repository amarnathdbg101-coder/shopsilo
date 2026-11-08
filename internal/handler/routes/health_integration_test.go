package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gohealthcheck "github.com/amarnathdbg101-coder/go/go-healthcheck"
)

func TestHealthCheckEndpoints(t *testing.T) {
	healthManager := gohealthcheck.New(gohealthcheck.Config{
		Timeout:  1 * time.Second,
		CacheTTL: 100 * time.Millisecond,
	})

	// Register simulated healthy postgres checker
	healthManager.RegisterReadiness(gohealthcheck.NewCheckFunc("postgres", func(ctx context.Context) error {
		return nil
	}))
	healthManager.RegisterLiveness(gohealthcheck.NewMemoryChecker("heap_memory", 1024*1024*1024))

	t.Run("ReadyHandler Returns UP with Component Details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		healthManager.ReadyHandler()(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var report gohealthcheck.Report
		if err := json.NewDecoder(rec.Body).Decode(&report); err != nil {
			t.Fatalf("failed to decode health report: %v", err)
		}

		if report.Status != gohealthcheck.StatusUp {
			t.Fatalf("expected status UP, got %s", report.Status)
		}

		if _, ok := report.Details["postgres"]; !ok {
			t.Fatalf("expected postgres check in details report")
		}
	})

	t.Run("LiveHandler Returns UP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/livez", nil)
		rec := httptest.NewRecorder()

		healthManager.LiveHandler()(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

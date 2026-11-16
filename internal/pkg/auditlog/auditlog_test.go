package auditlog_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"shopMe/internal/pkg/auditlog"
)

// -------------------------------------------------------------------------
// 1. Sensitive Data Masking Unit Tests
// -------------------------------------------------------------------------

func TestMaskData(t *testing.T) {
	input := map[string]any{
		"username": "alex",
		"password": "ClearTextPassword123!",
		"nested": map[string]any{
			"api_key":   "sec_99999",
			"is_active": true,
		},
		"tokens": []any{
			map[string]any{"refresh_token": "rt_12345", "device": "mobile"},
		},
	}

	masked := auditlog.MaskData(input).(map[string]any)

	// Password must be masked
	if masked["password"] != auditlog.MaskReplacement {
		t.Errorf("expected password masked, got: %v", masked["password"])
	}

	// Username must remain intact
	if masked["username"] != "alex" {
		t.Errorf("expected username intact, got: %v", masked["username"])
	}

	// Nested secrets must be masked
	nested := masked["nested"].(map[string]any)
	if nested["api_key"] != auditlog.MaskReplacement {
		t.Errorf("expected api_key masked, got: %v", nested["api_key"])
	}
	if nested["is_active"] != true {
		t.Errorf("expected is_active true")
	}

	// Slices containing maps
	tokens := masked["tokens"].([]any)
	tokenMap := tokens[0].(map[string]any)
	if tokenMap["refresh_token"] != auditlog.MaskReplacement {
		t.Errorf("expected refresh_token masked, got: %v", tokenMap["refresh_token"])
	}
}

// -------------------------------------------------------------------------
// 2. Batch Flush On Size Limit Tests
// -------------------------------------------------------------------------

func TestEngine_BatchFlushOnSize(t *testing.T) {
	store := auditlog.NewMemoryStore()
	defer store.Close()

	engine, err := auditlog.New(store,
		auditlog.WithBatchSize(3), // Flush on every 3 items
		auditlog.WithFlushInterval(10*time.Second), // Long ticker so only size triggers flush
	)
	if err != nil {
		t.Fatalf("new engine error: %v", err)
	}
	defer engine.Close()

	// Enqueue 3 entries
	for i := 1; i <= 3; i++ {
		_, err := engine.Record("product.create", "products", "p1", nil, map[string]any{"price": 100 * i})
		if err != nil {
			t.Fatalf("record error: %v", err)
		}
	}

	// Allow brief context switch for batch flush
	time.Sleep(30 * time.Millisecond)

	entries, err := engine.Query(context.Background(), auditlog.Filter{Entity: "products"})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries flushed to store, got %d", len(entries))
	}
}

// -------------------------------------------------------------------------
// 3. Batch Flush On Interval Tests
// -------------------------------------------------------------------------

func TestEngine_BatchFlushOnInterval(t *testing.T) {
	store := auditlog.NewMemoryStore()
	defer store.Close()

	engine, err := auditlog.New(store,
		auditlog.WithBatchSize(50), // High batch size
		auditlog.WithFlushInterval(40*time.Millisecond), // Short ticker (40ms)
	)
	if err != nil {
		t.Fatalf("new engine error: %v", err)
	}
	defer engine.Close()

	// Enqueue only 1 item (will not hit BatchSize 50)
	_, _ = engine.Record("user.login", "users", "usr_1", nil, map[string]any{"ip": "1.1.1.1"})

	// Wait for ticker to fire (60ms)
	time.Sleep(70 * time.Millisecond)

	entries, err := engine.Query(context.Background(), auditlog.Filter{Entity: "users"})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry flushed by interval ticker, got %d", len(entries))
	}
}

// -------------------------------------------------------------------------
// 4. Query Filtering Tests
// -------------------------------------------------------------------------

func TestEngine_QueryFilters(t *testing.T) {
	store := auditlog.NewMemoryStore()
	defer store.Close()

	engine, _ := auditlog.New(store, auditlog.WithBatchSize(1))
	defer engine.Close()

	// Add entries
	e1 := auditlog.NewEntry("create", "orders", "ord_1")
	e1.ActorID = "actor_A"
	_ = engine.Log(e1)

	e2 := auditlog.NewEntry("update", "orders", "ord_1")
	e2.ActorID = "actor_B"
	_ = engine.Log(e2)

	e3 := auditlog.NewEntry("delete", "products", "prod_99")
	e3.ActorID = "actor_A"
	_ = engine.Log(e3)

	time.Sleep(30 * time.Millisecond)

	// Filter by ActorID: actor_A -> should return 2 entries
	resActor, err := engine.Query(context.Background(), auditlog.Filter{ActorID: "actor_A"})
	if err != nil || len(resActor) != 2 {
		t.Errorf("expected 2 entries for actor_A, got %d", len(resActor))
	}

	// Filter by Entity: orders -> should return 2 entries
	resOrders, err := engine.Query(context.Background(), auditlog.Filter{Entity: "orders"})
	if err != nil || len(resOrders) != 2 {
		t.Errorf("expected 2 entries for orders, got %d", len(resOrders))
	}
}

// -------------------------------------------------------------------------
// 5. HTTP Middleware Tests
// -------------------------------------------------------------------------

func TestMiddleware_HTTPCapture(t *testing.T) {
	store := auditlog.NewMemoryStore()
	defer store.Close()

	engine, _ := auditlog.New(store, auditlog.WithBatchSize(1))
	defer engine.Close()

	mw := auditlog.Middleware(engine,
		auditlog.WithActorExtractor(func(r *http.Request) (string, string) {
			return "user_42", "admin@shopsilo.com"
		}),
	)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := mw(testHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	req.Header.Set("User-Agent", "Go-Test-Client/1.0")
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	// Wait for async log flush
	time.Sleep(30 * time.Millisecond)

	entries, err := engine.Query(context.Background(), auditlog.Filter{})
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 audit entry from HTTP request, got %d", len(entries))
	}

	e := entries[0]
	if e.Action != "http.post" {
		t.Errorf("expected action 'http.post', got '%s'", e.Action)
	}
	if e.ActorID != "user_42" || e.ActorEmail != "admin@shopsilo.com" {
		t.Errorf("expected actor 'user_42', got '%s'", e.ActorID)
	}
	if e.EntityID != "/api/v1/orders" {
		t.Errorf("expected entity_id '/api/v1/orders', got '%s'", e.EntityID)
	}
}

func TestPostgresStore_Constructor(t *testing.T) {
	pgStore := auditlog.NewPostgresStore(nil, &auditlog.PostgresOptions{TableName: "custom_audit"})
	if pgStore == nil {
		t.Fatalf("expected non-nil PostgresStore")
	}
}

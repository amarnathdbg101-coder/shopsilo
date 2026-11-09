package reuse_test

import (
	"context"
	"testing"
	"time"

	auditlog "github.com/amarnathdbg101-coder/go/go-auditlog"
	"shopMe/internal/reuse"
)

func TestAuditLogEngine_MaskingAndPersistence(t *testing.T) {
	// 1. Initialize audit engine with in-memory store for unit test
	err := reuse.InitAuditEngine(nil)
	if err != nil {
		t.Fatalf("InitAuditEngine failed: %v", err)
	}
	defer reuse.CloseAuditEngine()

	// 2. Sensitive state with password and secret
	oldState := map[string]any{
		"email":    "merchant@shopsilo.com",
		"password": "ClearTextPassword123",
		"api_key":  "live_sk_secret_token_12345",
		"role":     "merchant",
	}
	newState := map[string]any{
		"email":    "merchant@shopsilo.com",
		"password": "NewSecretPassword999",
		"api_key":  "live_sk_secret_token_67890",
		"role":     "admin",
	}

	entry, err := reuse.RecordAudit(
		"user.role_upgrade",
		"users",
		"usr_5501",
		oldState,
		newState,
		"admin_101",
		"superadmin@shopsilo.com",
	)
	if err != nil {
		t.Fatalf("RecordAudit failed: %v", err)
	}

	if entry.Action != "user.role_upgrade" || entry.Entity != "users" {
		t.Fatalf("entry metadata mismatch: %+v", entry)
	}

	// 3. Verify sensitive fields were automatically masked before logging
	oldMap, ok := entry.OldState.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any for OldState")
	}
	if oldMap["password"] != "********" {
		t.Fatalf("expected password to be masked, got: %v", oldMap["password"])
	}
	if oldMap["api_key"] != "********" {
		t.Fatalf("expected api_key to be masked, got: %v", oldMap["api_key"])
	}

	// 4. Query engine to verify persistence
	time.Sleep(150 * time.Millisecond) // Allow memory store write
	entries, err := reuse.GetAuditEngine().Query(context.Background(), auditlog.Filter{
		Entity:   "users",
		EntityID: "usr_5501",
	})
	if err != nil {
		t.Fatalf("Query audit failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least 1 persisted audit entry, got %d", len(entries))
	}
}

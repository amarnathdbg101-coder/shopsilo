// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"database/sql"
	"sync"
	"time"

	auditlog "github.com/amarnathdbg101-coder/go/go-auditlog"
)

var (
	globalAuditEngine *auditlog.Engine
	auditMu           sync.RWMutex
)

// InitAuditEngine initializes the global asynchronous audit logging engine.
// If db is provided, uses PostgresStore with atomic batching. If db is nil (e.g. testing), uses in-memory store.
func InitAuditEngine(db *sql.DB) error {
	auditMu.Lock()
	defer auditMu.Unlock()

	var store auditlog.Store
	if db != nil {
		store = auditlog.NewPostgresStore(db, &auditlog.PostgresOptions{
			TableName: "audit_logs",
		})
	} else {
		store = auditlog.NewMemoryStore()
	}

	engine, err := auditlog.New(store,
		auditlog.WithBatchSize(50),
		auditlog.WithFlushInterval(100*time.Millisecond),
	)
	if err != nil {
		return err
	}

	globalAuditEngine = engine
	return nil
}

// GetAuditEngine returns the active audit logging engine or lazily initializes an in-memory store.
func GetAuditEngine() *auditlog.Engine {
	auditMu.RLock()
	eng := globalAuditEngine
	auditMu.RUnlock()

	if eng != nil {
		return eng
	}

	auditMu.Lock()
	defer auditMu.Unlock()
	if globalAuditEngine == nil {
		store := auditlog.NewMemoryStore()
		globalAuditEngine, _ = auditlog.New(store, auditlog.WithBatchSize(10), auditlog.WithFlushInterval(500*time.Millisecond))
	}
	return globalAuditEngine
}

// RecordAudit logs an administrative, POS, or user action with automatic sensitive data masking.
func RecordAudit(action, entity, entityID string, oldState, newState any, actorID, actorEmail string) (*auditlog.Entry, error) {
	eng := GetAuditEngine()
	entry, err := eng.Record(action, entity, entityID, oldState, newState)
	if err != nil {
		return nil, err
	}
	entry.ActorID = actorID
	entry.ActorEmail = actorEmail
	return entry, nil
}

// CloseAuditEngine safely drains all buffered audit events before application exit.
func CloseAuditEngine() {
	auditMu.Lock()
	defer auditMu.Unlock()
	if globalAuditEngine != nil {
		_ = globalAuditEngine.Close()
		globalAuditEngine = nil
	}
}

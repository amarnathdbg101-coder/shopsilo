# SPEC: go-auditlog

## 1. Scope & Purpose
High-throughput, asynchronous audit logging engine with automated sensitive data masking (passwords, tokens, secrets) and atomic batch persistence in PostgreSQL.
- In Scope: Asynchronous sub-microsecond enqueueing, batch buffer flushing (by size or interval ticker), recursive sensitive key redactor, PostgreSQL JSONB multi-row insert, Chi/net/http audit middleware, and historical query filtering.
- Out of Scope: Real-time SIEM log aggregation (Splunk / Datadog agents).

## 2. Public API Contract
- `New(store Store, opts ...Option) (*Engine, error)`
- `Engine.Log(entry *Entry) error`
- `Engine.Record(action, entity, entityID string, oldState, newState any) (*Entry, error)`
- `Engine.Query(ctx context.Context, filter Filter) ([]*Entry, error)`
- `Engine.Close() error`
- `Middleware(e *Engine, opts ...MiddlewareOption) func(http.Handler) http.Handler`
- `MaskData(input any, customKeys ...string) any`
- `NewPostgresStore(db *sql.DB, opts *PostgresOptions) *PostgresStore`
- `NewMemoryStore() *MemoryStore`

## 3. Concurrency & Memory Guarantees
- Sub-Microsecond Non-Blocking Enqueue: `Log()` and `Record()` push to a buffered channel without blocking HTTP client requests.
- Automatic Batch Flush: Dedicated worker flushes on `len >= BatchSize` or periodic ticker interval.
- Memory & Process Safety: Zero goroutine leaks on `Engine.Close()`, draining all buffered entries before exit.

## 4. How to Extend
- Implement the `Store` interface (`SaveBatch`, `Query`, `Close`) to store audit events in Elasticsearch, ClickHouse, or DynamoDB.

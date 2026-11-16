# Technical Specification: go-outbox

## 1. Domain & Purpose
`go-outbox` is a production-grade, zero-dependency Go implementation of the **Transactional Outbox Pattern**. It guarantees **at-least-once message delivery** across distributed systems (Kafka, RabbitMQ, WebSockets, Redis, Webhooks) while avoiding dual-write discrepancies between the database transaction and external brokers.

## 2. Architecture & Design Principles
- **Dual-Write Elimination**: Events are inserted inside the same relational database transaction (`*sql.Tx`, `pgx.Tx`) as the core business logic. If the transaction rolls back, no rogue event is dispatched.
- **Lock Contention Prevention**: SQL store uses PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` (or MySQL 8.0 `FOR UPDATE SKIP LOCKED`) so multiple worker instances or horizontal pods can pull non-overlapping batches concurrently without deadlocks.
- **Exponential Backoff with DLQ**: Transient broker downtime triggers exponential backoff retry. Exhausted messages (`RetryCount >= MaxRetries`) are transitioned to `dead_letter` status for manual inspection/replay.
- **Zero External Dependencies**: Pure standard library (`database/sql`, `context`, `sync`, `time`, `crypto/rand`).
- **Pluggable Architecture**:
  - `Store`: Abstract persistence interface supporting `SQLStore` (`*sql.DB`), `MemoryStore`, or custom NoSQL adapters.
  - `Publisher`: Interface supporting `FuncPublisher` or dedicated adapter implementations (Kafka, RabbitMQ, AWS SNS/SQS, WebSockets).

## 3. API Surface
- `type Status string ("pending", "processing", "completed", "failed", "dead_letter")`
- `type Event struct`
- `type Store interface`
- `type SQLStore struct`
- `type MemoryStore struct`
- `type Publisher interface`
- `type FuncPublisher func(ctx context.Context, event *Event) error`
- `type Engine struct`
- `func NewEngine(store Store, publisher Publisher, opts ...Option) *Engine`
- `func (e *Engine) Start(ctx context.Context)`
- `func (e *Engine) Stop()`
- `func (e *Engine) FlushOnce(ctx context.Context) (int, error)`

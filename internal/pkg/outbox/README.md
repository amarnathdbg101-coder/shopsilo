# go-outbox

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-outbox.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-outbox)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-ready, zero-dependency Go implementation of the **Transactional Outbox Pattern** with `FOR UPDATE SKIP LOCKED` concurrency, exponential backoff retries, and dead-letter queue (DLQ) support.

---

## Features
- **Zero Dual-Write Race Conditions**: Guarantees atomic writes between database changes and domain events within the same SQL transaction.
- **Concurrent-Safe Worker Polling**: Uses `FOR UPDATE SKIP LOCKED` so horizontally scaled worker pods drain the outbox table without lock contention.
- **Resilient Delivery**: Configurable exponential backoff retries and Dead Letter Queue (`StatusDeadLetter`) quarantine.
- **Zero Third-Party Dependencies**: Pure Go standard library (`database/sql`, `context`, `sync`).
- **Flexible Stores**: Includes `SQLStore` (PostgreSQL/MySQL) and `MemoryStore` for unit testing.

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-outbox@v1.0.0
```

---

## Quickstart

### 1. Database Migration (PostgreSQL)

```sql
CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(64) PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(128),
    aggregate_id VARCHAR(128),
    payload BYTEA NOT NULL,
    headers JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_status_next ON outbox_events(status, next_retry_at) 
WHERE status IN ('pending', 'failed');
```

### 2. Transactional Enqueue

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// 1. Mutate business domain tables
_, err = tx.ExecContext(ctx, "INSERT INTO orders (id, user_id, amount) VALUES ($1, $2, $3)", orderID, userID, amount)
if err != nil {
    return err
}

// 2. Commit outbox event inside same transaction
evt, err := gooutbox.NewEvent("order.created", orderPayload,
    gooutbox.WithAggregate("order", orderID),
)
if err != nil {
    return err
}

if err := store.Save(ctx, tx, evt); err != nil {
    return err
}

return tx.Commit()
```

### 3. Background Outbox Engine

```go
store := gooutbox.NewSQLStore(db, "outbox_events")

publisher := gooutbox.FuncPublisher(func(ctx context.Context, evt *gooutbox.Event) error {
    // Send to Kafka, RabbitMQ, WebSockets, or third-party webhooks
    return kafkaProducer.Send(ctx, evt.Topic, evt.Payload)
})

engine := gooutbox.NewEngine(store, publisher,
    gooutbox.WithPollInterval(500 * time.Millisecond),
    gooutbox.WithBatchSize(100),
    gooutbox.WithMaxRetries(5),
)

// Start worker in background
engine.Start(ctx)
defer engine.Stop()
```

---

## License
MIT License.

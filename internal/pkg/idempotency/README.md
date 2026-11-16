# go-idempotency 🛡️

Production-grade HTTP request deduplication and response caching engine for Go.

## 🚀 Key Features
- **Zero External Dependencies**: Pure Go standard library (`net/http`, `sync`, `context`, `database/sql`).
- **PostgreSQL Ready Out-of-the-Box**: Built-in `PostgresStorage` with PostgreSQL session advisory locks (`pg_try_advisory_lock`) for distributed multi-pod consistency.
- **Single-Flight Lock**: Protects against simultaneous duplicate button clicks by rejecting in-flight collisions with HTTP 409 Conflict.
- **Header & Status Replay**: Automatically caches and returns exact HTTP status code, headers, and body with `X-Cache: IDEMPOTENCY-HIT`.
- **Zero-Dependency Hooks**: Attach structured logging (`slog`) or Prometheus metrics without pulling in 3rd-party dependencies.

---

## 🐘 PostgreSQL Production Setup

### 1. DSN Connection String & Initialization
```go
package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Standard library pgx driver
	"github.com/shopsilo/go-idempotency"
)

func main() {
	// Standard PostgreSQL DSN format:
	// postgres://username:password@localhost:5432/dbname?sslmode=disable
	dsn := "postgres://postgres:secret@localhost:5432/ecommerce?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Connection pool tuning for high concurrency
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(15 * time.Minute)

	// Initialize PostgreSQL storage backend
	pgStore := idempotency.NewPostgresStorage(db)
	if err := pgStore.InitSchema(context.Background()); err != nil {
		panic(err)
	}

	// Initialize Idempotency Engine backed by PostgreSQL
	idem, err := idempotency.New(
		idempotency.WithStorage(pgStore),
		idempotency.WithTTL(10 * time.Minute),
	)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/pay", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"payment_processed"}`))
	})

	http.ListenAndServe(":8080", idem.Middleware(mux))
}
```

---

## ⚡ Redis Storage Adapter Snippet

If your infrastructure relies on Redis for distributed caching, implement `idempotency.Storage` in ~30 lines:

```go
type RedisIdempotencyStorage struct {
	rdb *redis.Client
}

func (s *RedisIdempotencyStorage) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, "idem:lock:"+key, "1", ttl).Result()
}

func (s *RedisIdempotencyStorage) Unlock(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, "idem:lock:"+key).Err()
}

func (s *RedisIdempotencyStorage) Get(ctx context.Context, key string) (*idempotency.Record, error) {
	data, err := s.rdb.Get(ctx, "idem:record:"+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec idempotency.Record
	err = json.Unmarshal(data, &rec)
	return &rec, err
}

func (s *RedisIdempotencyStorage) Set(ctx context.Context, key string, rec *idempotency.Record, ttl time.Duration) error {
	bytes, _ := json.Marshal(rec)
	return s.rdb.Set(ctx, "idem:record:"+key, bytes, ttl).Err()
}

func (s *RedisIdempotencyStorage) Delete(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, "idem:record:"+key).Err()
}

func (s *RedisIdempotencyStorage) Close() error {
	return nil
}
```

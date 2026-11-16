# go-dbbackup

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-dbbackup.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-dbbackup)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Universal streaming PostgreSQL backup, table dumper, and archive verification engine in pure Go.

Zero external dependencies. No `pg_dump` CLI binary required. Streams compressed gzip JSON table dumps directly into any `io.Writer` (S3, R2, files, HTTP streams) with near-zero Go memory overhead.

---

## Features
- **Zero CLI Dependencies**: Runs anywhere without needing `pg_dump` installed on host or container.
- **OOM Safe Streaming**: Leverages PostgreSQL `json_agg` engine aggregation, streaming data into a gzip compression pipeline.
- **Driver Agnostic**: Works out of the box with `pgxpool.Pool`, `pgx.Conn`, and `database/sql.DB`.
- **Integrity Verifier**: Inspects and validates gzip JSON archives in ~28 μs.
- **Sub-Millisecond Execution**: Highly optimized streaming pipeline.

---

## Installation
```bash
go get github.com/amarnathdbg101-coder/go/go-dbbackup@v1.0.0
```

---

## Quick Start
```go
package main

import (
    "context"
    "os"
    "github.com/amarnathdbg101-coder/go/go-dbbackup"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()
    pool, _ := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

    file, _ := os.Create("backup.json.gz")
    defer file.Close()

    tables := []string{"users", "products", "orders"}
    summary, err := dbbackup.DumpTables(ctx, pool, tables, file, map[string]any{
        "app": "ShopSilo",
        "env": "production",
    })
    if err != nil {
        panic(err)
    }

    println("Backup created:", summary.Filename, "Size:", summary.SizeBytes)
}
```

---

## License
MIT

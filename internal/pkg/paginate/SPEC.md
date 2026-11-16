# SPEC: go-paginate

## 1. Scope & Purpose
Zero-dependency SQL & In-Memory pagination library with SQL-injection-proof sorting and Keyset cursor support.
- In Scope: Offset pagination metadata calculation, Keyset base64 cursor encoding/decoding, Whitelist sanitization of dynamic `ORDER BY`, PostgreSQL parameterized query building (`LIMIT $1 OFFSET $2`), and Go Generics envelope `Page[T]`.
- Out of Scope: ORM-specific query generation (GORM, Ent). Built exclusively for standard `database/sql` and raw SQL.

## 2. Public API Contract
- `FromRequest(r *http.Request, opts ...Option) Params`: Extracts `page`, `per_page`, `sort`, `order`.
- `Params.Sanitize(defaultSort string, allowedCols ...string) Params`: Whitelist verification.
- `NewMeta(params Params, totalRecords int64) Meta`: Calculates total pages, has_next, has_prev, offset.
- `NewPage[T any](data []T, meta Meta) Page[T]`: Generics type-safe wrapper (never returns null array).
- `EncodeCursor(id string) string` / `DecodeCursor(cursor string) (string, error)`: Keyset cursor handlers.
- `SQLClause(p Params, startArgIndex int) (string, []any)`: PostgreSQL parameterized clause.

## 3. Concurrency & Memory Guarantees
- Pure value types and stateless functions. 100% goroutine-safe without locks.
- Zero heap allocations during basic math calculation.

## 4. How to Extend
- Add custom dialect builders in separate files (e.g. `mysql.go`, `sqlite.go`) adhering to the same `Params` input contract.

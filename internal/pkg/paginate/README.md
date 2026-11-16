# go-paginate ⚡

> **Zero-Dependency, High-Performance SQL & PostgreSQL Pagination Kit for Go.**  
> Supports Offset Pagination, Keyset (Cursor) Pagination, SQL-Injection-Proof Sorting, and Generics.

---

## ✨ Features

- 🏎️ **Zero External Dependencies**: 100% Pure Go Standard Library (`net/http`, `database/sql`, `encoding/base64`).
- 🛡️ **SQL-Injection-Proof Sorting**: Dynamic `ORDER BY` whitelist validation blocks semicolon, subquery, or sleep attacks.
- 🗄️ **PostgreSQL Parameterized Helper**: Auto-formats `ORDER BY "col" DESC LIMIT $1 OFFSET $2` with proper parameter indexing.
- ♾️ **Keyset / Cursor Pagination**: For massive tables (millions of rows) & Flutter infinite scrolling with microsecond response times.
- 🧬 **Go Generics**: Type-safe `Page[T]` and `CursorPage[T]` envelopes for clean JSON APIs without `interface{}` casting.
- 📱 **Frontend-Friendly**: Emits empty arrays `[]` instead of `null` when results are empty.

---

## 📦 Installation

```bash
go get github.com/shopsilo/go-paginate
```

---

## ⚡ 2-Minute Quickstart

### In Chi / net/http Handler

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/shopsilo/go-paginate"
)

type Product struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Price float64 `json:"price"`
}

func handleListProducts(w http.ResponseWriter, r *http.Request) {
	// 1. Parse & Sanitize pagination parameters from ?page=1&per_page=10&sort=price&order=asc
	allowedSorts := []string{"id", "created_at", "price", "title"}
	params := paginate.FromRequest(r).Sanitize("created_at", allowedSorts...)

	// 2. Fetch products and total count from your PostgreSQL repository
	products, totalCount := fetchProductsFromDB(r.Context(), params)

	// 3. Construct response using Go Generics
	meta := paginate.NewMeta(params, totalCount)
	response := paginate.NewPage(products, meta)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

#### Sample JSON Response:

```json
{
  "data": [
    { "id": "prod_1", "title": "Wireless Mouse", "price": 499.00 },
    { "id": "prod_2", "title": "Mechanical Keyboard", "price": 1899.00 }
  ],
  "meta": {
    "current_page": 1,
    "per_page": 10,
    "total_records": 45,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false,
    "offset": 0
  }
}
```

---

## 🗄️ Production PostgreSQL Integration (`database/sql`)

### 1. Database Table (`psql`)

```sql
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    category_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexing for high-speed pagination
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);
CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
```

### 2. Repository Query Pattern

```go
func (r *ProductRepo) List(ctx context.Context, categoryID string, p paginate.Params) (paginate.Page[Product], error) {
	// Base query
	baseWhere := "WHERE category_id = $1"
	
	// 1. Get total matching count using BuildCountSQL
	countQuery := paginate.BuildCountSQL("SELECT id FROM products " + baseWhere)
	var totalRecords int64
	if err := r.db.QueryRowContext(ctx, countQuery, categoryID).Scan(&totalRecords); err != nil {
		return paginate.Page[Product]{}, err
	}

	// 2. Generate parameterized LIMIT & OFFSET clause starting at $2
	paginationClause, paginationArgs := paginate.SQLClause(p, 2)
	// paginationClause -> ORDER BY "created_at" DESC LIMIT $2 OFFSET $3

	query := fmt.Sprintf("SELECT id, title, price FROM products %s %s", baseWhere, paginationClause)
	
	// Combine query arguments: categoryID ($1), limit ($2), offset ($3)
	allArgs := append([]any{categoryID}, paginationArgs...)

	rows, err := r.db.QueryContext(ctx, query, allArgs...)
	if err != nil {
		return paginate.Page[Product]{}, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var prod Product
		if err := rows.Scan(&prod.ID, &prod.Title, &prod.Price); err != nil {
			return paginate.Page[Product]{}, err
		}
		products = append(products, prod)
	}

	meta := paginate.NewMeta(p, totalRecords)
	return paginate.NewPage(products, meta), nil
}
```

---

## ♾️ Keyset / Cursor Pagination (For Infinite Scroll)

When querying tables with millions of rows, `OFFSET 500000` becomes slow in PostgreSQL. Keyset cursor pagination eliminates this bottleneck:

```go
func handleInfiniteFeed(w http.ResponseWriter, r *http.Request) {
	cursorParams := paginate.CursorFromRequest(r, 20) // default 20 items

	// Decodes base64 cursor token to ID
	lastID, _ := paginate.DecodeCursor(cursorParams.Cursor)

	// Fetch 21 items (limit + 1) to determine if HasMore is true
	items := fetchFeed(lastID, cursorParams.Limit + 1)

	hasMore := len(items) > cursorParams.Limit
	if hasMore {
		items = items[:cursorParams.Limit] // trim extra item
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		nextCursor = paginate.EncodeCursor(items[len(items)-1].ID)
	}

	response := paginate.NewCursorPage(items, nextCursor, hasMore, cursorParams.Limit)
	json.NewEncoder(w).Encode(response)
}
```

---

## 🧪 Testing

```bash
cd C:\Library\go-paginate
go test -v -count=1 ./...
```

# go-sqlscan 🗃️

> **High-Performance Reflection-Cached SQL Rows to Struct Scanner for Go (Go 1.23+ Generics).**  
> Go ke native `database/sql` me har row ko manually `rows.Scan(&u.ID, &u.Name, ...)` likhna tedious aur error-prone hota hai. `go-sqlscan` struct tags (`db:"column_name"`) ke according rows ko auto-scan karta hai, aur struct reflection metadata ko cache karke near-native performance deta hai.

---

## 🚀 Features

- **Generics `ScanAll[T]` & `ScanOne[T]`**: Direct `[]T` ya `*T` return karta hai aur `rows.Close()` automatically handle karta hai.
- **Reflection Metadata Cache**: Struct type ko sirf pehli baar reflect karta hai; future 10,000 rows me zero reflection lookup overhead rehta hai.
- **Embedded Struct Support**: Anonymous embedded structs (`AuditInfo`, `BaseModel`) ko transparently map karta hai.
- **Discard Extra Columns**: Query me aane wale extra columns jo struct me defined nahi hain, unhe safely ignore karta hai bina scan error diye.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"database/sql"
	"fmt"
	"go-sqlscan"
)

type User struct {
	ID    int64  `db:"id"`
	Name  string `db:"full_name"`
	Email string `db:"email"`
}

func GetUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT id, full_name, email FROM users")
	if err != nil {
		return nil, err
	}

	// Single-line type-safe scan
	return sqlscan.ScanAll[User](rows)
}

func main() {
	// users, err := GetUsers(db)
	fmt.Println("sqlscan ready for clean architecture repositories")
}
```

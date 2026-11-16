package sqlscan

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"reflect"
	"sync"
	"testing"
)

var (
	testDriverOnce sync.Once
)

type testDriver struct{}

func (d *testDriver) Open(name string) (driver.Conn, error) {
	return &testConn{}, nil
}

type testConn struct{}

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return &testStmt{}, nil
}
func (c *testConn) Close() error { return nil }
func (c *testConn) Begin() (driver.Tx, error) {
	return nil, nil
}

type testStmt struct{}

func (s *testStmt) Close() error    { return nil }
func (s *testStmt) NumInput() int   { return -1 }
func (s *testStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, nil
}
func (s *testStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &testMockRows{
		cols: []string{"id", "full_name", "email", "age", "active"},
		data: [][]driver.Value{
			{int64(1), "Arjun Sharma", "arjun@example.com", int64(28), true},
			{int64(2), "Vikram Singh", "vikram@example.com", int64(32), false},
		},
	}, nil
}

type testMockRows struct {
	cols []string
	data [][]driver.Value
	pos  int
}

func (r *testMockRows) Columns() []string { return r.cols }
func (r *testMockRows) Close() error      { return nil }
func (r *testMockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	row := r.data[r.pos]
	for i := range dest {
		dest[i] = row[i]
	}
	r.pos++
	return nil
}

type User struct {
	ID       int64  `db:"id"`
	FullName string `db:"full_name"`
	Email    string `db:"email"`
	Age      int64  `db:"age"`
	Active   bool   `db:"active"`
}

func getTestDB() *sql.DB {
	testDriverOnce.Do(func() {
		sql.Register("sqlscan_test_driver", &testDriver{})
	})
	db, _ := sql.Open("sqlscan_test_driver", "test")
	return db
}

func TestScanAll(t *testing.T) {
	db := getTestDB()
	defer db.Close()

	rows, err := db.Query("SELECT * FROM users")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	users, err := ScanAll[User](rows)
	if err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	if users[0].ID != 1 || users[0].FullName != "Arjun Sharma" || users[0].Active != true {
		t.Fatalf("user[0] mismatch: %+v", users[0])
	}
	if users[1].ID != 2 || users[1].FullName != "Vikram Singh" || users[1].Active != false {
		t.Fatalf("user[1] mismatch: %+v", users[1])
	}
}

func TestScanOne(t *testing.T) {
	db := getTestDB()
	defer db.Close()

	rows, err := db.Query("SELECT * FROM users LIMIT 1")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	user, err := ScanOne[User](rows)
	if err != nil {
		t.Fatalf("ScanOne failed: %v", err)
	}

	if user.ID != 1 || user.FullName != "Arjun Sharma" {
		t.Fatalf("ScanOne result mismatch: %+v", user)
	}
}

func TestScanRowStruct_Embedded(t *testing.T) {
	type AuditInfo struct {
		CreatedAt string `db:"created_at"`
	}
	type Order struct {
		AuditInfo
		OrderID string `db:"order_id"`
		Amount  float64 `db:"amount"`
	}

	mapping := getFieldMap(reflect.TypeOf(Order{}))
	if len(mapping) != 3 {
		t.Fatalf("expected 3 mapped fields for embedded struct, got %d", len(mapping))
	}
}

func TestScanWithColumns(t *testing.T) {
	columns := []string{"id", "full_name", "unknown_column", "age"}
	mockValues := []any{int64(42), "Rohan Gupta", "ignored_data", int64(30)}

	var u User
	err := ScanWithColumns(columns, func(dest ...any) error {
		if len(dest) != len(mockValues) {
			t.Fatalf("dest count %d != value count %d", len(dest), len(mockValues))
		}
		// Populate values
		for i := range dest {
			switch ptr := dest[i].(type) {
			case *int64:
				*ptr = mockValues[i].(int64)
			case *string:
				*ptr = mockValues[i].(string)
			case *any:
				*ptr = mockValues[i]
			}
		}
		return nil
	}, &u)

	if err != nil {
		t.Fatalf("ScanWithColumns failed: %v", err)
	}

	if u.ID != 42 || u.FullName != "Rohan Gupta" || u.Age != 30 {
		t.Fatalf("user fields mismatch: %+v", u)
	}
}

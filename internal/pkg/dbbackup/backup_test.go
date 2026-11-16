package dbbackup_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"shopMe/internal/pkg/dbbackup"
)

type mockRowScanner struct {
	val string
	err error
}

func (m *mockRowScanner) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*string); ok {
			*p = m.val
		}
	}
	return nil
}

type mockQuerier struct {
	tableData map[string]string
}

func (m *mockQuerier) QueryRow(ctx context.Context, sql string, args ...any) dbbackup.RowScanner {
	for tbl, data := range m.tableData {
		if strings.Contains(sql, tbl) {
			return &mockRowScanner{val: data}
		}
	}
	return &mockRowScanner{val: "[]"}
}

func TestAdaptQueryRow(t *testing.T) {
	// Function returning *mockRowScanner (concrete type matching RowScanner)
	customQueryRow := func(ctx context.Context, sql string, args ...any) *mockRowScanner {
		return &mockRowScanner{val: `[{"id":1}]`}
	}

	querier := dbbackup.AdaptQueryRow(customQueryRow)
	var val string
	err := querier.QueryRow(context.Background(), "SELECT 1").Scan(&val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != `[{"id":1}]` {
		t.Errorf("got %q; want %q", val, `[{"id":1}]`)
	}
}

func TestDumpTables_Success(t *testing.T) {
	mock := &mockQuerier{
		tableData: map[string]string{
			"users":    `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`,
			"products": `[{"id":10,"title":"Atta"},{"id":11,"title":"Ghee"}]`,
			"orders":   `[{"id":100,"total":450.50}]`,
		},
	}

	tables := []string{"users", "products", "orders"}
	meta := map[string]any{
		"system":  "TestSystem",
		"version": "1.0.0",
	}

	compressed, summary, err := dbbackup.DumpTablesBytes(context.Background(), mock, tables, meta)
	if err != nil {
		t.Fatalf("DumpTablesBytes failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("expected non-empty compressed output")
	}
	if summary.SizeBytes != int64(len(compressed)) {
		t.Errorf("summary.SizeBytes = %d; want %d", summary.SizeBytes, len(compressed))
	}
	if len(summary.TablesIncluded) != 3 {
		t.Errorf("summary.TablesIncluded = %d; want 3", len(summary.TablesIncluded))
	}

	// Verify the backup using VerifyBackup
	result, err := dbbackup.VerifyBackup(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("VerifyBackup failed: %v", err)
	}

	if !result.Valid {
		t.Fatal("expected backup to be valid")
	}
	if len(result.TablesFound) != 3 {
		t.Errorf("expected 3 tables found in verified archive, got %d", len(result.TablesFound))
	}
	if result.Metadata["system"] != "TestSystem" {
		t.Errorf("expected system metadata TestSystem, got %v", result.Metadata["system"])
	}
}

func TestDumpTables_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mock := &mockQuerier{tableData: map[string]string{"users": "[]"}}
	var buf bytes.Buffer
	_, err := dbbackup.DumpTables(ctx, mock, []string{"users"}, &buf, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDumpTables_NilValidation(t *testing.T) {
	_, err := dbbackup.DumpTables(context.Background(), nil, []string{"users"}, &bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected error for nil querier")
	}

	_, err = dbbackup.DumpTables(context.Background(), &mockQuerier{}, []string{"users"}, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil writer")
	}

	_, err = dbbackup.VerifyBackup(nil)
	if err == nil {
		t.Fatal("expected error for nil reader")
	}
}

func TestFormatBackupFilename(t *testing.T) {
	now := time.Date(2026, 11, 9, 8, 30, 0, 0, time.UTC)
	filename := dbbackup.FormatBackupFilename("my_db", now)
	expected := "my_db_2026_11_09_083000.json.gz"
	if filename != expected {
		t.Errorf("FormatBackupFilename = %q; want %q", filename, expected)
	}
}

func BenchmarkDumpTables(b *testing.B) {
	mock := &mockQuerier{
		tableData: map[string]string{
			"users":    `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"},{"id":3,"name":"Charlie"}]`,
			"products": `[{"id":10,"title":"Atta 5kg","price":240},{"id":11,"title":"Ghee 1L","price":650}]`,
		},
	}
	tables := []string{"users", "products"}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_, _ = dbbackup.DumpTables(ctx, mock, tables, &buf, nil)
	}
}

func BenchmarkVerifyBackup(b *testing.B) {
	mock := &mockQuerier{
		tableData: map[string]string{
			"users": `[{"id":1,"name":"Alice"}]`,
		},
	}
	compressed, _, _ := dbbackup.DumpTablesBytes(context.Background(), mock, []string{"users"}, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dbbackup.VerifyBackup(bytes.NewReader(compressed))
	}
}

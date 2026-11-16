package gocsvkit

import (
	"bytes"
	"strings"
	"testing"
)

func TestReader_BOM_Stripping(t *testing.T) {
	// UTF-8 BOM bytes \xef\xbb\xbf followed by CSV data (typical Excel export)
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte("sku,name,price\nPROD-1,MacBook Pro,1999.00\n")...)

	reader, err := NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	headers := reader.Headers()
	if len(headers) != 3 || headers[0] != "sku" {
		t.Fatalf("BOM was not stripped cleanly; headers[0] = %q", headers[0])
	}

	row, err := reader.Next()
	if err != nil {
		t.Fatalf("Next row failed: %v", err)
	}

	if row.Get("sku") != "PROD-1" {
		t.Fatalf("expected sku PROD-1, got %q", row.Get("sku"))
	}
	if row.GetFloat64("price", 0) != 1999.00 {
		t.Fatalf("expected price 1999.00, got %f", row.GetFloat64("price", 0))
	}
}

func TestReader_Chunking(t *testing.T) {
	// 10 data rows
	var sb strings.Builder
	sb.WriteString("id,value\n")
	for i := 1; i <= 10; i++ {
		sb.WriteString(strings.Repeat("a", 1) + "," + strings.Repeat("b", 1) + "\n")
	}

	reader, err := NewReader(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	var batchCount int
	var totalRows int

	// Chunk in batches of 4: batches should be 4, 4, 2 = 3 calls
	err = reader.Chunk(4, func(rows []Row) error {
		batchCount++
		totalRows += len(rows)
		return nil
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if batchCount != 3 {
		t.Fatalf("expected 3 batches, got %d", batchCount)
	}
	if totalRows != 10 {
		t.Fatalf("expected 10 total rows, got %d", totalRows)
	}
}

func TestRow_TypedAccessors(t *testing.T) {
	csvData := "id,name,qty,price,is_active\n101,Smart Watch,15,4999.50,true\n"
	reader, _ := NewReader(strings.NewReader(csvData))

	row, err := reader.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	if row.GetInt("id", 0) != 101 {
		t.Fatalf("GetInt mismatch: %d", row.GetInt("id", 0))
	}
	if row.GetInt64("qty", 0) != 15 {
		t.Fatalf("GetInt64 mismatch: %d", row.GetInt64("qty", 0))
	}
	if row.GetFloat64("price", 0) != 4999.50 {
		t.Fatalf("GetFloat64 mismatch: %f", row.GetFloat64("price", 0))
	}
	if !row.GetBool("is_active", false) {
		t.Fatal("GetBool expected true")
	}

	m := row.ToMap()
	if m["name"] != "Smart Watch" {
		t.Fatalf("ToMap mismatch: %v", m)
	}
}

func TestWriter_WriteMap(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	err := writer.WriteHeader("id", "product", "price")
	if err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}

	err = writer.WriteMap(map[string]string{
		"product": "Wireless Keyboard",
		"price":   "1299.00",
		"id":      "KB-901",
	})
	if err != nil {
		t.Fatalf("WriteMap failed: %v", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	output := buf.String()
	expected := "id,product,price\nKB-901,Wireless Keyboard,1299.00\n"
	if output != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestReader_Empty(t *testing.T) {
	_, err := NewReader(strings.NewReader(""))
	if err != ErrEmptyCSV {
		t.Fatalf("expected ErrEmptyCSV on empty input, got %v", err)
	}
}

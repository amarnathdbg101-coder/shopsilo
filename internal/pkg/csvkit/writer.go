package gocsvkit

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// WriterConfig tunes CSV generation.
type WriterConfig struct {
	Comma   rune
	UseCRLF bool
}

// Writer wraps standard csv.Writer with map-based column alignment and auto-flushing.
type Writer struct {
	csvWriter *csv.Writer
	headers   []string
	headerMap map[string]int
}

// NewWriter constructs a streaming CSV writer.
func NewWriter(w io.Writer, cfg ...WriterConfig) *Writer {
	cw := csv.NewWriter(w)
	if len(cfg) > 0 {
		if cfg[0].Comma != 0 {
			cw.Comma = cfg[0].Comma
		}
		cw.UseCRLF = cfg[0].UseCRLF
	}

	return &Writer{
		csvWriter: cw,
		headerMap: make(map[string]int),
	}
}

// WriteHeader writes the CSV header row and stores column ordering for WriteMap calls.
func (w *Writer) WriteHeader(columns ...string) error {
	w.headers = columns
	w.headerMap = make(map[string]int, len(columns))
	for i, col := range columns {
		w.headerMap[strings.ToLower(strings.TrimSpace(col))] = i
	}
	return w.csvWriter.Write(columns)
}

// WriteRow writes a single row of string values.
func (w *Writer) WriteRow(values ...string) error {
	return w.csvWriter.Write(values)
}

// WriteRows writes multiple rows of string values.
func (w *Writer) WriteRows(rows [][]string) error {
	return w.csvWriter.WriteAll(rows)
}

// WriteMap writes a row matching the header columns previously registered with WriteHeader.
func (w *Writer) WriteMap(row map[string]string) error {
	if len(w.headers) == 0 {
		return fmt.Errorf("csvkit: WriteHeader must be called before WriteMap")
	}

	values := make([]string, len(w.headers))
	for col, val := range row {
		if idx, ok := w.headerMap[strings.ToLower(strings.TrimSpace(col))]; ok {
			values[idx] = val
		}
	}

	return w.csvWriter.Write(values)
}

// Flush flushes buffered CSV data to the underlying writer and returns any error.
func (w *Writer) Flush() error {
	w.csvWriter.Flush()
	return w.csvWriter.Error()
}

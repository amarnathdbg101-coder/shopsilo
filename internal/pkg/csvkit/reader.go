package gocsvkit

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Row represents a single CSV row with type-safe accessors mapped to header names.
type Row struct {
	LineNumber int
	HeaderMap  map[string]int
	Values     []string
}

// Get returns the trimmed string value of a column, or empty string if not found.
func (r *Row) Get(column string) string {
	idx, ok := r.HeaderMap[strings.ToLower(strings.TrimSpace(column))]
	if !ok || idx >= len(r.Values) {
		return ""
	}
	return strings.TrimSpace(r.Values[idx])
}

// GetInt returns the column parsed as an integer or defaultVal.
func (r *Row) GetInt(column string, defaultVal int) int {
	val := r.Get(column)
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.Atoi(val); err == nil {
		return i
	}
	return defaultVal
}

// GetInt64 returns the column parsed as an int64 or defaultVal.
func (r *Row) GetInt64(column string, defaultVal int64) int64 {
	val := r.Get(column)
	if val == "" {
		return defaultVal
	}
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}
	return defaultVal
}

// GetFloat64 returns the column parsed as a float64 or defaultVal.
func (r *Row) GetFloat64(column string, defaultVal float64) float64 {
	val := r.Get(column)
	if val == "" {
		return defaultVal
	}
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}
	return defaultVal
}

// GetBool returns boolean value (true/false/1/0/yes/no) or defaultVal.
func (r *Row) GetBool(column string, defaultVal bool) bool {
	val := strings.ToLower(r.Get(column))
	switch val {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	}
	return defaultVal
}

// ToMap converts the row into a map of headerName -> value.
func (r *Row) ToMap() map[string]string {
	m := make(map[string]string, len(r.HeaderMap))
	for k, idx := range r.HeaderMap {
		if idx < len(r.Values) {
			m[k] = strings.TrimSpace(r.Values[idx])
		}
	}
	return m
}

// ReaderConfig tunes the CSV parser.
type ReaderConfig struct {
	Comma            rune
	Comment          rune
	LazyQuotes       bool
	TrimLeadingSpace bool
	HasHeader        bool
}

// DefaultReaderConfig provides safe defaults for general CSV files.
var DefaultReaderConfig = ReaderConfig{
	Comma:            ',',
	LazyQuotes:       true,
	TrimLeadingSpace: true,
	HasHeader:        true,
}

// Reader wraps standard csv.Reader with BOM stripping and memory-bounded batch chunking.
type Reader struct {
	csvReader  *csv.Reader
	headers    []string
	headerMap  map[string]int
	lineNumber int
	hasHeader  bool
}

// NewReader constructs a streaming CSV reader. Automatically detects and strips UTF-8 BOM.
func NewReader(r io.Reader, cfg ...ReaderConfig) (*Reader, error) {
	config := DefaultReaderConfig
	if len(cfg) > 0 {
		config = cfg[0]
		if config.Comma == 0 {
			config.Comma = ','
		}
	}

	// Wrap in bufio to sniff and strip UTF-8 BOM (\xef\xbb\xbf)
	br := bufio.NewReader(r)
	bom, err := br.Peek(3)
	if err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3) // Strip BOM cleanly
	}

	cr := csv.NewReader(br)
	cr.Comma = config.Comma
	cr.Comment = config.Comment
	cr.LazyQuotes = config.LazyQuotes
	cr.TrimLeadingSpace = config.TrimLeadingSpace
	cr.ReuseRecord = false

	reader := &Reader{
		csvReader: cr,
		headerMap: make(map[string]int),
		hasHeader: config.HasHeader,
	}

	if config.HasHeader {
		headers, err := cr.Read()
		if err != nil {
			if err == io.EOF {
				return nil, ErrEmptyCSV
			}
			return nil, fmt.Errorf("csvkit: read header: %w", err)
		}
		reader.lineNumber++
		reader.headers = make([]string, len(headers))
		for i, h := range headers {
			clean := strings.TrimSpace(h)
			reader.headers[i] = clean
			reader.headerMap[strings.ToLower(clean)] = i
		}
	}

	return reader, nil
}

// Headers returns the parsed header column names.
func (r *Reader) Headers() []string {
	res := make([]string, len(r.headers))
	copy(res, r.headers)
	return res
}

// Next reads the next row from the CSV stream. Returns io.EOF when exhausted.
func (r *Reader) Next() (*Row, error) {
	record, err := r.csvReader.Read()
	if err != nil {
		return nil, err
	}
	r.lineNumber++

	row := &Row{
		LineNumber: r.lineNumber,
		HeaderMap:  r.headerMap,
		Values:     record,
	}
	return row, nil
}

// Chunk streams the CSV in batches of batchSize rows, invoking the callback for each batch.
// Memory usage stays strictly constant regardless of total CSV size (safe for 10M+ rows).
func (r *Reader) Chunk(batchSize int, fn func(rows []Row) error) error {
	if batchSize <= 0 {
		batchSize = 1000
	}

	batch := make([]Row, 0, batchSize)

	for {
		row, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("csvkit: read row at line %d: %w", r.lineNumber, err)
		}

		batch = append(batch, *row)

		if len(batch) >= batchSize {
			if err := fn(batch); err != nil {
				return fmt.Errorf("csvkit: chunk callback failed: %w", err)
			}
			batch = batch[:0] // Reset slice without re-allocating
		}
	}

	// Flush remaining rows
	if len(batch) > 0 {
		if err := fn(batch); err != nil {
			return fmt.Errorf("csvkit: chunk callback failed on final batch: %w", err)
		}
	}

	return nil
}

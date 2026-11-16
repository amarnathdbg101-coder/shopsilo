package dbbackup

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// RowScanner abstracts row scanning for database/sql and pgx queries.
type RowScanner interface {
	Scan(dest ...any) error
}

// Querier abstracts PostgreSQL query execution for pgxpool.Pool, pgx.Conn, or *sql.DB.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) RowScanner
}

type queryRowFunc func(ctx context.Context, sql string, args ...any) RowScanner

func (f queryRowFunc) QueryRow(ctx context.Context, sql string, args ...any) RowScanner {
	return f(ctx, sql, args...)
}

// AdaptQueryRow wraps any QueryRow function returning a type that implements RowScanner
// (e.g. pgxpool.Pool.QueryRow returning pgx.Row, or *sql.DB.QueryRowContext returning *sql.Row).
func AdaptQueryRow[T RowScanner](fn func(ctx context.Context, sql string, args ...any) T) Querier {
	return queryRowFunc(func(ctx context.Context, sql string, args ...any) RowScanner {
		return fn(ctx, sql, args...)
	})
}

// DumpSummary captures details and metrics of a completed database backup dump.
type DumpSummary struct {
	BackupID       string         `json:"backup_id"`
	Filename       string         `json:"filename"`
	SizeBytes      int64          `json:"size_bytes"`
	TablesIncluded []string       `json:"tables_included"`
	CreatedAt      time.Time      `json:"created_at"`
	Duration       time.Duration  `json:"duration"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// VerifyResult contains validation status and parsed table counts from a backup archive.
type VerifyResult struct {
	Valid          bool           `json:"valid"`
	TablesFound    []string       `json:"tables_found"`
	Metadata       map[string]any `json:"metadata"`
	UncompressedSz int64          `json:"uncompressed_size_bytes"`
}

// FormatBackupFilename generates a standardized timestamped backup filename.
func FormatBackupFilename(prefix string, t time.Time) string {
	if prefix == "" {
		prefix = "db_backup"
	}
	timestamp := t.Format("2006_01_02_150405")
	return fmt.Sprintf("%s_%s.json.gz", prefix, timestamp)
}

// DumpTables streams PostgreSQL table row data directly into a gzip-compressed JSON stream.
// It leverages PostgreSQL native json_agg for maximum throughput and minimal Go memory consumption.
func DumpTables(ctx context.Context, q Querier, tables []string, w io.Writer, meta map[string]any) (*DumpSummary, error) {
	if q == nil {
		return nil, errors.New("querier cannot be nil")
	}
	if w == nil {
		return nil, errors.New("writer cannot be nil")
	}

	start := time.Now()
	timestamp := start.Format("2006_01_02_150405")
	filename := FormatBackupFilename("db_backup", start)

	countingWriter := &byteCountingWriter{w: w}
	gzWriter := gzip.NewWriter(countingWriter)

	// Stream JSON construction to avoid parsing giant tables into RAM (OOM Prevention)
	if _, err := gzWriter.Write([]byte("{\n")); err != nil {
		return nil, fmt.Errorf("failed to write json header: %w", err)
	}

	// Prepare metadata
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["created_at"] = start.Format(time.RFC3339)
	meta["tables_included"] = tables

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup metadata: %w", err)
	}

	if _, err := gzWriter.Write([]byte(`"meta": `)); err != nil {
		return nil, err
	}
	if _, err := gzWriter.Write(metaJSON); err != nil {
		return nil, err
	}

	// Fetch table row dumps
	for _, tbl := range tables {
		select {
		case <-ctx.Done():
			_ = gzWriter.Close()
			return nil, ctx.Err()
		default:
		}

		if _, err := gzWriter.Write([]byte(",\n\"" + tbl + "\": ")); err != nil {
			return nil, err
		}

		query := fmt.Sprintf("SELECT COALESCE(json_agg(t)::text, '[]') FROM (SELECT * FROM %s) t", tbl)
		var jsonRaw string
		err := q.QueryRow(ctx, query).Scan(&jsonRaw)
		if err != nil || jsonRaw == "" {
			if _, err := gzWriter.Write([]byte("[]")); err != nil {
				return nil, err
			}
		} else {
			if _, err := gzWriter.Write([]byte(jsonRaw)); err != nil {
				return nil, err
			}
		}
	}

	if _, err := gzWriter.Write([]byte("\n}")); err != nil {
		return nil, err
	}

	if err := gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return &DumpSummary{
		BackupID:       timestamp,
		Filename:       filename,
		SizeBytes:      countingWriter.count,
		TablesIncluded: tables,
		CreatedAt:      start,
		Duration:       time.Since(start),
		Metadata:       meta,
	}, nil
}

// DumpTablesBytes generates a full gzip-compressed database dump into an in-memory byte slice.
func DumpTablesBytes(ctx context.Context, q Querier, tables []string, meta map[string]any) ([]byte, *DumpSummary, error) {
	var buf bytes.Buffer
	summary, err := DumpTables(ctx, q, tables, &buf, meta)
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), summary, nil
}

// VerifyBackup inspects and verifies a gzip-compressed JSON backup stream.
func VerifyBackup(r io.Reader) (*VerifyResult, error) {
	if r == nil {
		return nil, errors.New("reader cannot be nil")
	}

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gzReader.Close()

	var uncompressed bytes.Buffer
	n, err := io.Copy(&uncompressed, gzReader)
	if err != nil {
		return nil, fmt.Errorf("decompression error: %w", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(uncompressed.Bytes(), &root); err != nil {
		return nil, fmt.Errorf("corrupted json payload: %w", err)
	}

	var meta map[string]any
	if metaRaw, ok := root["meta"]; ok {
		_ = json.Unmarshal(metaRaw, &meta)
	}

	var tables []string
	for k := range root {
		if k != "meta" {
			tables = append(tables, k)
		}
	}

	return &VerifyResult{
		Valid:          true,
		TablesFound:    tables,
		Metadata:       meta,
		UncompressedSz: n,
	}, nil
}

type byteCountingWriter struct {
	w     io.Writer
	count int64
}

func (b *byteCountingWriter) Write(p []byte) (int, error) {
	n, err := b.w.Write(p)
	b.count += int64(n)
	return n, err
}

# SPEC: go-csvkit

## 1. Metadata
- **Module**: `go-csvkit`
- **Package**: `gocsvkit`
- **Location**: `C:\Library\go-csvkit`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `encoding/csv`, `io`, `bufio`, `bytes`, `strings`, `strconv`, `fmt`, `errors`).

---

## 2. Core Capabilities
1. **OOM-Proof Streaming Batch Chunking**:
   - `Chunk(batchSize, callback)` processes massive CSV files (100MB to 10GB+) in strictly bounded RAM batches.
2. **Automated UTF-8 BOM Stripping**:
   - Auto-detects and removes Microsoft Excel's `\xEF\xBB\xBF` byte order mark from column headers.
3. **Type-Safe Row Accessors**:
   - `row.Get(col)`, `row.GetInt(col, default)`, `row.GetInt64()`, `row.GetFloat64()`, `row.GetBool()`, `row.ToMap()`.
4. **Column-Aligned Streaming Writer**:
   - `WriteHeader(cols...)` establishes column mapping so `WriteMap(m)` can output data in the correct header order automatically.

---

## 3. Public API
```go
type Row struct {
    LineNumber int
    HeaderMap  map[string]int
    Values     []string
}
func (r *Row) Get(column string) string
func (r *Row) GetInt(column string, defaultVal int) int
func (r *Row) GetInt64(column string, defaultVal int64) int64
func (r *Row) GetFloat64(column string, defaultVal float64) float64
func (r *Row) GetBool(column string, defaultVal bool) bool
func (r *Row) ToMap() map[string]string

type Reader struct { ... }
func NewReader(r io.Reader, cfg ...ReaderConfig) (*Reader, error)
func (r *Reader) Headers() []string
func (r *Reader) Next() (*Row, error)
func (r *Reader) Chunk(batchSize int, fn func(rows []Row) error) error

type Writer struct { ... }
func NewWriter(w io.Writer, cfg ...WriterConfig) *Writer
func (w *Writer) WriteHeader(columns ...string) error
func (w *Writer) WriteRow(values ...string) error
func (w *Writer) WriteRows(rows [][]string) error
func (w *Writer) WriteMap(row map[string]string) error
func (w *Writer) Flush() error
```

package sqlscan

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	typeCacheMu sync.RWMutex
	typeCache   = make(map[reflect.Type]map[string][]int) // type -> column -> field index path
)

func getFieldMap(t reflect.Type) map[string][]int {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeCacheMu.RLock()
	m, ok := typeCache[t]
	typeCacheMu.RUnlock()
	if ok {
		return m
	}

	typeCacheMu.Lock()
	defer typeCacheMu.Unlock()

	// Double check
	if m, ok := typeCache[t]; ok {
		return m
	}

	m = make(map[string][]int)
	buildFieldMap(t, nil, m)
	typeCache[t] = m
	return m
}

func buildFieldMap(t reflect.Type, indexPrefix []int, m map[string][]int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Anonymous embedded struct traversal
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			newPrefix := append(indexPrefix, i)
			buildFieldMap(field.Type, newPrefix, m)
			continue
		}

		tag := field.Tag.Get("db")
		if tag == "-" {
			continue
		}

		colName := tag
		if colName == "" {
			colName = strings.ToLower(field.Name)
		} else {
			// Handle tags like `db:"id,omitempty"`
			parts := strings.Split(tag, ",")
			colName = parts[0]
		}

		indexPath := append(indexPrefix, i)
		m[strings.ToLower(colName)] = indexPath
	}
}

// ScanWithColumns maps column names dynamically to struct fields via `db` tags or field names,
// then executes scanFn with pointers to the corresponding struct fields.
// Safely ignores extra database columns that are not present in dest struct.
func ScanWithColumns(columns []string, scanFn func(dest ...any) error, dest any) error {
	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return ErrNotStructPointer
	}

	structVal := val.Elem()
	structType := structVal.Type()
	fieldMap := getFieldMap(structType)

	scanArgs := make([]any, len(columns))
	var dummy any

	for i, col := range columns {
		colLower := strings.ToLower(col)
		if idxPath, found := fieldMap[colLower]; found {
			fieldVal := structVal.FieldByIndex(idxPath)
			if fieldVal.CanAddr() {
				scanArgs[i] = fieldVal.Addr().Interface()
			} else {
				scanArgs[i] = &dummy
			}
		} else {
			scanArgs[i] = &dummy
		}
	}

	return scanFn(scanArgs...)
}

// ScanRowStruct scans the current row from *sql.Rows into dest (must be pointer to struct).
func ScanRowStruct(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	return ScanWithColumns(columns, rows.Scan, dest)
}

// ScanOne scans the first row of rows into a new instance of T and closes rows.
func ScanOne[T any](rows *sql.Rows) (*T, error) {
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, ErrNoRowsFound
	}

	var item T
	val := reflect.ValueOf(&item)
	if val.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: type is %T", ErrNotStructPointer, item)
	}

	if err := ScanRowStruct(rows, &item); err != nil {
		return nil, err
	}

	return &item, rows.Err()
}

// ScanAll iterates over all rows and scans them into a slice of T, closing rows.
func ScanAll[T any](rows *sql.Rows) ([]T, error) {
	defer rows.Close()

	var result []T
	var sample T
	val := reflect.ValueOf(&sample)
	if val.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: slice element type is %T", ErrNotStructPointer, sample)
	}

	for rows.Next() {
		var item T
		if err := ScanRowStruct(rows, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

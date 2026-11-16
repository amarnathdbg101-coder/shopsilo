package sqlscan

import "errors"

var (
	// ErrNotStructPointer is returned when dest is not a pointer to a struct.
	ErrNotStructPointer = errors.New("sqlscan: destination must be a pointer to a struct")
	// ErrNotSlicePointer is returned when dest is not a pointer to a slice of structs.
	ErrNotSlicePointer = errors.New("sqlscan: destination must be a pointer to a slice")
	// ErrNoRowsFound is returned when ScanOne finds 0 rows.
	ErrNoRowsFound = errors.New("sqlscan: no rows found in result set")
)

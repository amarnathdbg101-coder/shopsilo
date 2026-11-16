package gocsvkit

import "errors"

var (
	// ErrEmptyCSV occurs when the CSV input contains no rows or header.
	ErrEmptyCSV = errors.New("csvkit: empty csv input")

	// ErrHeaderMismatch occurs when row columns count does not match header count.
	ErrHeaderMismatch = errors.New("csvkit: row length does not match header length")

	// ErrColumnNotFound occurs when requesting an unknown column from a Row.
	ErrColumnNotFound = errors.New("csvkit: column not found in header")
)

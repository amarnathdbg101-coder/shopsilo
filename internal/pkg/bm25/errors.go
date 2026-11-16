package bm25

import "errors"

var (
	// ErrEmptyDocument is returned when attempting to index a document with an empty ID or content.
	ErrEmptyDocument = errors.New("bm25: document ID and content cannot be empty")

	// ErrDocumentNotFound is returned when attempting to delete a non-existent document.
	ErrDocumentNotFound = errors.New("bm25: document not found in index")

	// ErrEmptyQuery is returned when a search query contains no searchable tokens.
	ErrEmptyQuery = errors.New("bm25: search query contains no tokens")
)

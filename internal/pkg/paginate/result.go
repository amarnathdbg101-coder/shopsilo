package paginate

// Page is a generic container holding paginated slice data and calculation metadata.
// Go Generics (Go 1.18+) use karke kisi bhi struct (Product, User, Order) ke sath strongly typed rehta hai.
type Page[T any] struct {
	Data []T  `json:"data"`
	Meta Meta `json:"meta"`
}

// CursorPage is a generic container for cursor-based (keyset) paginated responses.
type CursorPage[T any] struct {
	Data []T        `json:"data"`
	Meta CursorMeta `json:"meta"`
}

// NewPage constructs a type-safe paginated result.
// Agar data slice nil ho, toh empty slice []T{} banata hai taaki JSON me "null" ke bajaye "[]" serialize ho.
func NewPage[T any](data []T, meta Meta) Page[T] {
	if data == nil {
		data = make([]T, 0)
	}
	return Page[T]{
		Data: data,
		Meta: meta,
	}
}

// NewCursorPage constructs a type-safe cursor paginated result.
func NewCursorPage[T any](data []T, nextCursor string, hasMore bool, limit int) CursorPage[T] {
	if data == nil {
		data = make([]T, 0)
	}
	return CursorPage[T]{
		Data: data,
		Meta: CursorMeta{
			NextCursor: nextCursor,
			HasMore:    hasMore,
			Limit:      limit,
		},
	}
}

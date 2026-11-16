// Package paginate provides clean, high-performance, and SQL-injection-safe
// offset and cursor pagination for Go backends and PostgreSQL databases.
package paginate

// PaginateSlice performs in-memory pagination on an arbitrary slice.
// Jab data cache se ya external microservice se pura array milta hai,
// toh ye function memory me hi slice karke same standard Page[T] JSON structure generate karta hai.
func PaginateSlice[T any](items []T, params Params) Page[T] {
	totalRecords := int64(len(items))
	meta := NewMeta(params, totalRecords)

	if totalRecords == 0 {
		return NewPage([]T{}, meta)
	}

	offset := params.Offset()
	if offset >= len(items) {
		return NewPage([]T{}, meta)
	}

	end := offset + params.Limit()
	if end > len(items) {
		end = len(items)
	}

	return NewPage(items[offset:end], meta)
}

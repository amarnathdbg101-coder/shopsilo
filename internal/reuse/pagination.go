package reuse

import (
	"net/http"

	"github.com/amarnathdbg101-coder/go/go-paginate"
)

// PaginationParams aliases the standardized pagination parameters.
type PaginationParams = paginate.Params

// PaginationMeta aliases the standardized pagination metadata.
type PaginationMeta = paginate.Meta

// ParsePagination extracts, validates, and bounds pagination parameters from an HTTP request.
// Supports both ?page= & ?limit= / ?per_page= query params with automatic DOS protection (max 100).
func ParsePagination(r *http.Request, defaultLimit int) paginate.Params {
	if defaultLimit <= 0 {
		defaultLimit = 20
	}
	return paginate.FromRequest(r,
		paginate.WithDefaults(1, defaultLimit, "created_at", "DESC"),
		paginate.WithMaxPerPage(100),
	)
}

// BuildPaginationMeta constructs standardized pagination metadata (total, pages, has_next, etc.).
func BuildPaginationMeta(params paginate.Params, totalRecords int64) paginate.Meta {
	return paginate.NewMeta(params, totalRecords)
}

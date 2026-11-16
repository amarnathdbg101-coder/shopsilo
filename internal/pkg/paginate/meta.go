package paginate

// Meta holds calculated pagination metadata returned to API clients.
// Flutter ya React frontend is metadata ko use karke pagination controls ya page numbers render karta hai.
type Meta struct {
	CurrentPage  int   `json:"current_page"`  // Currently requested page (1-based)
	PerPage      int   `json:"per_page"`      // Number of records per page
	TotalRecords int64 `json:"total_records"` // Total matching records in database
	TotalPages   int   `json:"total_pages"`   // Total number of pages available
	HasNext      bool  `json:"has_next"`      // True if there is a next page
	HasPrev      bool  `json:"has_prev"`      // True if there is a previous page
	Offset       int   `json:"offset"`        // Zero-based SQL offset used for query
}

// NewMeta calculates complete pagination metadata based on requested parameters and total DB count.
// Pure integer arithmetic use kiya gaya hai taaki floating point precision errors na hon.
func NewMeta(params Params, totalRecords int64) Meta {
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = DefaultPerPage
	}

	page := params.Page
	if page <= 0 {
		page = DefaultPage
	}

	var totalPages int
	if totalRecords > 0 {
		// Integer ceil division: (totalRecords + perPage - 1) / perPage
		totalPages = int((totalRecords + int64(perPage) - 1) / int64(perPage))
	} else {
		totalPages = 0
	}

	hasNext := page < totalPages
	hasPrev := page > 1 && totalPages > 0

	return Meta{
		CurrentPage:  page,
		PerPage:      perPage,
		TotalRecords: totalRecords,
		TotalPages:   totalPages,
		HasNext:      hasNext,
		HasPrev:      hasPrev,
		Offset:       params.Offset(),
	}
}

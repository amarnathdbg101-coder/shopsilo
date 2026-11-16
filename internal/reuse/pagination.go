package reuse

import (
	"net/http"
	"strconv"
)

type PaginationParams struct {
	Page    int
	Limit   int
	PerPage int
	OrderBy string
	Order   string
}

func (p PaginationParams) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	limit := p.Limit
	if limit <= 0 {
		limit = p.PerPage
	}
	return (p.Page - 1) * limit
}

type PaginationMeta struct {
	CurrentPage  int   `json:"current_page"`
	PerPage      int   `json:"per_page"`
	TotalRecords int64 `json:"total_records"`
	Total        int64 `json:"total"`
	TotalPages   int   `json:"total_pages"`
	HasNext      bool  `json:"has_next"`
	HasPrev      bool  `json:"has_prev"`
	HasPrevious  bool  `json:"has_previous"`
}

// ParsePagination extracts, validates, and bounds pagination parameters from an HTTP request.
// Supports both ?page= & ?limit= / ?per_page= query params with automatic DOS protection (max 100).
func ParsePagination(r *http.Request, defaultLimit int) PaginationParams {
	if defaultLimit <= 0 {
		defaultLimit = 20
	}
	page := 1
	if pStr := r.URL.Query().Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := defaultLimit
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	} else if lStr := r.URL.Query().Get("per_page"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}
	return PaginationParams{
		Page:    page,
		Limit:   limit,
		PerPage: limit,
		OrderBy: "created_at",
		Order:   "DESC",
	}
}

// BuildPaginationMeta constructs standardized pagination metadata (total, pages, has_next, etc.).
func BuildPaginationMeta(params PaginationParams, totalRecords int64) PaginationMeta {
	totalPages := int((totalRecords + int64(params.Limit) - 1) / int64(params.Limit))
	if totalPages < 1 {
		totalPages = 1
	}
	return PaginationMeta{
		CurrentPage:  params.Page,
		PerPage:      params.Limit,
		TotalRecords: totalRecords,
		Total:        totalRecords,
		TotalPages:   totalPages,
		HasNext:      params.Page < totalPages,
		HasPrev:      params.Page > 1,
		HasPrevious:  params.Page > 1,
	}
}

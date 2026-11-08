package reuse_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"shopMe/internal/reuse"
)

func TestParsePagination(t *testing.T) {
	t.Run("Default values when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/sales", nil)
		p := reuse.ParsePagination(req, 20)

		if p.Page != 1 {
			t.Errorf("expected page 1, got %d", p.Page)
		}
		if p.PerPage != 20 {
			t.Errorf("expected per_page 20, got %d", p.PerPage)
		}
		if p.Offset() != 0 {
			t.Errorf("expected offset 0, got %d", p.Offset())
		}
	})

	t.Run("Parses page and limit query params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/sales?page=3&limit=25", nil)
		p := reuse.ParsePagination(req, 20)

		if p.Page != 3 {
			t.Errorf("expected page 3, got %d", p.Page)
		}
		if p.PerPage != 25 {
			t.Errorf("expected per_page 25, got %d", p.PerPage)
		}
		if p.Offset() != 50 {
			t.Errorf("expected offset 50 (page 3 with limit 25), got %d", p.Offset())
		}
	})

	t.Run("Caps malicious limit at 100", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/sales?page=1&limit=999999", nil)
		p := reuse.ParsePagination(req, 20)

		if p.PerPage != 100 {
			t.Errorf("expected capped limit 100, got %d", p.PerPage)
		}
	})

	t.Run("Builds Pagination Metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/shops/me/sales?page=2&limit=10", nil)
		p := reuse.ParsePagination(req, 10)
		meta := reuse.BuildPaginationMeta(p, 45)

		if meta.TotalRecords != 45 {
			t.Errorf("expected 45 total records, got %d", meta.TotalRecords)
		}
		if meta.TotalPages != 5 {
			t.Errorf("expected 5 total pages, got %d", meta.TotalPages)
		}
		if !meta.HasNext {
			t.Errorf("expected HasNext to be true on page 2 of 5")
		}
		if !meta.HasPrev {
			t.Errorf("expected HasPrev to be true on page 2 of 5")
		}
	})
}

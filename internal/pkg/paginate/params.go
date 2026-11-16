package paginate

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Default pagination bounds.
const (
	DefaultPage    = 1
	DefaultPerPage = 10
	MaxPerPage     = 100
)

// Params encapsulates incoming pagination query parameters.
// API request se parse hokar ye struct database layer me pass kiya jata hai.
type Params struct {
	Page    int    `json:"page"`     // 1-indexed page number (e.g. 1, 2, 3...)
	PerPage int    `json:"per_page"` // Number of records per page (e.g. 10, 20)
	Sort    string `json:"sort"`     // Column name to order by (e.g. "created_at", "price")
	Order   string `json:"order"`    // Sort direction: "ASC" or "DESC"
}

// Config allows overriding default pagination constraints.
type Config struct {
	DefaultPage    int
	DefaultPerPage int
	MaxPerPage     int
	DefaultSort    string
	DefaultOrder   string
}

// DefaultConfig provides standard sensible pagination limits.
func DefaultConfig() Config {
	return Config{
		DefaultPage:    DefaultPage,
		DefaultPerPage: DefaultPerPage,
		MaxPerPage:     MaxPerPage,
		DefaultSort:    "id",
		DefaultOrder:   "DESC",
	}
}

// Option is a functional option for customizing pagination parsing.
type Option func(*Config)

// WithDefaults overrides the default per-page and sort values.
func WithDefaults(defaultPage, defaultPerPage int, defaultSort, defaultOrder string) Option {
	return func(c *Config) {
		if defaultPage > 0 {
			c.DefaultPage = defaultPage
		}
		if defaultPerPage > 0 {
			c.DefaultPerPage = defaultPerPage
		}
		if defaultSort != "" {
			c.DefaultSort = defaultSort
		}
		if defaultOrder != "" {
			c.DefaultOrder = strings.ToUpper(defaultOrder)
		}
	}
}

// WithMaxPerPage configures the upper bound for records per page (preventing Denial of Service).
func WithMaxPerPage(max int) Option {
	return func(c *Config) {
		if max > 0 {
			c.MaxPerPage = max
		}
	}
}

// FromRequest extracts and sanitizes pagination parameters directly from an HTTP request.
// Works seamlessly with Chi, standard net/http, or any standard HTTP handler.
func FromRequest(r *http.Request, opts ...Option) Params {
	return FromValues(r.URL.Query(), opts...)
}

// FromValues parses pagination parameters from url.Values query map.
func FromValues(values url.Values, opts ...Option) Params {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// 1. Parse Page (supports: ?page=)
	page := cfg.DefaultPage
	if val := values.Get("page"); val != "" {
		if p, err := strconv.Atoi(val); err == nil && p > 0 {
			page = p
		}
	}

	// 2. Parse PerPage (supports: ?per_page= or ?limit=)
	perPage := cfg.DefaultPerPage
	perPageStr := values.Get("per_page")
	if perPageStr == "" {
		perPageStr = values.Get("limit")
	}
	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			if pp > cfg.MaxPerPage {
				perPage = cfg.MaxPerPage
			} else {
				perPage = pp
			}
		}
	}

	// 3. Parse Sort (supports: ?sort= or ?sort_by=)
	sort := cfg.DefaultSort
	sortStr := values.Get("sort")
	if sortStr == "" {
		sortStr = values.Get("sort_by")
	}
	if sortStr != "" {
		sort = strings.TrimSpace(sortStr)
	}

	// 4. Parse Order (supports: ?order= or ?dir=)
	order := cfg.DefaultOrder
	orderStr := values.Get("order")
	if orderStr == "" {
		orderStr = values.Get("dir")
	}
	if orderStr != "" {
		upper := strings.ToUpper(strings.TrimSpace(orderStr))
		if upper == "ASC" || upper == "DESC" {
			order = upper
		}
	}

	return Params{
		Page:    page,
		PerPage: perPage,
		Sort:    sort,
		Order:   order,
	}
}

// Offset returns the zero-based SQL OFFSET calculated as: (Page - 1) * PerPage.
func (p Params) Offset() int {
	if p.Page <= 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// Limit returns PerPage count for SQL LIMIT.
func (p Params) Limit() int {
	if p.PerPage <= 0 {
		return DefaultPerPage
	}
	return p.PerPage
}
